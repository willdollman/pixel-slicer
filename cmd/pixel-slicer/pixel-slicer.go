package main

import (
	"fmt"
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
	"github.com/willdollman/pixel-slicer/internal/config"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer"
	"github.com/willdollman/pixel-slicer/internal/s3"
)

func main() {
	app := &cli.App{
		Name:  "pixel-slicer",
		Usage: "Media resizing and uploading",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Usage: "location of config file"},
			&cli.StringFlag{Name: "dir", Usage: "directory to process"},
			&cli.StringFlag{Name: "outputdir", Usage: "directory to output files to"},
			&cli.BoolFlag{Name: "move-processed", Usage: "whether to move files to a separate directory once processed"},
			&cli.StringFlag{Name: "processeddir", Usage: "directory to move files to once they have been processed"},
			&cli.BoolFlag{Name: "enable-s3", Usage: "Enable S3 upload, if configured"},
			&cli.BoolFlag{Name: "sample-config", Usage: "Write a sample config file to example-config.yaml, including any supplied modifications"},
			&cli.BoolFlag{Name: "print-config", Usage: "Print the current configuration and exit"},
			&cli.BoolFlag{Name: "watch", Usage: "Watch the input directory for new files"},
			// TODO: Allow num workers to be passed as a parameter
		},
		Action: func(c *cli.Context) error {
			fmt.Println("Ready to go")

			// Pass cli params to Viper
			// TODO: Consider switching cli -> Cobra as an experiment
			if inputDir := c.String("dir"); inputDir != "" {
				viper.Set("InputDir", inputDir)
			}
			if outputDir := c.String("outputdir"); outputDir != "" {
				viper.Set("OutputDir", outputDir)
			}
			if processedDir := c.String("processeddir"); processedDir != "" {
				viper.Set("ProcessedDir", processedDir)
			}
			if moveProcessed := c.Bool("move-processed"); moveProcessed {
				viper.Set("MoveProcessed", moveProcessed)
			}
			if watch := c.Bool("watch"); watch {
				viper.Set("Watch", watch)
			}
			if s3Enabled := c.Bool("enable-s3"); s3Enabled {
				viper.Set("S3Enabled.Enabled", s3Enabled) // TODO: Does this work after changing ReadableConfig?
			}

			// Read config file
			configPath := c.String("config")
			conf, err := config.GetConfig(configPath)
			if err != nil {
				log.Fatal("Unable to read config file:", err)
			}

			// If requested, print config and exit
			if c.Bool("print-config") {
				spew.Dump(conf)
				os.Exit(0)
			}

			// Validation
			if err := conf.ValidateConfig(); err != nil {
				log.Fatal("Configuration is not valid:", err)
			}

			// If requested, write sample config and exit
			if c.Bool("sample-config") {
				configPath := "sample-config.yaml"
				viper.WriteConfigAs(configPath)
				log.Printf("Wrote sample config to '%s'", configPath)
				os.Exit(0)
			}

			p := &pixelslicer.PixelSlicer{
				S3Client:    s3.NewClient(conf.S3Config),
				FSConfig:    conf.GetFSConfig(),
				MediaConfig: conf.GetMediaConfig(),
			}

			p.ProcessFiles(*conf)

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
