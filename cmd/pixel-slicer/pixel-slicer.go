package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer/config"
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
				ImageConfigurations: []config.ImageConfiguration{
					{MaxWidth: 500, Quality: 80, FileType: config.JPG},
					{MaxWidth: 2000, Quality: 80, FileType: config.JPG},
				},
				VideoConfigurations: []config.VideoConfiguration{
					{MaxWidth: 1200, Quality: 23, Preset: "medium", FileType: config.MP4},
				},
			}
			if err := conf.ValidateConfig(); err != nil {
				log.Fatal("Config validation error:", err)
			}

			pixelslicer.ProcessOneShot(conf)

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
