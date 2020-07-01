package config

// Manage configuration

type Config struct {
	InputDir            string
	OutputDir           string
	ImageConfigurations []ImageConfiguration
}

type ImageConfiguration struct {
	MaxWidth int
	Quality  int
	FileType FileOutputType
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
