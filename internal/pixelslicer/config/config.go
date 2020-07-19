package config

import "fmt"

// Manage configuration

type Config struct {
	InputDir            string
	OutputDir           string
	MoveProcessed       bool
	ProcessedDir        string
	ImageConfigurations []ImageConfiguration
	VideoConfigurations []VideoConfiguration
}

type MediaConfiguration interface {
	OutputFileName(bool) string
}

// Might be useful to have a MediaConfiguration interface that we can use, with a "OutputFileName"
// method which can be used by pixelio.GetFileOutputPath
type ImageConfiguration struct {
	MaxWidth int
	Quality  int
	FileType FileOutputType
}

func (i ImageConfiguration) OutputFileName(simpleName bool) string {
	if simpleName {
		return fmt.Sprintf("-%d.%s", i.MaxWidth, string(i.FileType))
	}

	return fmt.Sprintf("-%d-q%d.%s", i.MaxWidth, i.Quality, string(i.FileType))
}

type VideoConfiguration struct {
	MaxWidth int
	Quality  int
	Preset   string
	FileType FileOutputType
}

func (i VideoConfiguration) OutputFileName(simpleName bool) string {
	if simpleName {
		return fmt.Sprintf("-%d.%s", i.MaxWidth, string(i.FileType))
	}
	return fmt.Sprintf("-%d-q%d-p%s.%s", i.MaxWidth, i.Quality, i.Preset, string(i.FileType))
}

type FileOutputType string

const (
	JPG     FileOutputType = "jpg"
	WebP    FileOutputType = "webp"
	WebPBin FileOutputType = "webpbin.webp"
	MP4     FileOutputType = "mp4"
	WebM    FileOutputType = "webm"
)

// Read from config file with Viper?
