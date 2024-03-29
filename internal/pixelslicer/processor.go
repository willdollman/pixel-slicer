package pixelslicer

import (
	"fmt"
	"log"
	"time"

	"github.com/radovskyb/watcher"
	"github.com/schollz/progressbar/v3"
	"github.com/willdollman/pixel-slicer/internal/config"
	"github.com/willdollman/pixel-slicer/internal/mediaprocessor"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
)

// ProcessFiles oversees all file processing tasks.
// It spawns a worker pool, and then calls filesystem-observing functions to queue jobs for those workers.
// It then monitors the workers, and shuts down when required.
func (p *PixelSlicer) ProcessFiles(conf config.ReadableConfig) {
	jobQueue := make(chan mediaprocessor.MediaJob, 2048)

	// Always queue any files which are already in the directory
	numInitialJobs := p.processOneShot(jobQueue) // TODO: multiply by number of render types?
	// fmt.Printf("\nProcessing %d jobs in initial directory...\n\n", numInitialJobs)
	bar := progressbar.New(numInitialJobs)

	errc := make(chan error)
	completion := make(chan bool)
	for w := 1; w <= conf.Workers; w++ {
		go WorkerProcessMedia(jobQueue, errc, completion, bar)
	}

	// We need to be careful that we don't try and process the same file twice - otherwise the workers will fight over it.
	// So until we mitigate that, start monitoring the directory once processOneShot has completed
	// Additionally, there's a "write hole" between its directory scan and the jobs being queued - though this should be fairly small
	// One way to avoid this might be to have an intermediate queue that's deduped?

	if conf.Watch {
		fmt.Println("Continuing to monitor input directory for new files...")
		p.processWatchDir(jobQueue)
	} else {
		// Not monitoring inputDir - we're only interested in the files already in the input directory,
		// so close jobs to signal we have no further tasks
		close(jobQueue)
	}

	// If jobs is closed, workers will send completion to indicate they're out of tasks.
	// Count out the workers so we can terminate cleanly.
	go func() {
		for i := 1; i <= conf.Workers; i++ {
			_ = <-completion
			// fmt.Println("A worker has finished!")
		}
		fmt.Println("\nAll jobs complete")
		close(errc)
	}()

	// Report errors
	for err := range errc {
		fmt.Printf("Error processing job: %s\n", err)
	}

	// TODO: Have a way to exit cleanly in the middle of a batch - ie workers should finish their current jobs, and then
	// call completion
}

// processWatchDir watches the input directory for newly added media files. If a new file is found,
// it is added to the jobQueue.
func (p *PixelSlicer) processWatchDir(jobQueue chan<- mediaprocessor.MediaJob) {
	w := watcher.New()

	w.FilterOps(watcher.Create)
	// TODO: Handle case where a file is renamed before it can be processed
	// w.FilterOps(watcher.Create, watcher.Rename)

	go func() {
		for {
			select {
			case event := <-w.Event:
				// Only
				if !event.IsDir() {
					fmt.Println(event)
					inputFile, err := pixelio.InputFileFromFullPath(p.FSConfig.InputDir, event.Path)
					if err != nil {
						log.Printf("Unable to create InputFile from event: %s\n", err)
						continue
					}
					fmt.Printf("Created inputfile from event: %+v\n", inputFile)

					// Check if inputFile is a valid file type
					validInputFiles := pixelio.FilterValidFiles([]*pixelio.InputFile{inputFile})
					if len(validInputFiles) == 0 {
						fmt.Printf("File was not a valid filetype")
						continue
					}

					job := p.CreateJob(inputFile)
					jobQueue <- job
				}
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	// Add input dir to watched directories
	if err := w.AddRecursive(p.FSConfig.InputDir); err != nil {
		log.Fatal(err)
	}

	// Start the watching process
	if err := w.Start(time.Second * 1); err != nil {
		log.Fatal(err)
	}
}

// processOneShot crawls a directory tree looking for files of the correct type. Any matching
// files are added to the jobQueue.
func (p *PixelSlicer) processOneShot(jobQueue chan<- mediaprocessor.MediaJob) int {
	files, err := pixelio.EnumerateDirContents(p.FSConfig.InputDir)
	if err != nil {
		log.Fatal("Cannot enumerate supplied directory", p.FSConfig.InputDir)
	}

	// Filter out valid file types
	filteredFiles := pixelio.FilterValidFiles(files)
	// Only used for stats - deletable
	mediaFiles := make(map[string][]*pixelio.InputFile)
	for mediaType, _ := range pixelio.TypeExtension() {
		mediaFiles[mediaType] = pixelio.FilterFileType(files, mediaType)
	}
	fmt.Printf("Found %d images and %d videos in '%s'\n\n", len(mediaFiles["image"]), len(mediaFiles["video"]), p.FSConfig.InputDir)

	for _, file := range filteredFiles {
		// fmt.Printf("Queued '%s' (%d/%d)\n", file.Filename, i+1, len(filteredFiles)) // TODO: verbose
		// Multithreaded image processing
		job := p.CreateJob(file)
		jobQueue <- job
	}

	return len(filteredFiles)
}

// CreateJob creates a mediaprocessor.MediaJob for a given input file
func (p *PixelSlicer) CreateJob(file *pixelio.InputFile) mediaprocessor.MediaJob {
	return mediaprocessor.MediaJob{
		FSConfig:       p.FSConfig,
		MediaConfig:    p.MediaConfig,
		MediaProcessor: p.MediaProcessor,
		S3Client:       p.S3Client,
		InputFile:      file,
	}
}
