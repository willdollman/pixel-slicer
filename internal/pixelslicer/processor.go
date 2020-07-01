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

	// This should filter into each supported/enabled type and run the appropriate processor
	filteredFiles := pixelio.FilterFileType(files, "image")
	fmt.Printf("Processing %d files, %d images\n", len(files), len(filteredFiles))

	numWorkers := runtime.NumCPU() / 2
	numWorkers = 1 // TODO: Move to config/CLI switch
	fmt.Println("Got", numWorkers, "cores")
	jobs := make(chan mediaprocessor.ImageJob, len(filteredFiles))
	results := make(chan bool, len(filteredFiles))

	for w := 1; w <= numWorkers; w++ {
		go mediaprocessor.WorkerProcessImage(jobs, results)
	}

	log.Println("Will queue", len(filteredFiles), "jobs on", numWorkers, "workers")

	for i, file := range filteredFiles {
		fmt.Printf("Processing file '%s' (%d/%d)\n", file.Filename, i+1, len(filteredFiles))
		// Classic image processing!
		// err := mediaprocessor.ProcessImage(conf, file)

		// Multithreaded image processing
		job := mediaprocessor.ImageJob{Config: conf, InputFile: file}
		jobs <- job

		if err != nil {
			log.Fatal("Error processing file", file)
			// Unsure why this needs to be fatal - we segfault for some reason otherwise...
		}

		// log.Fatal("Finishing after one image")
	}

	for i, _ := range filteredFiles {
		<-results
		fmt.Println("Finished processing job", i)
	}
}
