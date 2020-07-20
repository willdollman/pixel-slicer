package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

func GetConfig(configPath string) (*Config, error) {
	// Default configuration
	viper.SetDefault("InputDir", "input")
	viper.SetDefault("OutputDir", "output")
	viper.SetDefault("ProcessedDir", "processed")
	viper.SetDefault("MoveProcessed", false)
	// Default S3 configurations
	viper.SetDefault("S3Enabled", false)
	viper.SetDefault("S3", map[string]string{"Endoint": "", "Region": "", "Bucket": "pixelslicer"})
	// Set default media configurations
	viper.SetDefault("ImageConfiguration", []map[string]string{{"MaxWidth": "1000", "Quality": "80", "FileType": "jpg"}})
	viper.SetDefault("VideoConfiguration", []map[string]string{{"MaxWidth": "600", "Quality": "23", "Preset": "ultrafast", "FileType": "mp4"}})

	// Config location
	if configPath != "" {
		fmt.Println("Loading config from supplied path:", configPath)
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

	var appConfig Config

	if err := viper.Unmarshal(&appConfig); err != nil {
		log.Fatal("Error unmarshalling config")
		return nil, err
	}

	return &appConfig, nil
}
