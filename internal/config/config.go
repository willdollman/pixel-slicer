package config

import (
	"fmt"
	"path/filepath"

	"github.com/willdollman/pixel-slicer/internal/mediaprocessor"
	"github.com/willdollman/pixel-slicer/internal/s3"
)

// Manage configuration

// ReadableConfig is a configuration struct that can be read from and stored to a YAML file, and be configured
// via terminal parameters. It contains all configuration objects, so shouldn't be accessed directly.
// TODO: Can this be made internal?
// TODO: Map FSConfig and MediaConfig with anon structs?
type ReadableConfig struct {
	InputDir            string
	OutputDir           string
	MoveProcessed       bool
	ProcessedDir        string
	Watch               bool
	S3Config            s3.S3Config `mapstructure:"S3"`
	ImageConfigurations []*mediaprocessor.ImageConfiguration
	VideoConfigurations []*mediaprocessor.VideoConfiguration
}

func (c ReadableConfig) GetFSConfig() *mediaprocessor.FSConfig {
	return &mediaprocessor.FSConfig{
		InputDir:      c.InputDir,
		OutputDir:     c.OutputDir,
		MoveProcessed: c.MoveProcessed,
		ProcessedDir:  c.ProcessedDir,
		Watch:         c.Watch,
	}
}

func (c ReadableConfig) GetMediaConfig() *mediaprocessor.MediaConfig {
	return &mediaprocessor.MediaConfig{
		ImageConfigurations: c.ImageConfigurations,
		VideoConfigurations: c.VideoConfigurations,
	}
}

// ValidateConfig validates that a given config is valid
func (c ReadableConfig) ValidateConfig() (err error) {
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

	if c.InputDir == "" {
		return fmt.Errorf("No input dir supplied")
	}

	if c.OutputDir == "" {
		return fmt.Errorf("No output dir supplied")
	}

	if c.Watch && !c.MoveProcessed {
		return fmt.Errorf("--watch requires --move-processed to be enabled, to avoid files being processed multiple times")
	}

	return
}

// Read from config file with Viper?
