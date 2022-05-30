package main

import (
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
	"github.com/willdollman/pixel-slicer/internal/config"
	"github.com/willdollman/pixel-slicer/internal/mediaprocessor"
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
			&cli.IntFlag{Name: "workers", Usage: "Number of workers to use for meda processing"},
			&cli.BoolFlag{Name: "debug-filenames", Usage: "Include encoder debug information in generated filenames"},
		},
		Action: func(c *cli.Context) error {
			// Pass cli params to Viper
			// TODO: Consider switching cli -> Cobra
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
			if workers := c.Bool("workers"); workers {
				viper.Set("Workers", workers)
			}
			if s3Enabled := c.Bool("enable-s3"); s3Enabled {
			if debugFilenames := c.Bool("debug-filenames"); debugFilenames {
				viper.Set("DebugFilenames", debugFilenames)
			}

			// Read config file
			configPath := c.String("config")
			conf, err := config.GetConfig(configPath)
			if err != nil {
				log.Fatal("Unable to read config file: ", err)
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
				S3Client:       s3.NewClient(conf.S3Config),
				FSConfig:       conf.GetFSConfig(),
				MediaConfig:    conf.GetMediaConfig(),
				MediaProcessor: mediaprocessor.New(),
			}

			// TODO: Only load libvips when image-libvips module is used
			vips.LoggingSettings(nil, vips.LogLevelWarning)
			vips.Startup(&vips.Config{})
			defer vips.Shutdown()

			p.ProcessFiles(*conf)

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
