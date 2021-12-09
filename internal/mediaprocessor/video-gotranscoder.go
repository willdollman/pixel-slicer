package mediaprocessor

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/floostack/transcoder/ffmpeg"
)

// VideoGotranscoder is based on github.com/floostack/transcoder.
// This library allows custom ffmpeg commands to be passed, which permits two-pass
// codecs such as VP9.
type VideoGotranscoder struct{}

func (v *VideoGotranscoder) Thumbnail(m *MediaJob, videoConfig *VideoConfiguration) (err error) {
	vFrames := 1
	skipAudio := true
	seekTime := "0"
	videoFilter := fmt.Sprintf("scale=%d:-2", videoConfig.MaxWidth)

	opts := ffmpeg.Options{
		Vframes:     &vFrames,
		SkipAudio:   &skipAudio,
		SeekTime:    &seekTime,
		VideoFilter: &videoFilter,
	}
	customOpts := CustomOptions{
		QScaleVideo: &videoConfig.Quality,
	}

	err = transcodeVideo(m, videoConfig, opts, customOpts)
	if err != nil {
		return err
	}

	return
}

func (v *VideoGotranscoder) Transcode(m *MediaJob, videoConfig *VideoConfiguration) (err error) {
	// Validate config - ensure maxWidth is even, which is required by some codecs
	if videoConfig.MaxWidth%2 != 0 {
		videoConfig.MaxWidth++
	}

	// Generate ffmpeg options and custom options for first pass
	var opts ffmpeg.Options
	var customOpts CustomOptions
	var secondPass bool
	switch videoConfig.Codec {
	case H264:
		opts, customOpts, secondPass = getH264Params(videoConfig)
	case H265:
		opts, customOpts, secondPass = getH265Params(videoConfig)
	case VP9:
		opts, customOpts, secondPass = getVp9Params(m, videoConfig, 1)
	case AV1:
		opts, customOpts, secondPass = getAv1Params(m, videoConfig, 1)
	default:
		return fmt.Errorf("unknown codec type '%s'", videoConfig.Codec)
	}

	err = transcodeVideo(m, videoConfig, opts, customOpts)
	if err != nil {
		return err
	}

	// If using a 2-pass codec, perform the second pass
	if secondPass {
		switch videoConfig.Codec {
		case VP9:
			opts, customOpts, _ = getVp9Params(m, videoConfig, 2)
		case AV1:
			opts, customOpts, _ = getAv1Params(m, videoConfig, 2)
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
func transcodeVideo(m *MediaJob, videoConfig *VideoConfiguration, opts ffmpeg.Options, customOpts CustomOptions) (err error) {
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

	for range progress {
		// for msg := range progress {
		// log.Printf("%+v", msg)
	}

	return
}

/*
getH264Params provides ffmpeg parameters for h264 encoding.
https://trac.ffmpeg.org/wiki/Encode/H.264

	* 1-pass encoding
	* Preset can be selected (default 'slow')
*/
func getH264Params(c *VideoConfiguration) (opts ffmpeg.Options, optsCustom CustomOptions, twoPass bool) {
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

/*
getH265Params provides ffmpeg parameters for h265 encoding.
https://trac.ffmpeg.org/wiki/Encode/H.265

	* 1-pass encoding
	* Preset can be selected (default 'slow')
*/
func getH265Params(c *VideoConfiguration) (opts ffmpeg.Options, optsCustom CustomOptions, twoPass bool) {
	videoCodec := "libx265"
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

	crf := c.Quality

	optsCustom = CustomOptions{
		Crf: &crf,
	}

	return opts, optsCustom, false
}

/*
getVp9Params provides ffmpeg parameters for VP9 encoding.
https://trac.ffmpeg.org/wiki/Encode/VP9

	* 2-pass encoding recommended
	* TODO: Does -b:v 0 need to be set?
*/
func getVp9Params(m *MediaJob, c *VideoConfiguration, pass int) (opts ffmpeg.Options, customOpts CustomOptions, twoPass bool) {
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

/*
getAv1Params provides ffmpeg parameters for AV1 encoding.
https://trac.ffmpeg.org/wiki/Encode/AV1

	* Performs 2-pass encoding as this may help encoding efficiency - need to verify
	* -cpu-used 8 minimises CPU load at the slight expense of quality; worth it as AV1 is expensive
	* libopus audio codec
*/
func getAv1Params(m *MediaJob, c *VideoConfiguration, pass int) (opts ffmpeg.Options, customOpts CustomOptions, twoPass bool) {
	videoCodec := "libaom-av1"
	audioCodec := "libopus"
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
		cpuUsed := 8

		customOpts = CustomOptions{
			Pass:        &pass,
			PassLogFile: &passLogFile,
			Crf:         &c.Quality,
			CpuUsed:     &cpuUsed,
		}
	} else if pass == 2 {
		// Second pass
		opts = ffmpeg.Options{
			VideoCodec:  &videoCodec,
			AudioCodec:  &audioCodec,
			Overwrite:   &overwrite,
			VideoFilter: &videoFilter,
		}

		pass := 2
		passLogFile := m.OutputPath(c) + ".log"
		cpuUsed := 8 // TODO: Tune?

		customOpts = CustomOptions{
			Pass:        &pass,
			PassLogFile: &passLogFile,
			Crf:         &c.Quality,
			CpuUsed:     &cpuUsed,
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
	Crf         *int    `flag:"-crf"`      // Work around bug with *uint32 in ffmpeg.Options
	CpuUsed     *int    `flag:"-cpu-used"` // Used with AV1 codec
	QScaleVideo *int    `flag:"-qscale:v"` // Used for thumbnails
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
