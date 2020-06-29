package config

// Manage configuration

type PixelSlicerConfig struct {
	InputDir            string
	OutputDir           string
	ImageConfigurations []ImageConfiguration
}

type ImageConfiguration struct {
	MaxWidth int
	Quality  int
}

// Read from config file with Viper?
