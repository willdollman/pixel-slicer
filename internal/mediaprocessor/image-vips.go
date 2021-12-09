package mediaprocessor

import (
	"fmt"
	"io/ioutil"
	"log"

	vips "github.com/davidbyttow/govips/v2/vips"
)

// N.B. Error: invalid flag in pkg-config --cflags: -Xpreprocessor ?
// Set export CGO_CFLAGS_ALLOW="-Xpreprocessor" before
// running go get github.com/davidbyttow/govips/v2/vips

// ImageVips is an ImageProcessor which uses libvips
type ImageVips struct{}

var STARTEDLIBVIPS bool

func (i *ImageVips) Resize(m *MediaJob) (filenames []string, err error) {
	fmt.Printf("Hoping to resize some images here...\n")

	imgOrig, err := vips.NewImageFromFile(m.InputFile.Path)
	if err != nil {
		log.Fatalf("Could not load image")
	}

	for _, imageConfig := range m.MediaConfig.ImageConfigurations {
		img, err := imgOrig.Copy()
		if err != nil {
			log.Fatalf("Error copying image")
		}

		// Scale image
		err = img.Thumbnail(imageConfig.MaxWidth, 5000, vips.InterestingNone)
		if err != nil {
			log.Fatalf("Could not resize image")
		}

		// TODO: JPG vs webm vs ...
		var ep *vips.ExportParams
		switch imageConfig.FileType {
		case JPG:
			ep = getJpgExportParams(imageConfig)
		case WebP:
			ep = getWebpExportParams(imageConfig)
		default:
			log.Fatalf("Undefined FileType '%s'", imageConfig.FileType)
		}

		imgBytes, _, err := img.Export(ep)
		if err != nil {
			log.Fatalf("Failed to export image: %s", err)
		}

		outputFilepath := m.OutputPath(imageConfig)
		err = ioutil.WriteFile(outputFilepath, imgBytes, 0644)
		if err != nil {
			log.Fatalf("Unable to write file")
		}
	}

	return
}

func getJpgExportParams(i *ImageConfiguration) *vips.ExportParams {
	ep := vips.NewDefaultJPEGExportParams()

	ep.StripMetadata = true
	ep.Quality = i.Quality
	// TODO: anything else?

	return ep
}

func getWebpExportParams(i *ImageConfiguration) *vips.ExportParams {
	ep := vips.NewDefaultWEBPExportParams()

	ep.StripMetadata = true
	ep.Quality = i.Quality

	return ep
}
