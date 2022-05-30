package mediaprocessor

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
	"github.com/willdollman/pixel-slicer/internal/s3"
)

// ImageProcessor is an interface for types which can process images
type ImageProcessor interface {
	Resize(*MediaJob) ([]string, error)
}

// VideoProcessor is an interface for types which can process videos
type VideoProcessor interface {
	Thumbnail(*MediaJob, *VideoConfiguration) error
	Transcode(*MediaJob, *VideoConfiguration) error
}

type MediaProcessor struct {
	Image ImageProcessor
	Video VideoProcessor
}

func New() (mediaProcessor *MediaProcessor) {
	// Return default media processors
	// TODO: Allow media processors to be selected
	return &MediaProcessor{
		Image: &ImageVips{},
		Video: &VideoGotranscoder{},
	}
}

// MediaJob represents an encoding job to perform, and contains types representing the
// target file, where the files should be written, encoding settings, and which processor to use.
type MediaJob struct {
	MediaConfig    *MediaConfig
	FSConfig       *FSConfig
	S3Client       *s3.S3Client
	InputFile      *pixelio.InputFile
	MediaProcessor *MediaProcessor
}

// OutputPath returns the full output path for a MediaJob with a specific MediaConfiguration
// e.g. output/subdir1/sunset-x100.jpg
func (m *MediaJob) OutputPath(mediaConfiguration MediaConfiguration) string {
	return pixelio.GetFileOutputPath(
		m.FSConfig.OutputDir,
		m.InputFile,
		mediaConfiguration.OutputFileSuffix(m.FSConfig.DebugFilenames),
	)
}

// CheckOutputDir ensures that a job's output subdirectory exists
func (m *MediaJob) CheckOutputDir() {
	if err := pixelio.EnsureOutputDirExists(m.FSConfig.OutputDir, m.InputFile.Subdir); err != nil {
		log.Fatal("Unable to prepare output dir:", err)
	}
}

// ProcessImage dispatches an image resize job to the configured ImageProcessor
func (m *MediaJob) ProcessImage() (filenames []string, err error) {
	if len(m.MediaConfig.ImageConfigurations) == 0 {
		return
	}

	// Image encoding is more efficient if image file is read in and decoded once
	// and output at multiple sizes, so this is performed in Resize()
	m.CheckOutputDir()
	return m.MediaProcessor.Image.Resize(m)
}

// ProcessVideo dispatchse a video resize job to the configured VideoProcessor
func (m *MediaJob) ProcessVideo() (filenames []string, errs error) {
	if len(m.MediaConfig.VideoConfigurations) == 0 {
		return
	}

	fmt.Println("Transcoding video, this may take a while...")

	m.CheckOutputDir()

	// Video encoding doesn't store the file in memory, so iterate through the MediaTypes here
	for _, videoConfig := range m.MediaConfig.VideoConfigurations {
		var err error
		encodeStartTime := time.Now()

		// Depending on the requested media type, either transcode video or generate a thumbnail
		switch videoConfig.FileType.GetMediaType() {
		case Image:
			err = m.MediaProcessor.Video.Thumbnail(m, videoConfig)
		case Video:
			err = m.MediaProcessor.Video.Transcode(m, videoConfig)
		default:
			err = fmt.Errorf("configuration contains unknown media type: %s", videoConfig.FileType)
		}

		fmt.Printf("Encoding took %.2fs\n", time.Since(encodeStartTime).Seconds())

		if err == nil {
			outputFilepath := m.OutputPath(videoConfig)
			filenames = append(filenames, outputFilepath)
		} else {
			errs = multierror.Append(errs, err)
		}
	}

	return
}
