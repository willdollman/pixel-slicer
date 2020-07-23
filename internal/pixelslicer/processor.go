package pixelslicer

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/radovskyb/watcher"
	"github.com/willdollman/pixel-slicer/internal/mediaprocessor"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer/config"
)

// ProcessFiles oversees all file processing tasks.
// It spawns a worker pool, and then calls filesystem-observing functions to queue jobs for those workers.
// It then monitors the workers, and shuts down when required.
func ProcessFiles(conf config.Config) {
	// Create some workers, based on the number of CPU cores available
	numWorkers := runtime.NumCPU() / 2
	fmt.Println("Creating", numWorkers, "workers")
	jobs := make(chan mediaprocessor.MediaJob, 2048)
	errc := make(chan error)
	completion := make(chan bool)
	for w := 1; w <= numWorkers; w++ {
		go mediaprocessor.WorkerProcessMedia(jobs, errc, completion)
	}

	// Always queue any files which are already in the directory
	processOneShot(conf, jobs)

	// We need to be careful that we don't try and process the same file twice - otherwise the workers will fight over it.
	// So until we mitigate that, start monitoring the directory once processOneShot has completed
	// Additionally, there's a "write hole" between its directory scan and the jobs being queued - though this should be fairly small
	// One way to avoid this might be to have an intermediate queue that's deduped?

	if conf.Watch {
		fmt.Println("Continuing to monitor input directory for new files...")
		processWatchDir(conf, jobs)
	} else {
		// Not monitoring inputDir - we're only interested in the files already in the input directory,
		// so close jobs to signal we have no further tasks
		close(jobs)
	}

	// If jobs is closed, workers will send completion to indicate they're out of tasks.
	// Count out the workers so we can terminate cleanly.
	go func() {
		for i := 1; i <= numWorkers; i++ {
			_ = <-completion
			fmt.Println("A worker has finished!")
		}
		fmt.Println("All workers have finished - we can shut down!")
		close(errc)
	}()

	// Report errors
	for err := range errc {
		fmt.Printf("Error processing job: %s\n", err)
	}

	// TODO: Have a way to exit cleanly in the middle of a batch - ie workers should finish their current jobs, and then
	// call completion
}

func processWatchDir(conf config.Config, jobQueue chan<- mediaprocessor.MediaJob) {
	w := watcher.New()

	w.FilterOps(watcher.Create)
	// TODO: What about rename? - if a file is renamed before it can be processed
	// w.FilterOps(watcher.Create, watcher.Rename)

	go func() {
		for {
			select {
			case event := <-w.Event:
				// Only
				if !event.IsDir() {
					fmt.Println(event)
					inputFile, err := pixelio.InputFileFromFullPath(conf.InputDir, event.Path)
					if err != nil {
						log.Printf("Unable to create InputFile from event: %s\n", err)
						continue
					}
					// TOOD: Queue inputFile as job
					fmt.Printf("Created inputfile from event: %+v\n", inputFile)

					// Check if inputFile is a valid file type
					validInputFiles := pixelio.FilterValidFiles([]pixelio.InputFile{inputFile})
					if len(validInputFiles) == 0 {
						fmt.Printf("File was not a valid filetype")
						continue
					}

					job := mediaprocessor.MediaJob{Config: conf, InputFile: inputFile} // TODO: Would we save memory by passing a reference to conf?
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
	if err := w.AddRecursive(conf.InputDir); err != nil {
		log.Fatal(err)
	}

	// Start the watching process
	if err := w.Start(time.Second * 1); err != nil {
		log.Fatal(err)
	}
}

// ProcessWatchDir watches a directory and processes all new files which are added
func processWatchDirFsnotify(conf config.Config) {
	// We need to process all the files which are already in the directory
	// Then we need to monitor it for new files...

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					fmt.Println("File has been created:", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error", err)
			}
		}
	}()

	// Need to queue input directory AND all subdirectories
	fmt.Println("Watching dir", conf.InputDir)
	err = watcher.Add(conf.InputDir)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

// ProcessOneShot processes a directory of files in a one-shot fashion,
// not worrying about new files being added
func processOneShot(conf config.Config, jobQueue chan<- mediaprocessor.MediaJob) {
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

	log.Println("Will queue", len(filteredFiles), "jobs")

	for i, file := range filteredFiles {
		fmt.Printf("Queueing job '%s' (%d/%d)\n", file.Filename, i+1, len(filteredFiles))
		// Classic image processing!
		// if err := mediaprocessor.ProcessImage(conf, file); err != nil {
		// 	log.Fatal("Error processing file", file)
		// 	// Unsure why this needs to be fatal - we segfault for some reason otherwise...
		// }

		// Multithreaded image processing
		job := mediaprocessor.MediaJob{Config: conf, InputFile: file} // TODO: Would we save memory by passing a reference to conf?
		jobQueue <- job
	}

	fmt.Println("Finished queueing jobs in ProcessOneShot")

}
