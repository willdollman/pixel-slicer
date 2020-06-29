package mediaprocessor

import (
	"fmt"
	"image"
	"log"
	"os"

	"image/jpeg"

	"github.com/disintegration/imaging"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer/config"
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

type ImageJob struct {
	Config    config.PixelSlicerConfig
	InputFile pixelio.InputFile
}

// This is fine for a one-shot thing where you have a fixed number of jobs, but how
// should it work with an unknown # jobs (and unknown delay between jobs)?
func WorkerProcessImage(jobs <-chan ImageJob, results chan<- bool) {
	for j := range jobs {
		success := true
		if err := ProcessImage(j.Config, j.InputFile); err != nil {
			success = false
		}
		results <- success
	}
}

// We want to wrap ProcessImage in a worker.
// ProcessImage processes a single image...
func ProcessImage(conf config.PixelSlicerConfig, inputFile pixelio.InputFile) (err error) {
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
		outputFilepath := pixelio.GetFileOutputPath(inputFile, imageConfig.MaxWidth, "jpg")
		fmt.Println("File output path is", outputFilepath)
		outfh, err := os.Create(outputFilepath)
		defer outfh.Close()
		if err != nil {
			log.Fatal(err)
		}
		jpeg.Encode(outfh, resizedImage, &jpeg.Options{Quality: imageConfig.Quality})
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
