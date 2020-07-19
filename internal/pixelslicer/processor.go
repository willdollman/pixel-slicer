package pixelslicer

import (
	"fmt"
	"log"
	"runtime"

	"github.com/willdollman/pixel-slicer/internal/mediaprocessor"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer/config"
)

// ProcessOneShot processes a directory of files in a one-shot fashion,
// not worrying about new files being added
func ProcessOneShot(conf config.Config) {
	fmt.Println("Processing directory", conf.InputDir)

	files, err := pixelio.EnumerateDirContents(conf.InputDir)
	if err != nil {
		log.Fatal("Cannot enumerate supplied directory", conf.InputDir)
	}

	// Filter out valid file types
	filteredFiles := pixelio.FilterValidFiles(files)
	// Only used for stats - deletable
	mediaFiles := make(map[string][]pixelio.InputFile)
	for mediaType, _ := range pixelio.TypeExtension() {
		mediaFiles[mediaType] = pixelio.FilterFileType(files, mediaType)
	}
	fmt.Printf("Processing %d files, %d images, %d videos\n", len(files), len(mediaFiles["image"]), len(mediaFiles["video"]))

	numWorkers := runtime.NumCPU() / 2
	// numWorkers = 1 // TODO: Move to config/CLI switch
	fmt.Println("Got", numWorkers, "cores")
	jobs := make(chan mediaprocessor.MediaJob, len(filteredFiles))
	results := make(chan bool, len(filteredFiles))

	for w := 1; w <= numWorkers; w++ {
		go mediaprocessor.WorkerProcessMedia(jobs, results)
	}

	log.Println("Will queue", len(filteredFiles), "jobs on", numWorkers, "workers")

	for i, file := range filteredFiles {
		fmt.Printf("Processing file '%s' (%d/%d)\n", file.Filename, i+1, len(filteredFiles))
		// Classic image processing!
		// if err := mediaprocessor.ProcessImage(conf, file); err != nil {
		// 	log.Fatal("Error processing file", file)
		// 	// Unsure why this needs to be fatal - we segfault for some reason otherwise...
		// }

		// Multithreaded image processing
		job := mediaprocessor.MediaJob{Config: conf, InputFile: file}
		jobs <- job

		// log.Fatal("Finishing after one image")
	}

	for i, _ := range filteredFiles {
		<-results
		fmt.Println("Finished processing job", i)
	}
}
