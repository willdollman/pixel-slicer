package pixelslicer

import (
	"github.com/willdollman/pixel-slicer/internal/mediaprocessor"
	"github.com/willdollman/pixel-slicer/internal/s3"
)

type PixelSlicer struct {
	S3Client       *s3.S3Client
	FSConfig       *mediaprocessor.FSConfig
	MediaConfig    *mediaprocessor.MediaConfig
	MediaProcessor *mediaprocessor.MediaProcessor
}
