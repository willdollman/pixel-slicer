package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer/config"
	"github.com/willdollman/pixel-slicer/internal/s3"
)

func main() {
	// directory to monitor, config file, output location

	app := &cli.App{
		Name:  "pixel-slicer",
		Usage: "Media resizing and uploading",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "location of config file",
			},
			&cli.StringFlag{
				Name:  "dir",
				Value: "/Users/will/Dropbox/code/pixel-slicer-go/example-simple",
				Usage: "directory to process",
			},
			&cli.StringFlag{
				Name:  "outputdir",
				Value: "output",
				Usage: "directory to output files to",
			},
			&cli.BoolFlag{
				Name:  "move-processed",
				Usage: "whether to move files to a separate directory once processed",
			},
			&cli.StringFlag{
				Name:  "processeddir",
				Value: "processed",
				Usage: "directory to move files to once they have been processed",
			},
			&cli.BoolFlag{
				Name:  "enable-s3",
				Usage: "Enable S3 upload, if configured",
			},
		},
		Action: func(c *cli.Context) error {
			fmt.Println("Ready to go")

			if c.String("dir") == "" {
				log.Fatal("No directory supplied to process")
			}

			// TODO: Load from config file with Viper(?), if no values are passed

			conf := config.Config{
				InputDir:      c.String("dir"),
				OutputDir:     c.String("outputdir"),
				MoveProcessed: c.Bool("move-processed"),
				ProcessedDir:  c.String("processeddir"),
				S3Enabled:     c.Bool("enable-s3"),
				ImageConfigurations: []config.ImageConfiguration{
					{MaxWidth: 500, Quality: 80, FileType: config.JPG},
					{MaxWidth: 2000, Quality: 80, FileType: config.JPG},
				},
				// Thumbnail quality: 2-5 acceptable as an image; 10 borderline, 30 fine if blurred (though 20 probably better quality/size tradeoff)
				VideoConfigurations: []config.VideoConfiguration{
					{MaxWidth: 500, Quality: 2, FileType: config.JPG},  // Thumbnail
					{MaxWidth: 500, Quality: 30, FileType: config.JPG}, // Thumbnail
					//{MaxWidth: 1200, Quality: 23, Preset: "medium", FileType: config.MP4},
					{MaxWidth: 500, Quality: 23, Preset: "ultrafast", FileType: config.MP4},
				},
				S3Config: config.S3Config{
					EndpointURL: "https://s3.us-west-000.backblazeb2.com",
					Region:      "us-east-1",
					Bucket:      "photolog",
				},
			}
			if err := conf.ValidateConfig(); err != nil {
				log.Fatal("Config validation error:", err)
			}
			// TODO: This shouldn't be added to the config - it should be passed as part of a new struct
			// which contains the config, S3Session, FTPSession, etc
			conf.S3Session = s3.S3Session(conf.S3Config)

			pixelslicer.ProcessOneShot(conf)

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
