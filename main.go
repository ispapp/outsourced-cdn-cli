package main

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
)

type Image struct {
	Img  string `json:"img"`
	Name string `json:"name"`
}

var bannedWords = []string{"banned1", "banned2"}

func containsBannedWord(name string) bool {
	for _, word := range bannedWords {
		if strings.Contains(name, word) {
			return true
		}
	}
	return false
}

func main() {
	app := &cli.App{
		Name:  "Image Uploader",
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
				Usage:   "Output JSON file",
				Value:   "output.json",
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

			for i, img := range images {
				if containsBannedWord(img.Name) {
					log.Fatalf("Image name contains banned word: %s", img.Name)
				}

				uploadResult, err := cld.Upload.Upload(context.Background(), img.Img, uploader.UploadParams{Folder: folder})
				if err != nil {
					log.Fatalf("Failed to upload image %s: %v", img.Img, err)
				}
				images[i].Img = uploadResult.SecureURL

				bar.Add(1)
				time.Sleep(100 * time.Millisecond) // Simulate progress
			}

			outputData, err := json.Marshal(images)
			if err != nil {
				log.Fatalf("Failed to marshal output JSON: %v", err)
			}

			if err := os.WriteFile(output, outputData, 0644); err != nil {
				log.Fatalf("Failed to write output file: %v", err)
			}

			fmt.Println("Images uploaded and JSON file created successfully")
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
