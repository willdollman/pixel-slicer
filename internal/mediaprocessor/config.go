package mediaprocessor

import "fmt"

// FSConfig contains the filesystem-related parameters used when processing media
type FSConfig struct {
	InputDir      string
	OutputDir     string
	MoveProcessed bool
	ProcessedDir  string
	Watch         bool
}

// MediaConfig contains the image and video output parameters used when encoding media
type MediaConfig struct {
	ImageConfigurations []ImageConfiguration
	VideoConfigurations []VideoConfiguration
}

type MediaConfiguration interface {
	OutputFileSuffix(bool) string // Return the file suffix for a given media configuration. e.g. -100px.jpg
}

// ImageConfiguration describes output size, quality, and format for an output image file
type ImageConfiguration struct {
	MaxWidth int
	Quality  int
	FileType FileOutputType
}

func (i ImageConfiguration) OutputFileSuffix(simpleName bool) string {
	if simpleName {
		return fmt.Sprintf("-%d.%s", i.MaxWidth, string(i.FileType))
	}

	return fmt.Sprintf("x%d.%s", i.MaxWidth, string(i.FileType))
	// return fmt.Sprintf("-%d-q%d.%s", i.MaxWidth, i.Quality, string(i.FileType))
}

// VideoConfiguration describes output size, quality, format, and other information for an encoded
// output video file
type VideoConfiguration struct {
	MaxWidth int
	Quality  int
	Preset   string
	FileType FileOutputType
}

func (i VideoConfiguration) OutputFileSuffix(simpleName bool) string {
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
	VP9     FileOutputType = "webm"
	AV1     FileOutputType = "webmxxx"
)

// This *works*, but is a bit ugly. What if a new FileOutputType is added which doesn't have a type?

// GetMediaType returns the MediaType of a given FileOutputType
func (f FileOutputType) GetMediaType() MediaType {
	switch f {
	case JPG, WebP, WebPBin:
		return Image
	case MP4, VP9, AV1:
		return Video
	default:
		return Unknown
	}
}

// MediaType is the type of a piece of media - Image, Video, etc
type MediaType string

const (
	Image   MediaType = "image"
	Video   MediaType = "video"
	Unknown MediaType = "unknown"
)
