package mediaprocessor

import (
	"fmt"
	"image"
	"log"
	"os"

	"image/jpeg"

	"github.com/disintegration/imaging"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
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

// ProcessImage processes a single image...
func ProcessImage(inputFile pixelio.InputFile) (err error) {
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

	// TODO: Move to config file
	imageOutputConfigurations := []ImageOutputConfig{
		{100, 80},
		{500, 80},
		{1000, 80},
		{2000, 80},
	}

	for _, config := range imageOutputConfigurations {
		// Resize image
		resizedImage := resizeImage(srcImage, config.width)

		// Write file out
		if err := pixelio.EnsureOutputDirExists(inputFile.Subdir); err != nil {
			log.Fatal("Unable to prepare output dir:", err)
		}
		outputFilepath := pixelio.GetFileOutputPath(inputFile, config.width, "jpg")
		fmt.Println("File output path is", outputFilepath)
		outfh, err := os.Create(outputFilepath)
		defer outfh.Close()
		if err != nil {
			log.Fatal(err)
		}
		jpeg.Encode(outfh, resizedImage, &jpeg.Options{Quality: config.quality})
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
