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
	ImageConfigurations []*ImageConfiguration
	VideoConfigurations []*VideoConfiguration
}

type MediaConfiguration interface {
	Validate() error              // Return an error if the supplied media configuration is invalid
	OutputFileSuffix(bool) string // Return the file suffix for a given media configuration. e.g. -100px.jpg
}

// ImageConfiguration describes output size, quality, and format for an output image file
type ImageConfiguration struct {
	MaxWidth int
	Quality  int
	FileType FileOutputType
}

func (i *ImageConfiguration) Validate() error {
	if i.Quality > 100 || i.Quality <= 0 {
		return fmt.Errorf("image quality should be between 0 and 100 (%d)", i.Quality)
	}

	switch i.FileType.GetMediaType() {
	case Video:
		return fmt.Errorf("image configuration cannot accept video FileType (%s)", i.FileType)
	case Unknown:
		return fmt.Errorf("unknown media filetype '%s'", i.FileType)

	}
	if i.FileType.GetMediaType() == Unknown {
		return fmt.Errorf("unknown media type '%s'", i.FileType)
	}

	return nil
}

func (i *ImageConfiguration) OutputFileSuffix(simpleName bool) string {
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
	Codec    VideoCodec
}

// Validate validates a VideoConfiguration
func (v *VideoConfiguration) Validate() error {
	if v.Quality <= 0 || v.Quality > 30 {
		return fmt.Errorf("video quality should be between 0 and 30")
	}

	// Apply default video container if applicable
	if v.FileType == "" && validCodecContainer[v.Codec] != "" {
		v.FileType = validCodecContainer[v.Codec]
	}

	switch v.FileType.GetMediaType() {
	case Unknown:
		return fmt.Errorf("unknown media filetype '%s'", v.FileType)
	case Image:
		// TODO: Validate as an ImageConfiguration
	case Video:
		// Apply default codec
		if v.Codec == "" {
			v.Codec = defaultFiletypeCodec[v.FileType]
		}

		// Validate supplied codec
		if err := v.Codec.Validate(); err != nil {
			return err
		}

		// Validate codec-container pairing
		if validCodecContainer[v.Codec] != v.FileType {
			return fmt.Errorf("codec '%s' cannot be used with container '%s' (use %s)", v.Codec, v.FileType, validCodecContainer[v.Codec])
		}

		if v.Codec == H264 && v.Preset == "" {
			v.Preset = "slow"
		}
	}

	// Ensure maxWidth is even - required by some codecs
	if v.MaxWidth%2 != 0 {
		return fmt.Errorf("video width '%d' should be even (required by most codecs)", v.MaxWidth)
	}

	return nil
}

func (v *VideoConfiguration) OutputFileSuffix(simpleName bool) string {
	if v.FileType.GetMediaType() == Image {
		return fmt.Sprintf(".%s", v.FileType)
	}

	if simpleName {
		return fmt.Sprintf("-%d.%s", v.MaxWidth, string(v.FileType))
	}
	return fmt.Sprintf("-%d-q%d-p%s.%s.%s", v.MaxWidth, v.Quality, v.Preset, v.Codec, v.FileType)
}

// FileOutputType is the file extension of the output media file.
// For images, this represents the image format.
// For videos, this represents the container format.
type FileOutputType string

const (
	JPG  FileOutputType = "jpg"
	WebP FileOutputType = "webp"
	MP4  FileOutputType = "mp4"
	WebM FileOutputType = "webm"
)

// This *works*, but is a bit ugly. What if a new FileOutputType is added which doesn't have a type?

// GetMediaType returns the MediaType of a given FileOutputType
func (f FileOutputType) GetMediaType() MediaType {
	switch f {
	case JPG, WebP:
		return Image
	case MP4, WebM:
		return Video
	default:
		return Unknown
	}
}

// VideoCodec represents the codec used to encode a video file, as part of a VideoConfiguration
type VideoCodec string

const (
	H264 VideoCodec = "h264"
	H265 VideoCodec = "h265"
	VP9  VideoCodec = "vp9"
	AV1  VideoCodec = "av1"
)

func (c VideoCodec) Validate() error {
	switch c {
	case H264, H265, VP9, AV1:
		return nil
	default:
		return fmt.Errorf("unknown video codec '%s'", c)
	}
}

// MediaType is the type of a piece of media - Image, Video, etc
type MediaType string

const (
	Image   MediaType = "image"
	Video   MediaType = "video"
	Unknown MediaType = "unknown"
)

// defaultFiletypeCodec maps the default codec for each container format
var defaultFiletypeCodec = map[FileOutputType]VideoCodec{
	MP4:  H264,
	WebM: VP9,
}

// validCodecContainer maps the valid containers for each codec
var validCodecContainer = map[VideoCodec]FileOutputType{
	H264: MP4,
	H265: MP4,
	VP9:  WebM,
	AV1:  WebM,
}
