package mediaprocessor

import (
	"fmt"
	"log"

	"github.com/xfrr/goffmpeg/transcoder"
)

// VideoGoffmpeg is based on github.com/xfrr/goffmpeg. This library is fairly robust, but
// doesn't allow custom flags to be passed.
type VideoGoffmpeg struct{}

func (v *VideoGoffmpeg) Thumbnail(m *MediaJob, videoConfig *VideoConfiguration) (err error) {
	outputFilepath := m.OutputPath(videoConfig)

	t := new(transcoder.Transcoder)
	if err = t.Initialize(m.InputFile.Path, outputFilepath); err != nil {
		log.Println("Error initialising video transcoder:", err)
		return
	}

	// Ensure maxWidth is even - required by some codecs
	// TODO: Could do this in ValidateConfig() ?
	maxWidth := videoConfig.MaxWidth
	if maxWidth%2 != 0 {
		maxWidth++
	}

	t.MediaFile().SetVframes(1)
	t.MediaFile().SetSkipAudio(true)
	t.MediaFile().SetSeekTime("0")                                     // Use first frame for thumbnail
	t.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:-2", maxWidth)) // -2 ensures height is a multiple of 2
	t.MediaFile().SetQScale(uint32(videoConfig.Quality))

	done := t.Run(false)
	err = <-done

	return
}

func (v *VideoGoffmpeg) Transcode(m *MediaJob, videoConfig *VideoConfiguration) (err error) {
	outputFilepath := m.OutputPath(videoConfig)

	// Ensure maxWidth is even - required by some codecs
	maxWidth := videoConfig.MaxWidth
	if maxWidth%2 != 0 {
		maxWidth++
	}

	t := new(transcoder.Transcoder)
	if err = t.Initialize(m.InputFile.Path, outputFilepath); err != nil {
		log.Println("Error initialising video transcoder:", err)
		return
	}

	fmt.Printf("Config is %+v\n", videoConfig)

	switch videoConfig.FileType {
	case "mp4":
		t.MediaFile().SetVideoCodec("libx264")                             // TODO: Configure codecs via config
		t.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:-2", maxWidth)) // -2 ensures height is a multiple of 2
		t.MediaFile().SetMovFlags("+faststart")
		t.MediaFile().SetCRF(uint32(videoConfig.Quality))
		t.MediaFile().SetPreset(videoConfig.Preset)
		// t.MediaFile().SetAudioCodec("aac") // unsure if we want to explicitly say - will ffmpeg pick a good default otherwise?
		//t.MediaFile().SetSkipAudio(true) // disable audio - we want to strip audio when generating input file instead
	case "webm":
		t.MediaFile().SetVideoCodec("libvpx-vp9")                          // TODO: Configure codecs via config
		t.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:-2", maxWidth)) // -2 ensures height is a multiple of 2
	case "av1":
		t.MediaFile().SetVideoCodec("libaom-av1")
		t.MediaFile().SetVideoFilter(fmt.Sprintf("scale=%d:-2", maxWidth))
		t.MediaFile().SetCRF(uint32(videoConfig.Quality))
		t.MediaFile().SetVideoBitRate("0")
	default:
		err = fmt.Errorf("Unknown video file type '%s'", videoConfig.FileType)
		return
	}

	fmt.Printf("Writing file to %s\n", outputFilepath)

	done := t.Run(false)
	err = <-done
	return
}
