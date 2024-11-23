package main

import (
	"couldinaryuploader/cmd"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "Image Uploader",
		Usage: "Upload images to Cloudinary",
		Commands: []*cli.Command{
			cmd.UploadCommand,
			cmd.ServeCommand,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
