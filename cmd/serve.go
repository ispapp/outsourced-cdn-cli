package cmd

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/fogleman/gg"
	"github.com/urfave/cli/v2"
	"go.mills.io/bitcask/v2"
)

var ServeCommand = &cli.Command{
	Name:  "serve",
	Usage: "Serve images and proxy output JSON with fast search",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "input",
			Aliases:  []string{"i"},
			Usage:    "Input db folder (cdn.db: bitcask)",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "Port to serve on",
			Value:   "8080",
		},
		&cli.StringFlag{
			Name:    "notfound",
			Aliases: []string{"nf"},
			Usage:   "Path to custom 'not found' image",
		},
	},
	Action: func(c *cli.Context) error {
		input := c.String("input")
		port := c.String("port")
		notFoundImage := c.String("notfound")
		db, err := bitcask.Open(input, bitcask.WithOpenReadonly(true), bitcask.WithAutoRecovery(false))
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()
		http.HandleFunc("/img", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("q")
			params := r.URL.Query().Get("p")
			if params == "" {
				params = "c_fit,h_300,w_300"
			}
			if query == "" {
				http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
				return
			}
			var found []bitcask.Value
			err = db.Scan(bitcask.Key(strings.ToUpper(query)), bitcask.KeyFunc(func(k bitcask.Key) error {
				_found, err := db.Get(k)
				found = append(found, _found)
				if err != nil {
					return err
				}
				return nil
			}))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(found) == 0 {
				if notFoundImage != "" {
					http.ServeFile(w, r, notFoundImage)
				} else {
					img := image.NewRGBA(image.Rect(0, 0, 300, 300))
					blue := color.RGBA{255, 255, 255, 0}
					draw.Draw(img, img.Bounds(), &image.Uniform{blue}, image.Point{}, draw.Src)
					dc := gg.NewContextForRGBA(img)
					dc.SetColor(color.Black)
					dc.DrawStringAnchored("Not Found", 150, 150, 0.5, 0.5)
					w.Header().Set("Content-Type", "image/png")
					jpeg.Encode(w, img, nil)
				}
				return
			}
			uriparts := strings.Split(string(found[0]), "/upload/")
			uri := fmt.Sprintf("%s/upload/%s/%s", uriparts[0], params, uriparts[1])
			resp, err := http.Head(uri)
			if err != nil || resp.StatusCode != http.StatusOK {
				http.Error(w, "Invalid image URL or image not accessible", http.StatusBadGateway)
				return
			}
			// http.Redirect(w, r, string(uri), http.StatusFound)
			resp, err = http.Get(uri)
			if err != nil || resp.StatusCode != http.StatusOK {
				http.Error(w, "Failed to fetch image", http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()
			w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
			w.WriteHeader(resp.StatusCode)
			if _, err := io.Copy(w, resp.Body); err != nil {
				http.Error(w, "Failed to serve image", http.StatusInternalServerError)
			}
		})

		fmt.Printf("Serving on port %s\n", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}

		return nil
	},
}
