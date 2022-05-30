package config

import (
	"log"
	"runtime"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/willdollman/pixel-slicer/internal/mediaprocessor"
)

func GetConfig(configPath string) (*ReadableConfig, error) {
	// Default configuration
	viper.SetDefault("InputDir", "input")
	viper.SetDefault("OutputDir", "output")
	viper.SetDefault("ProcessedDir", "processed")
	viper.SetDefault("MoveProcessed", false)
	viper.SetDefault("Watch", false)
	viper.SetDefault("Workers", runtime.NumCPU()/2) // Base worker threads on number of CPU cores available
	// Default S3 configurations
	viper.SetDefault("S3Enabled", false)
	viper.SetDefault("S3", map[string]string{"Endoint": "", "Region": "", "Bucket": "pixelslicer"})
	// Set default media configurations
	viper.SetDefault("ImageConfigurations", []*mediaprocessor.ImageConfiguration{
		{MaxWidth: 500, Quality: 80, FileType: mediaprocessor.FileOutputType("jpg")},
		{MaxWidth: 500, Quality: 80, FileType: mediaprocessor.FileOutputType("webp")},
		{MaxWidth: 2000, Quality: 80, FileType: mediaprocessor.FileOutputType("jpg")},
		{MaxWidth: 2000, Quality: 80, FileType: mediaprocessor.FileOutputType("webp")},
	})
	viper.SetDefault("VideoConfigurations", []*mediaprocessor.VideoConfiguration{
		{MaxWidth: 480, Quality: 23, FileType: mediaprocessor.FileOutputType("mp4")},
		{MaxWidth: 720, Quality: 23, FileType: mediaprocessor.FileOutputType("mp4")},
	})

	// Config location
	if configPath != "" {
		// fmt.Printf("Loading config from '%s'\n", configPath) // TODO: verbose
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("$HOME/.pixel-slicer") // Look for the config file in ~/.pixel-slicer/
		viper.AddConfigPath(".")                   // Look for the config file in the working directory
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("No config file found")
		} else {
			// Config file was found but another error was returned
			log.Fatal("Error loading config file:", err)
		}
	}

	var appConfig ReadableConfig

	if err := viper.Unmarshal(&appConfig); err != nil {
		log.Fatal("Error unmarshalling config")
		return nil, err
	}

	// Validate media configs
	for _, c := range appConfig.ImageConfigurations {
		if err := c.Validate(); err != nil {
			return nil, errors.Wrap(err, "invalid image configuration")
		}
	}
	for _, c := range appConfig.VideoConfigurations {
		if err := c.Validate(); err != nil {
			return nil, errors.Wrap(err, "invalid video configuration")
		}
	}

	return &appConfig, nil
}
