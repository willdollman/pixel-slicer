package pixelslicer

import (
	"fmt"
	"log"

	"github.com/willdollman/pixel-slicer/internal/mediaprocessor"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
)

func main() {
	fmt.Println("Hi")
}

// ProcessOneShot processes a directory of files in a one-shot fashion,
// not worrying about new files being added
func ProcessOneShot(dir string) {
	fmt.Println("Processing directory", dir)

	files, err := pixelio.EnumerateDirContents(dir)
	if err != nil {
		log.Fatal("Cannot enumerate supplied directory", dir)
	}

	// This should filter into each supported/enabled type and run the appropriate processor
	filteredFiles := pixelio.FilterFileType(files, "image")
	fmt.Printf("Processing %d files, %d images\n", len(files), len(filteredFiles))

	for i, file := range filteredFiles {
		fmt.Printf("Processing file '%s' (%d/%d)\n", file.Filename, i+1, len(filteredFiles))
		err := mediaprocessor.ProcessImage(file)
		if err != nil {
			log.Fatal("Error processing file", file)
			// Unsure why this needs to be fatal - we segfault for some reason otherwise...
		}

		// log.Fatal("Finishing after one image")
	}
}
