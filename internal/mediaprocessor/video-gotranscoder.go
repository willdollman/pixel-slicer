package mediaprocessor

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/floostack/transcoder/ffmpeg"
)

type VideoGotranscoder struct{}

func (v *VideoGotranscoder) Thumbnail(m *MediaJob, videoConfig VideoConfiguration) (err error) {

	return
}

func (v *VideoGotranscoder) Transcode(m *MediaJob, videoConfig VideoConfiguration) (err error) {
	// Validate config - ensure maxWidth is even, which is required by some codecs
	if videoConfig.MaxWidth%2 != 0 {
		videoConfig.MaxWidth++
	}

	// Generate ffmpeg options and custom options for first pass
	var opts ffmpeg.Options
	var customOpts CustomOptions
	var secondPass bool
	switch videoConfig.FileType {
	case "mp4":
		opts, customOpts, secondPass = getH264Params(videoConfig)
	case "webm":
		opts, customOpts, secondPass = getWebmParams(m, videoConfig, 1)
	default:
		return fmt.Errorf("unknown video file type '%s'", videoConfig.FileType)
	}

	err = transcodeVideo(m, videoConfig, opts, customOpts)
	if err != nil {
		return err
	}

	// If using a 2-pass codec, perform the second pass
	if secondPass {
		switch videoConfig.FileType {
		case "webm":
			opts, customOpts, _ = getWebmParams(m, videoConfig, 2)
		default:
			return fmt.Errorf("no second pass action configured for file type '%s'", videoConfig.FileType)
		}

		err = transcodeVideo(m, videoConfig, opts, customOpts)
		if err != nil {
			return err
		}
	}

	return
}

// transcodeVideo performs the actual video transcoding, based on passed configuration
func transcodeVideo(m *MediaJob, videoConfig VideoConfiguration, opts ffmpeg.Options, customOpts CustomOptions) (err error) {
	ffmpegConf := &ffmpeg.Config{
		FfmpegBinPath:   "/usr/local/bin/ffmpeg",
		FfprobeBinPath:  "/usr/local/bin/ffprobe",
		ProgressEnabled: true,
	}

	params := strings.Join(opts.GetStrArguments(), " ")
	paramsCustom := strings.Join(customOpts.GetStrArguments(), " ")

	// For debugging
	fmt.Printf("** ffmpeg command is:\n    %s\n    %s\n", params, paramsCustom)

	outputFilepath := m.OutputPath(videoConfig)
	progress, err := ffmpeg.
		New(ffmpegConf).
		Input(m.InputFile.Path).
		Output(outputFilepath).
		WithOptions(opts).
		WithAdditionalOptions(customOpts).
		Start(opts)

	if err != nil {
		log.Fatal(err)
	}

	for msg := range progress {
		log.Printf("%+v", msg)
	}

	return
}

func getH264Params(c VideoConfiguration) (opts ffmpeg.Options, optsCustom CustomOptions, twoPass bool) {
	videoCodec := "libx264"
	overwrite := true
	videoFilter := fmt.Sprintf("scale=%d:-2", c.MaxWidth)
	movFlags := "+faststart"
	// crf := uint32(c.Quality) // There's a bug with uint32 types in transcode

	opts = ffmpeg.Options{
		// OutputFormat: &outputFormat,
		Overwrite:   &overwrite,
		VideoCodec:  &videoCodec,
		VideoFilter: &videoFilter,
		MovFlags:    &movFlags,
		Preset:      &c.Preset,
		// Crf:          &crf, // Currently not working
	}

	// Work around bug in transcode library
	// TODO: Pull request
	crf := c.Quality

	optsCustom = CustomOptions{
		Crf: &crf,
	}

	return opts, optsCustom, false
}

func getWebmParams(m *MediaJob, c VideoConfiguration, pass int) (opts ffmpeg.Options, customOpts CustomOptions, twoPass bool) {
	videoCodec := "libvpx-vp9"
	overwrite := true
	videoFilter := fmt.Sprintf("scale=%d:-2", c.MaxWidth)

	if pass == 1 {
		skipAudio := true

		// First pass
		opts = ffmpeg.Options{
			VideoCodec:  &videoCodec,
			Overwrite:   &overwrite,
			VideoFilter: &videoFilter,
			SkipAudio:   &skipAudio,
		}

		pass := 1
		passLogFile := m.OutputPath(c) + ".log"

		customOpts = CustomOptions{
			Pass:        &pass,
			PassLogFile: &passLogFile,
			Crf:         &c.Quality,
		}
	} else if pass == 2 {
		// Second pass
		opts = ffmpeg.Options{
			VideoCodec:  &videoCodec,
			Overwrite:   &overwrite,
			VideoFilter: &videoFilter,
		}

		pass := 2
		passLogFile := m.OutputPath(c) + ".log"

		customOpts = CustomOptions{
			Pass:        &pass,
			PassLogFile: &passLogFile,
			Crf:         &c.Quality,
		}
	} else {
		log.Fatalf("Unknown pass number")
	}

	return opts, customOpts, true
}

// TODO: Submit PR
type CustomOptions struct {
	Pass        *int    `flag:"-pass"`
	PassLogFile *string `flag:"-passlogfile"`
	Crf         *int    `flag:"-crf"` // Work around bug with *uint32 in ffmpeg.Options
}

func (opts CustomOptions) GetStrArguments() []string {
	f := reflect.TypeOf(opts)
	v := reflect.ValueOf(opts)

	values := []string{}

	for i := 0; i < f.NumField(); i++ {
		flag := f.Field(i).Tag.Get("flag")
		value := v.Field(i).Interface()

		if !v.Field(i).IsNil() {

			if _, ok := value.(*bool); ok {
				values = append(values, flag)
			}

			if vs, ok := value.(*string); ok {
				values = append(values, flag, *vs)
			}

			if va, ok := value.([]string); ok {

				for i := 0; i < len(va); i++ {
					item := va[i]
					values = append(values, flag, item)
				}
			}

			if vm, ok := value.(map[string]interface{}); ok {
				for k, v := range vm {
					values = append(values, k, fmt.Sprintf("%v", v))
				}
			}

			if vi, ok := value.(*int); ok {
				values = append(values, flag, fmt.Sprintf("%d", *vi))
			}

		}
	}

	// fmt.Printf("\nValues are %+v\n\n", values)

	return values
}
