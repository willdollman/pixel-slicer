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
	"github.com/pkg/errors"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer/config"
	"github.com/willdollman/pixel-slicer/internal/s3"
	"github.com/xfrr/goffmpeg/transcoder"
)

// ImageOutputConfig defines an output image configuration
type ImageOutputConfig struct {
	width   int
	quality int
}

type ImageProcessor interface {
	ResizeImage(inputFile string, outputDir string, width int, quality int, format string) (fileName string, err error)
}

type VideoProcessor interface {
	ResizeVideo(inputFile string, outputDir string, width int, quality int, format string) (fileName string, err error)
}

type MediaJob struct {
	Config    config.Config
	InputFile pixelio.InputFile
}

// WorkerProcessMedia is a worker in a worker pool. It reads media jobs from the queue, and reports success/failure.
// This is fine for a one-shot thing where you have a fixed number of jobs, but how
// should it work with an unknown # jobs (and unknown delay between jobs)?
// Also doesn't allow us to pass errors back up the caller.
func WorkerProcessMedia(jobs <-chan MediaJob, errc chan<- error, completion chan<- bool) {
	for j := range jobs {
		mediaType := pixelio.GetMediaType(j.InputFile)

		var filenames []string
		var err error
		startTime := time.Now()

		// TODO: Here, or in the ProcessX methods, we should check the file still exists

		switch mediaType {
		case "image":
			filenames, err = ProcessImage(j.Config, j.InputFile)
			if err != nil {
				errc <- errors.Wrap(err, "Error processing image") // TODO: not sure this wrap adds much!
				continue
			}
		case "video":
			filenames, err = ProcessVideo(j.Config, j.InputFile)
			if err != nil {
				errc <- errors.Wrap(err, "Error processing video")
				continue
			}
		default:
			errc <- errors.Wrapf(err, "Unable to process media, unknown media type '%s'", mediaType)
			continue
		}
		fmt.Printf("Encoding '%s' took %.2fs\n", j.InputFile.Filename, time.Now().Sub(startTime).Seconds())

		postProcessStart := time.Now()
		if err := jobPostProcess(j, filenames); err != nil {
			errc <- errors.Wrap(err, "Error post-processing job")
			continue
		}
		fmt.Printf("Post-processing '%s' took %.2fs\n", j.InputFile.Filename, time.Now().Sub(postProcessStart).Seconds())
	}
	// When jobs is closed, signal completion to indicate this worker is finished
	completion <- true
}

// Perform any post-processing tasks after a job has been processed
func jobPostProcess(job MediaJob, filenames []string) error {
	// TODO: May not want to resized files to remain locally, so could remove them after moving

	// TODO: This should be updated to use concurrency in some way. Currently just uploads files sequentially.
	// Could upload files in parallel (make the most use of the network bandwidth) <- this option, I think
	// Or could just shove the jobs into the background to free up the processing thread <- but then what if there's an error?
	// Equally, multiple workers mean we'll already be uploading in parallel - too much could actually slow it down

	for _, filename := range filenames {
		filekey := pixelio.StripFileOutputDir(filename)

		// S3 upload
		if job.Config.S3Enabled {
			fmt.Printf("Uploading to S3: %s\n", filename)
			err := s3.UploadFile(job.Config.S3Session, job.Config.S3Config.Bucket, filename, filekey)
			if err != nil {
				return errors.Wrap(err, "Unable to upload output files to S3")
			}
		}
	}

	if job.Config.MoveProcessed {
		// Move file to output dir
		if err := pixelio.MoveOriginal(job.InputFile, job.Config.ProcessedDir); err != nil {
			return errors.Wrap(err, "Unable to move processed file to processed dirj")
		}
	}
	return nil
}

