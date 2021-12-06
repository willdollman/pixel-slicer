package mediaprocessor

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"math"
	"os"
	"time"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/hashicorp/go-multierror"
	"github.com/nickalie/go-webpbin"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
	"github.com/willdollman/pixel-slicer/internal/s3"
	"github.com/xfrr/goffmpeg/transcoder"
)

// ImageOutputConfig defines an output image configuration
type ImageOutputConfig struct {
	width   int
	quality int
}

// TODO: These currently aren't used for anything
type ImageProcessor interface {
	ResizeImage(inputFile string, outputDir string, width int, quality int, format string) (fileName string, err error)
}
type VideoProcessor interface {
	ResizeVideo(inputFile string, outputDir string, width int, quality int, format string) (fileName string, err error)
}

type MediaJob struct {
	MediaConfig *MediaConfig // TODO: Might be a bit verbose
	FSConfig    *FSConfig    // TODO: Might be a bit verbose
	S3Client    *s3.S3Client
	InputFile   *pixelio.InputFile
}

// ProcessVideo processes a single video
func (m *MediaJob) ProcessVideo() (filenames []string, allErrors error) {
	fmt.Println("Transcoding video, this may take a while...")

	if err := pixelio.EnsureOutputDirExists(m.InputFile.Subdir); err != nil {
		log.Fatal("Unable to prepare output dir:", err)
	}

	for _, videoConfig := range m.MediaConfig.VideoConfigurations {
		var err error
		// Depending on the requested media type, either transcode video or generate a thumbnail
		if videoConfig.FileType.GetMediaType() == Video {
			encodeStartTime := time.Now()
			err = videoTranscode(videoConfig, m.InputFile)
			fmt.Printf("Encoding took %.2fs\n", time.Now().Sub(encodeStartTime).Seconds())
		} else if videoConfig.FileType.GetMediaType() == Image {
			err = videoThumbnail(videoConfig, m.InputFile)
		} else {
			err = fmt.Errorf("Configuration contains unknown media type: %s", videoConfig.FileType)
		}

		if err == nil {
			outputFilepath := pixelio.GetFileOutputPath(m.InputFile, videoConfig.OutputFileSuffix(false))
			filenames = append(filenames, outputFilepath)
		} else {
			allErrors = multierror.Append(allErrors, err)
		}
	}
	return
}

func videoThumbnail(videoConfig VideoConfiguration, inputFile *pixelio.InputFile) (err error) {
	outputFilepath := pixelio.GetFileOutputPath(inputFile, videoConfig.OutputFileSuffix(false))

	t := new(transcoder.Transcoder)
	if err = t.Initialize(inputFile.Path, outputFilepath); err != nil {
		log.Println("Error initialising video transcoder:", err)
		return
	}

	// Ensure maxWidth is even - required by some codecs
	// TODO: Could do this in ValidateConfig() ?
	maxWidth := videoConfig.MaxWidth
	if maxWidth%2 != 0 {
		maxWidth++
	}

	t.MediaFile().SetVframes(1)
	t.MediaFile().SetSkipAudio(true)
	t.MediaFile().SetSeekTime("0")                                     // Use first frame for thumbnail
	t.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:-2", maxWidth)) // -2 ensures height is a multiple of 2
	t.MediaFile().SetQScale(uint32(videoConfig.Quality))

	done := t.Run(false)
	err = <-done

	return
}

