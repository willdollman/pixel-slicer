package mediaprocessor

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/nickalie/go-webpbin"
	"github.com/willdollman/pixel-slicer/internal/pixelio"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer/config"
	"github.com/xfrr/goffmpeg/transcoder"
)

// ImageOutputConfig defines an output image configuration
type ImageOutputConfig struct {
	width   int
	quality int
}

type ImageProcessor interface {
	ResizeImage(inputFile string, outputDir string, width int, quality int, format string) (fileName string, err error)
}

type VideoProcessor interface {
	ResizeVideo(inputFile string, outputDir string, width int, quality int, format string) (fileName string, err error)
}

type MediaJob struct {
	Config    config.Config
	InputFile pixelio.InputFile
}

// TODO: Process errors properly

// WorkerProcessMedia is a worker in a worker pool. It reads media jobs from the queue, and reports success/failure.
// This is fine for a one-shot thing where you have a fixed number of jobs, but how
// should it work with an unknown # jobs (and unknown delay between jobs)?
// Also doesn't allow us to pass errors back up the caller.
func WorkerProcessMedia(jobs <-chan MediaJob, results chan<- bool) {
	for j := range jobs {
		mediaType := pixelio.GetMediaType(j.InputFile)
		success := true

		switch mediaType {
		case "image":
			if err := ProcessImage(j.Config, j.InputFile); err != nil {
				fmt.Println("Error processing image:", err)
				success = false
			}
		case "video":
			if err := ProcessVideo(j.Config, j.InputFile); err != nil {
				fmt.Println("Error processing video:", err)
				success = false
			}
		default:
			success = false
		}

		if err := jobPostProcess(j); err != nil {
			fmt.Println("Error post-processing job:", err)
			success = false
		}
		results <- success
	}
}

// Perform any post-processing tasks after a job has been processed
func jobPostProcess(job MediaJob) (err error) {
	// TODO: Perhaps moving files to remote output location (SFTP, S3, ...) should occur here?
	// May also not want to resized files to remain locally, so could remove them after moving

	if job.Config.MoveProcessed {
		// Move file to output dir
		if err = pixelio.MoveOriginal(job.InputFile, job.Config.ProcessedDir); err != nil {
			log.Println("Unable to move processed file to processed dir:", err)
		}
	}
	return
}

// ProcessVideo processes a single video
func ProcessVideo(conf config.Config, inputFile pixelio.InputFile) (err error) {
	fmt.Println("Transcoding video, this may take a while...")
	t := new(transcoder.Transcoder)

	for _, videoConfig := range conf.VideoConfigurations {
		outputFilepath := pixelio.GetFileOutputPath(inputFile, videoConfig)

		if err = t.Initialize(inputFile.Path, outputFilepath); err != nil {
			log.Println("Error initialising video transcoder:", err)
			return
		}

		// Configuration options to try:
		// bitrate, resolution, presets (?), encode quality/speed

		// TODO: Configure codecs via config
		t.MediaFile().SetVideoCodec("libx264")
		// t.MediaFile().SetAudioCodec("aac") // unsure if we want to explicitly say - will ffmpeg pick a good default otherwise?
		//t.MediaFile().SetSkipAudio(true) // disable audio - we want to strip audio when generating input file instead

		// TODO: either fix libary to let you specify one of the dimensions, or compute dimensions

		t.MediaFile().SetResolution("1120x630") // Bug in the library which means you have to specify both dimensions
		t.MediaFile().SetMovFlags("+faststart")
		t.MediaFile().SetCRF(uint32(videoConfig.Quality))
		t.MediaFile().SetPreset(videoConfig.Preset)

		done := t.Run(false)
		err = <-done
		if err != nil {
			return
		}
	}
	return
}

// ProcessImage processes a single image
func ProcessImage(conf config.Config, inputFile pixelio.InputFile) (err error) {
	// Read file in
	// os.File conforms to io.Reader, which we can call Decode on
	fh, err := os.Open(inputFile.Path)
	defer fh.Close()
	if err != nil {
		log.Fatal("Could not read file")
	}

	srcImage, _, err := image.Decode(fh)
	if err != nil {
		log.Fatal("Error decoding image: ", inputFile.Path)
	}

	for _, imageConfig := range conf.ImageConfigurations {
		// Resize image
		resizedImage := resizeImage(srcImage, imageConfig.MaxWidth)

		// Write file out
		if err := pixelio.EnsureOutputDirExists(inputFile.Subdir); err != nil {
			log.Fatal("Unable to prepare output dir:", err)
		}
		outputFilepath := pixelio.GetFileOutputPath(inputFile, imageConfig)
		fmt.Println("File output path is", outputFilepath)
		outfh, err := os.Create(outputFilepath)
		defer outfh.Close()
		if err != nil {
			log.Fatal(err)
		}

		// TODO: time each encode operation
		// Select encoder based on config...
		switch imageConfig.FileType {
		case config.JPG:
			fmt.Println("Encoding output file to JPG")
			jpeg.Encode(outfh, resizedImage, &jpeg.Options{Quality: imageConfig.Quality})
		case config.WebP:
			fmt.Println("Encoding output file to WebP with chai2010")
			var buf bytes.Buffer
			if err = webp.Encode(&buf, resizedImage, &webp.Options{Quality: float32(imageConfig.Quality)}); err != nil {
				log.Println(err)
			}
			if err = ioutil.WriteFile(outputFilepath, buf.Bytes(), 0664); err != nil {
				log.Println(err)
			}
		case config.WebPBin:
			fmt.Println("Encoding output file to WebP with webpbin")
			f, err := os.Create(outputFilepath)
			defer f.Close()
			if err != nil {
				log.Println(err)
			}

			wb := webpbin.Encoder{Quality: uint(imageConfig.Quality)}
			if err = wb.Encode(f, resizedImage); err != nil {
				log.Println(err)
			}

		default:
			fmt.Println("Error: unknown output format:", imageConfig.FileType)
		}
	}

	return
}

func openImage() {

}

// imaging library typically returns image.NRGBA, so let's roll with that for now
func resizeImage(srcImage image.Image, resizeWidth int) (resizedImage *image.NRGBA) {
	width := srcImage.Bounds().Max.X
	height := srcImage.Bounds().Max.Y

	resizeHeight := int(float64(resizeWidth) * (float64(height) / float64(width)))
	// fmt.Printf("Resizing %d x %d -> %d x %d\n", width, height, resizeWidth, resizeHeight)

	// TODO: select best resizing algorithm. Lanczos sounds like a good starting point.
	resizedImage = imaging.Resize(srcImage, resizeWidth, resizeHeight, imaging.Lanczos)

	return
}