// ProcessVideo processes a single video
func ProcessVideo(conf config.Config, inputFile pixelio.InputFile) (filenames []string, allErrors error) {
	fmt.Println("Transcoding video, this may take a while...")

	if err := pixelio.EnsureOutputDirExists(inputFile.Subdir); err != nil {
		log.Fatal("Unable to prepare output dir:", err)
	}

	for _, videoConfig := range conf.VideoConfigurations {
		var err error
		// Depending on the requested media type, either transcode video or generate a thumbnail
		if videoConfig.FileType.GetMediaType() == config.Video {
			err = videoTranscode(videoConfig, inputFile)
		} else if videoConfig.FileType.GetMediaType() == config.Image {
			err = videoThumbnail(videoConfig, inputFile)
		} else {
			err = fmt.Errorf("Configuration contains unknown media type: %s", videoConfig.FileType)
		}

		if err == nil {
			outputFilepath := pixelio.GetFileOutputPath(inputFile, videoConfig)
			filenames = append(filenames, outputFilepath)
		} else {
			allErrors = multierror.Append(allErrors, err)
		}
	}
	return
}

func videoThumbnail(videoConfig config.VideoConfiguration, inputFile pixelio.InputFile) (err error) {
	outputFilepath := pixelio.GetFileOutputPath(inputFile, videoConfig)

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

func videoTranscode(videoConfig config.VideoConfiguration, inputFile pixelio.InputFile) (err error) {
	outputFilepath := pixelio.GetFileOutputPath(inputFile, videoConfig)

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

	t.MediaFile().SetVideoCodec("libx264")                             // TODO: Configure codecs via config
	t.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:-2", maxWidth)) // -2 ensures height is a multiple of 2
	t.MediaFile().SetMovFlags("+faststart")
	t.MediaFile().SetCRF(uint32(videoConfig.Quality))
	t.MediaFile().SetPreset(videoConfig.Preset)
	// t.MediaFile().SetAudioCodec("aac") // unsure if we want to explicitly say - will ffmpeg pick a good default otherwise?
	//t.MediaFile().SetSkipAudio(true) // disable audio - we want to strip audio when generating input file instead

	done := t.Run(false)
	err = <-done
	return
}

// ProcessImage processes a single image
func ProcessImage(conf config.Config, inputFile pixelio.InputFile) (filenames []string, err error) {
	// TODO: Do a better job of handling errors - returning early and using multierror to report all errors to the caller
	// Read file in
	// os.File conforms to io.Reader, which we can call Decode on
	fh, err := os.Open(inputFile.Path)
	defer fh.Close()
	if err != nil {
		log.Fatal("Could not read file")
	}

	srcImage, _, err := image.Decode(fh)
	if err != nil {
		log.Fatal("Error decoding image: ", inputFile.Path)
	}

	for _, imageConfig := range conf.ImageConfigurations {
		// Resize image
		resizedImage := resizeImage(srcImage, imageConfig.MaxWidth)

		// Write file out
		if err := pixelio.EnsureOutputDirExists(inputFile.Subdir); err != nil {
			log.Fatal("Unable to prepare output dir:", err)
		}
		outputFilepath := pixelio.GetFileOutputPath(inputFile, imageConfig)
		fmt.Println("File output path is", outputFilepath)
		outfh, err := os.Create(outputFilepath)
		defer outfh.Close()
		if err != nil {
			log.Fatal(err)
		}

		// TODO: time each encode operation
		// Select encoder based on config...
		switch imageConfig.FileType {
		case config.JPG:
			fmt.Println("Encoding output file to JPG")
			jpeg.Encode(outfh, resizedImage, &jpeg.Options{Quality: imageConfig.Quality})
		case config.WebP:
			fmt.Println("Encoding output file to WebP with chai2010")
			var buf bytes.Buffer
			if err = webp.Encode(&buf, resizedImage, &webp.Options{Quality: float32(imageConfig.Quality)}); err != nil {
				log.Println(err)
			}
			if err = ioutil.WriteFile(outputFilepath, buf.Bytes(), 0664); err != nil {
				log.Println(err)
			}
		case config.WebPBin:
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
