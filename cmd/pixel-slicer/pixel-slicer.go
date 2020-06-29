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
				Value: "../pixel-slicer/example-simple",
				Usage: "directory to process",
			},
			&cli.StringFlag{
				Name:  "outputdir",
				Value: "output",
				Usage: "directory to output files to",
			},
		},
		Action: func(c *cli.Context) error {
			fmt.Println("Ready to go")

			if c.String("dir") == "" {
				log.Fatal("No directory supplied to process")
			}

			// TODO: Load from config file with Viper(?), if no values are passed

			conf := config.PixelSlicerConfig{
				InputDir:  c.String("dir"),
				OutputDir: c.String("outputdir"),
				ImageConfigurations: []config.ImageConfiguration{
					{100, 80},
					{500, 80},
					{1000, 80},
					{2000, 80},
				},
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
