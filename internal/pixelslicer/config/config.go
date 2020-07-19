package config

import (
	"fmt"
	"path/filepath"
)

// Manage configuration

type Config struct {
	InputDir            string
	OutputDir           string
	MoveProcessed       bool
	ProcessedDir        string
	ImageConfigurations []ImageConfiguration
	VideoConfigurations []VideoConfiguration
}

// ValidateConfig validates that a given config is valid
func (c Config) ValidateConfig() (err error) {
	// Check that input dir and processed dir are not the same directory
	inputDirFull, err := filepath.Abs(c.InputDir)
	if err != nil {
		return err
	}
	processedDirFull, err := filepath.Abs(c.ProcessedDir)
	if err != nil {
		return err
	}
	fmt.Printf("%s =? %s\n", inputDirFull, processedDirFull)
	if inputDirFull == processedDirFull {
		return fmt.Errorf("Input dir '%s' cannot match Processed dir '%s'", inputDirFull, processedDirFull)
	}

	return
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
