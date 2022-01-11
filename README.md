# pixel-slicer

A fast and multithreaded media encoder built for the web.

## Overview

pixel-slicer is an image and video encoder written in [Go](https://go.dev/).
It's designed to optimise your web media for every possible browser and screen size, upload your media wherever it needs to go, and to do it all as quickly as possible.

pixel-slicer uses the extremely efficient [libvips](https://github.com/libvips/libvips) for image encoding and [FFmpeg](https://github.com/FFmpeg/FFmpeg) for video encoding, though it's designed to be extendable and can work with any media encoder.

pixel-slicer's goal is to help you optimise media-heavy websites by encoding resources at multiple resolutions and formats, so the smallest file can be served to each client tuned for screen size and supported formats.
It was written to allow high performance sites packed with media to be served on a shoestring budget with pro performance - I wrote pixel-slicer as a backend processor for my own static [photolog site](https://photos.dollman.org/).

## Getting Started
### Docker

The easiest way to run pixel-slicer is via Docker, which includes all required media encoding libraries:

```
git clone git@github.com:willdollman/pixel-slicer.git
docker build -t px:latest .
...
```

### Downloading binaries and building from source

pixel-slicer relies on various media encoding libraries, which must be installed before it can use them.

```
# macOS (using Homebrew)
brew install vips ffmpeg

# Linux
apt-get install libvips-dev ffmpeg
```

Once installed, you can download the pre-built binaries:

> TODO: Create pre-built binaries

Alternatively, if you have Go installed you can download and build from source:

```
go install github.com/willdollman/pixel-slicer/cmd/...@latest

# Or clone and run
git clone git@github.com:willdollman/pixel-slicer.git
cd pixel-slicer
go run cmd/pixel-slicer/main.go
```

If installing from source fails on macOS, try setting `CGO_CFLAGS_ALLOW=-Xpreprocessor` to work around an issue in the macOS libvips library.

## Configuration

pixel-slicer configuration is stored as YAML, and contains conversion rules for each media type:

<details>
  <summary>Sample Config</summary>

```
inputDir: sample-data/
outputDir: output-media/
moveProcessed: false
watch: false

# Upload all generated media to S3-compatible storage
S3:
  Enabled: true
  Endpoint: https://s3.us-west-000.backblazeb2.com
  Region: us-east-1
  Bucket: pixel-slicer

# Convert images to JPG and WebP at a variety of sizes
ImageConfigurations:
  - MaxWidth: 500
    Quality: 80
    FileType: jpg
  - MaxWidth: 500
    Quality: 80
    FileType: webp

  - MaxWidth: 1000
    Quality: 75
    FileType: jpg
  - MaxWidth: 1000
    Quality: 80
    FileType: webp

  - MaxWidth: 2000
    Quality: 70
    FileType: jpg
  - MaxWidth: 2000
    Quality: 75
    FileType: webp

# Convert videos to H.264 and AV1 at two sizes, and include a JPG thumbnail
VideoConfigurations:
  - MaxWidth: 500
    Quality: 2
    FileType: jpg

  - MaxWidth: 360
    Quality: 23
    Preset: slow
    FileType: mp4
  - MaxWidth: 360
    Quality: 40
    Codec: av1

  - MaxWidth: 720
    Quality: 23
    Preset: slow
    FileType: mp4
  - MaxWidth: 720
    Quality: 40
    Codec: av1
  ```
</details>

YAML config files can be passed with `--config <file.yaml>`. All config file options can be overridden by command line flags.

Useful configuration flags:

* `--watch`: watch the input directory for new files
* `--move-processed`: move processed files to a separate directory. Useful when used with `--watch`
* See `--help` for a full list


## Supported Output Formats

pixel-slicer is designed for the web, and out-of-the-box it supports a carefully considered selection of widely-supported and more modern high-efficiency formats.

| Format | Media Type | Browser Support  | Description |
|--------|------------|----------|-------------|
| JPG    | Image      | [Universal](https://caniuse.com/jpg) | Supported everywhere, outdated efficiency |
| WebP   | Image      | [Modern, good](https://caniuse.com/webp) | 25-34% smaller than JPG|
| H.264  | Video      | [Universal](https://caniuse.com/mpeg4) | Supported everywhere, outdated efficiency |
| H.265  | Video      | [Poor; Apple platforms only](https://caniuse.com/?search=h265) | ~30-50% efficiency gain over H.264, not open source |
| VP9    | Video      | [Medium; modern browsers excluding Apple](https://caniuse.com/?search=vp9) | ~30-50% efficiency gain over H.264, open source |
| AV1    | Video      | [Medium; modern browsers excluding Apple](https://caniuse.com/?search=av1) | Successor to VP9; slow encoding speeds|

By making multiple forms of an image or video available [using source sets](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/source), a browser can select the most appropriate filetype to use.
