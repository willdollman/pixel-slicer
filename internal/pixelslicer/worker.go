package pixelslicer

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"
	"github.com/willdollman/pixel-slicer/internal/mediaprocessor"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
)

// WorkerProcessMedia is a worker in a worker pool. It reads media jobs from the queue, and reports success/failure.
// This is fine for a one-shot thing where you have a fixed number of jobs, but how
// should it work with an unknown # jobs (and unknown delay between jobs)?
// Also doesn't allow us to pass errors back up the caller.
// func WorkerProcessMedia(jobs <-chan mediaprocessor.MediaJob, errc chan<- error, completion chan<- bool) {
func WorkerProcessMedia(jobs <-chan mediaprocessor.MediaJob, errc chan<- error, completion chan<- bool, progress *progressbar.ProgressBar) {
	for j := range jobs {
		mediaType := pixelio.GetMediaType(j.InputFile)

		var filenames []string
		var err error
		startTime := time.Now()

		// TODO: Here, or in the ProcessX methods, we should check the file still exists

		switch mediaType {
		case "image":
			filenames, err = j.ProcessImage()
			if err != nil {
				errc <- errors.Wrap(err, "Error processing image") // TODO: does this wrap add value?
				continue
			}
		case "video":
			filenames, err = j.ProcessVideo()
			if err != nil {
				errc <- errors.Wrap(err, "Error processing video")
				continue
			}
		default:
			errc <- errors.Wrapf(err, "Unable to process media, unknown media type '%s'", mediaType)
			continue
		}
		_ = startTime
		// fmt.Printf("Encoding '%s' took %.2fs\n", j.InputFile.Filename, time.Since(startTime).Seconds())

		postProcessStart := time.Now()
		if err := jobPostProcess(j, filenames); err != nil {
			errc <- errors.Wrap(err, "Error post-processing job")
			continue
		}
		_ = postProcessStart
		// fmt.Printf("Post-processing '%s' took %.2fs\n", j.InputFile.Filename, time.Since(postProcessStart).Seconds())
		progress.Add(1)
	}
	// When jobs is closed, signal completion to indicate this worker is finished
	completion <- true
}

// Perform any post-processing tasks after a job has been processed
func jobPostProcess(job mediaprocessor.MediaJob, filenames []string) error {
	// TODO: May not want to resized files to remain locally, so could remove them after moving

	// TODO: This should be updated to use concurrency in some way. Currently just uploads files sequentially.
	// Could upload files in parallel (make the most use of the network bandwidth) <- this option, I think
	// Or could just shove the jobs into the background to free up the processing thread <- but then what if there's an error?
	// Equally, multiple workers mean we'll already be uploading in parallel - too much could actually slow it down

	for _, filename := range filenames {
		filekey := pixelio.StripFileOutputDir(job.FSConfig.OutputDir, filename)

		// S3 upload
		if job.S3Client.Config.Enabled {
			fmt.Printf("Uploading to S3: %s\n", filename)
			err := job.S3Client.UploadFile(filename, filekey)
			if err != nil {
				return errors.Wrap(err, "Unable to upload output files to S3")
			}
		}
	}

	if job.FSConfig.MoveProcessed {
		// Move file to output dir
		if err := pixelio.MoveOriginal(job.InputFile, job.FSConfig.ProcessedDir); err != nil {
			return errors.Wrap(err, "Unable to move processed file to processed dirj")
		}
	}
	return nil
}
