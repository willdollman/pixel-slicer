package mediaprocessor

import (
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"

	"image/jpeg"

	"github.com/disintegration/imaging"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
)

type ImageProcessor interface {
	ResizeImage(inputFile string, outputDir string, width int, quality int, format string) (fileName string, err error)
}

type VideoProcessor interface {
	ResizeVideo(inputFile string, outputDir string, width int, quality int, format string) (fileName string, err error)
}

// ProcessImage processes a single image...
func ProcessImage(inputFile pixelio.InputFile) (err error) {
	fmt.Println("Going to resize", inputFile.Filename)

	// Read file in

	// Apparently os.File is an io.Reader ?
	fh, err := os.Open(inputFile.Path)
	if err != nil {
		log.Fatal("Could not read file")
	}

	srcImage, _, err := image.Decode(fh)
	if err != nil {
		log.Fatal("Error decoding image", inputFile)
	}

	// Resize image
	// TODO: Maintain aspect ratio; select best resizing algorithm
	dstImage128 := imaging.Resize(srcImage, 128, 128, imaging.Lanczos)

	// Write file out
	// We can have an output folder, but we also need to know what subfolder the image was in
	// Prepare by building filename and checking parent dir exists
	outputDir := "output"
	fileOutputDir := filepath.Join(outputDir, inputFile.Subdir)
	f, err := os.Stat(fileOutputDir)
	// If subdir doesn't exist, create it
	if os.IsNotExist(err) {
		fmt.Println("Creating dir", fileOutputDir)
		if err := os.Mkdir(fileOutputDir, 0755); err != nil {
			return err
		}
	} else {
		// If file with subdirectory name exists, ensure it is a directory
		if !f.IsDir() {
			fmt.Println("Not a directory", fileOutputDir)
			// err := MediaprocessorError{s: "Output subdirectory exists as a file"}
			// fmt.Println("Err:", err.s)
			return errors.New("Output subdirectory exists as a file")
		}
	}

	outputFilename := strings.TrimSuffix(inputFile.Filename, filepath.Ext(inputFile.Filename)) + ".jpg"

	outputFilepath := filepath.Join(fileOutputDir, outputFilename) // TODO: need to handle the file extension
	outfh, err := os.Create(outputFilepath)
	if err != nil {
		log.Fatal(err)
	}

	jpeg.Encode(outfh, dstImage128, nil)

	return
}
