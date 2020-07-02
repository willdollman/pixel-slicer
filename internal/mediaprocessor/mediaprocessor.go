package mediaprocessor

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/nickalie/go-webpbin"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer/config"
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
func WorkerProcessMedia(jobs <-chan MediaJob, results chan<- bool) {
	for j := range jobs {
		mediaType := pixelio.GetMediaType(j.InputFile)
		success := true

		switch mediaType {
		case "image":
			if err := ProcessImage(j.Config, j.InputFile); err != nil {
				fmt.Println("Error processing image:", err)
				success = false
			}
		case "video":
			if err := ProcessVideo(j.Config, j.InputFile); err != nil {
				fmt.Println("Error processing video:", err)
				success = false
			}
		default:
			success = false
		}
		results <- success
	}
}

// ProcessVideo processes a single video
func ProcessVideo(conf config.Config, inputFile pixelio.InputFile) (err error) {
	fmt.Println("Transcoding video, this may take a while...")
	t := new(transcoder.Transcoder)

	for _, videoConfig := range conf.VideoConfigurations {
		outputFilepath := pixelio.GetFileOutputPath(inputFile, videoConfig.MaxWidth, string(videoConfig.FileType))

		if err = t.Initialize(inputFile.Path, outputFilepath); err != nil {
			log.Printf("Error initialising video transcoder:", err)
			return
		}

		done := t.Run(false)
		err = <-done
		if err != nil {
			return
		}
	}
	return
}

// ProcessImage processes a single image
func ProcessImage(conf config.Config, inputFile pixelio.InputFile) (err error) {
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
		outputFilepath := pixelio.GetFileOutputPath(inputFile, imageConfig.MaxWidth, string(imageConfig.FileType))
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
		}
	}

	return
}

func openImage() {

}

// imaging library typically returns image.NRGBA, so let's roll with that for now
func resizeImage(srcImage image.Image, resizeWidth int) (resizedImage *image.NRGBA) {
	width := srcImage.Bounds().Max.X
	height := srcImage.Bounds().Max.Y

	resizeHeight := int(float64(resizeWidth) * (float64(height) / float64(width)))
	// fmt.Printf("Resizing %d x %d -> %d x %d\n", width, height, resizeWidth, resizeHeight)

	// TODO: select best resizing algorithm. Lanczos sounds like a good starting point.
	resizedImage = imaging.Resize(srcImage, resizeWidth, resizeHeight, imaging.Lanczos)

	return
}
