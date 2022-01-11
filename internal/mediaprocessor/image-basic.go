package mediaprocessor

import (
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"math"
	"os"

	"github.com/disintegration/imaging"
)

// ImageBasic is an ImageProcessor which uses a mix of Go image generation libraries
type ImageBasic struct{}

// ProcessImage processes a single image
func (i *ImageBasic) Resize(m *MediaJob) (filenames []string, err error) {
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
		outputFilepath := m.OutputPath(imageConfig)
		fmt.Println("File output path is", outputFilepath)
		outfh, err := os.Create(outputFilepath)
		defer outfh.Close()
		if err != nil {
			log.Fatal(err)
		}

		switch imageConfig.FileType {
		case JPG:
			fmt.Println("Encoding output file to JPG")
			jpeg.Encode(outfh, resizedImage, &jpeg.Options{Quality: imageConfig.Quality})
		case WebP:
			fmt.Println("WebP output disabled")
			// fmt.Println("Encoding output file to WebP with chai2010")
			// var buf bytes.Buffer
			// if err = webp.Encode(&buf, resizedImage, &webp.Options{Quality: float32(imageConfig.Quality)}); err != nil {
			// 	log.Println(err)
			// }
			// if err = ioutil.WriteFile(outputFilepath, buf.Bytes(), 0664); err != nil {
			// 	log.Println(err)
			// }
		default:
			fmt.Println("Error: unknown output format:", imageConfig.FileType)
			continue
		}

		filenames = append(filenames, outputFilepath)
	}

	return
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
