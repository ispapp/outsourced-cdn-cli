package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	"go.mills.io/bitcask/v2"
)

type Image struct {
	Img  string `json:"img"`
	Name string `json:"name"`
}

var UploadCommand = &cli.Command{
	Name:  "upload",
	Usage: "Upload images to Cloudinary",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "input",
			Aliases:  []string{"i"},
			Usage:    "Input JSON file",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "folder",
			Aliases:  []string{"f"},
			Usage:    "Cloudinary folder",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output database folder",
			Value:   "cdn.db",
		},
		&cli.StringFlag{
			Name:  "access",
			Usage: "Cloudinary access URL",
		},
		&cli.BoolFlag{
			Name:  "env",
			Usage: "Use .env file for Cloudinary access URL",
		},
	},
	Action: func(c *cli.Context) error {
		input := c.String("input")
		folder := c.String("folder")
		output := c.String("output")
		access := c.String("access")
		env := c.Bool("env")
		db, err := bitcask.Open(output)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()
		if input == "" || folder == "" || (access == "" && !env) {
			return cli.ShowAppHelp(c)
		}

		var cloudinaryURL string
		if env {
			if err := godotenv.Load(); err != nil {
				log.Fatalf("Error loading .env file")
			}
			cloudinaryURL = os.Getenv("CLOUDINARY_URL")
			if cloudinaryURL == "" {
				log.Fatal("CLOUDINARY_URL not found in .env file")
			}
		} else {
			if !strings.HasPrefix(access, "CLOUDINARY_URL=cloudinary://") {
				log.Fatal("Invalid CLOUDINARY_URL format")
			}
			cloudinaryURL = strings.TrimPrefix(access, "CLOUDINARY_URL=")
		}

		cld, err := cloudinary.NewFromURL(cloudinaryURL)
		if err != nil {
			log.Fatalf("Failed to initialize Cloudinary: %v", err)
		}

		data, err := os.ReadFile(input)
		if err != nil {
			log.Fatalf("Failed to read input file: %v", err)
		}

		var images []Image
		if err := json.Unmarshal(data, &images); err != nil {
			log.Fatalf("Failed to parse JSON: %v", err)
		}

		bar := progressbar.New(len(images))

		for _, img := range images {
			if len(img.Name) < 3 {
				log.Fatalf("Image name must be at least 3 characters long for image %s", img.Img)
			}
			uploadResult, err := cld.Upload.Upload(context.Background(), img.Img, uploader.UploadParams{Folder: folder, PublicID: img.Name})
			if err != nil {
				log.Fatalf("Failed to upload image %s: %v", img.Img, err)
			}
			db.Put(bitcask.Key(strings.ToUpper(img.Name)), bitcask.Value(uploadResult.SecureURL))

			bar.Add(1)
			time.Sleep(100 * time.Millisecond) // Simulate progress
		}

		fmt.Println("Images uploaded db created successfully")
		return nil
	},
}