func videoTranscode(videoConfig VideoConfiguration, inputFile *pixelio.InputFile) (err error) {
	outputFilepath := pixelio.GetFileOutputPath(inputFile, videoConfig.OutputFileSuffix(false))

	// Ensure maxWidth is even - required by some codecs
	maxWidth := videoConfig.MaxWidth
	if maxWidth%2 != 0 {
		maxWidth++
	}

	t := new(transcoder.Transcoder)
	if err = t.Initialize(inputFile.Path, outputFilepath); err != nil {
		log.Println("Error initialising video transcoder:", err)
		return
	}

	fmt.Printf("Config is %+v\n", videoConfig)

	if videoConfig.FileType == "mp4" {
		t.MediaFile().SetVideoCodec("libx264")                             // TODO: Configure codecs via config
		t.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:-2", maxWidth)) // -2 ensures height is a multiple of 2
		t.MediaFile().SetMovFlags("+faststart")
		t.MediaFile().SetCRF(uint32(videoConfig.Quality))
		t.MediaFile().SetPreset(videoConfig.Preset)
		// t.MediaFile().SetAudioCodec("aac") // unsure if we want to explicitly say - will ffmpeg pick a good default otherwise?
		//t.MediaFile().SetSkipAudio(true) // disable audio - we want to strip audio when generating input file instead
	} else if videoConfig.FileType == "webm" {
		t.MediaFile().SetVideoCodec("libvpx-vp9")                          // TODO: Configure codecs via config
		t.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:-2", maxWidth)) // -2 ensures height is a multiple of 2
	} else if videoConfig.FileType == "av1" {
		t.MediaFile().SetVideoCodec("libaom-av1")
		t.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:-2", maxWidth))
		t.MediaFile().SetCRF(uint32(videoConfig.Quality))
		t.MediaFile().SetVideoBitRate("0")
	} else {
		err = fmt.Errorf("Unknown video file type '%s'", videoConfig.FileType)
		return
	}

	fmt.Printf("Writing file to %s\n", outputFilepath)

	done := t.Run(false)
	err = <-done
	return
}

// ProcessImage processes a single image
func (m *MediaJob) ProcessImage() (filenames []string, err error) {
	// TODO: Do a better job of handling errors - returning early and using multierror to report all errors to the caller
	// Read file in
	// os.File conforms to io.Reader, which we can call Decode on
	fh, err := os.Open(m.InputFile.Path)
	defer fh.Close()
	if err != nil {
		log.Fatal("Could not read file")
	}

	srcImage, _, err := image.Decode(fh)
	if err != nil {
		log.Fatal("Error decoding image: ", m.InputFile.Path)
	}

	for _, imageConfig := range m.MediaConfig.ImageConfigurations {
		// Resize image
		resizedImage := resizeImage(srcImage, imageConfig.MaxWidth)

		// Write file out
		if err := pixelio.EnsureOutputDirExists(m.InputFile.Subdir); err != nil {
			log.Fatal("Unable to prepare output dir:", err)
		}
		outputFilepath := pixelio.GetFileOutputPath(m.InputFile, imageConfig.OutputFileSuffix(false))
		fmt.Println("File output path is", outputFilepath)
		outfh, err := os.Create(outputFilepath)
		defer outfh.Close()
		if err != nil {
			log.Fatal(err)
		}

		// TODO: time each encode operation
		// Select encoder based on config...
		switch imageConfig.FileType {
		case JPG:
			fmt.Println("Encoding output file to JPG")
			jpeg.Encode(outfh, resizedImage, &jpeg.Options{Quality: imageConfig.Quality})
		case WebP:
			fmt.Println("Encoding output file to WebP with chai2010")
			var buf bytes.Buffer
			if err = webp.Encode(&buf, resizedImage, &webp.Options{Quality: float32(imageConfig.Quality)}); err != nil {
				log.Println(err)
			}
			if err = ioutil.WriteFile(outputFilepath, buf.Bytes(), 0664); err != nil {
				log.Println(err)
			}
		case WebPBin:
			fmt.Println("Encoding output file to WebP with webpbin")
			f, err := os.Create(outputFilepath)
			defer f.Close()
			if err != nil {
				log.Println(err)
			}

			wb := webpbin.Encoder{Quality: uint(imageConfig.Quality)}
			if err = wb.Encode(f, resizedImage); err != nil {
				log.Println(err)
			}

		default:
			fmt.Println("Error: unknown output format:", imageConfig.FileType)
			continue
		}

		filenames = append(filenames, outputFilepath)
	}

	return
}

func openImage() {

}

// imaging library typically returns image.NRGBA, so let's roll with that for now
func resizeImage(srcImage image.Image, targetWidth int) (resizedImage *image.NRGBA) {
	imgWidth := srcImage.Bounds().Max.X
	imgHeight := srcImage.Bounds().Max.Y
	aspectRatio := float64(imgHeight) / float64(imgWidth)

	resizeWidth := int(math.Min(float64(imgWidth), float64(targetWidth)))
	resizeHeight := int(float64(resizeWidth) * aspectRatio)

	// fmt.Printf("Resizing %d x %d -> %d x %d\n", imgWidth, imgHeight, resizeWidth, resizeHeight)

	// TODO: select best resizing algorithm. Lanczos sounds like a good starting point.
	resizedImage = imaging.Resize(srcImage, resizeWidth, resizeHeight, imaging.Lanczos)

	return
}
