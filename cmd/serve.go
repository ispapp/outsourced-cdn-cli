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
	"os/exec"
	"sort"
	"strings"

	"github.com/fogleman/gg"
	"github.com/urfave/cli/v2"
	"go.mills.io/bitcask/v2"
)

// SearchResult holds a match with its score
type SearchResult struct {
	Key   bitcask.Key
	Value bitcask.Value
	Score int
}

// generateSearchTerms splits a query into progressive parts (30%, 60%, 100%)
func generateSearchTerms(query string) []string {
	query = strings.ToUpper(query)
	length := len(query)
	if length <= 3 {
		return []string{query}
	}

	terms := make([]string, 0, 3)
	// Add 30% of the term
	firstCutoff := int(float64(length) * 0.3)
	if firstCutoff < 1 {
		firstCutoff = 1
	}
	terms = append(terms, query[:firstCutoff])

	// Add 60% of the term
	secondCutoff := int(float64(length) * 0.6)
	if secondCutoff > firstCutoff {
		terms = append(terms, query[:secondCutoff])
	}

	// Add full term
	terms = append(terms, query)

	return terms
}

// calculateScore computes a relevance score between a key and query
func calculateScore(key string, query string) int {
	keyUpper := strings.ToUpper(key)
	queryUpper := strings.ToUpper(query)

	// Check if query contains + signs for multi-term search
	if strings.Contains(queryUpper, "+") {
		terms := strings.Split(queryUpper, "+")
		totalScore := 0
		matchedTerms := 0

		for _, term := range terms {
			term = strings.TrimSpace(term)
			if term == "" {
				continue
			}

			termScore := calculateSingleTermScore(keyUpper, term)
			if termScore > 0 {
				matchedTerms++
				totalScore += termScore
			}
		}

		// If no terms matched, return 0
		if matchedTerms == 0 {
			return 0
		}

		// Find the highest score
		highestScore := 0
		otherScoresTotal := 0

		for _, term := range terms {
			termScore := calculateSingleTermScore(keyUpper, strings.TrimSpace(term))
			if termScore > highestScore {
				otherScoresTotal += highestScore // Add previous highest to other scores
				highestScore = termScore
			} else {
				otherScoresTotal += termScore
			}
		}

		// Calculate final score: top score * 2 + other scores / terms number
		otherScoresAvg := 0
		if matchedTerms > 1 { // Only calculate average if there are other terms besides the highest
			otherScoresAvg = otherScoresTotal / (matchedTerms - 1)
		}

		return (highestScore * 2) + otherScoresAvg
	}

	// Single term search
	return calculateSingleTermScore(keyUpper, queryUpper)
}

// calculateSingleTermScore computes score for a single search term
func calculateSingleTermScore(keyUpper string, queryTerm string) int {
	// Exact match gets highest score
	if keyUpper == queryTerm {
		return 100
	}

	// Prefix match (starts with query)
	if strings.HasPrefix(keyUpper, queryTerm) {
		return 80
	}

	// Contains match
	if strings.Contains(keyUpper, queryTerm) {
		return 60
	}

	// Calculate partial match score based on common prefix length
	commonPrefixLen := 0
	for i := 0; i < len(queryTerm) && i < len(keyUpper); i++ {
		if queryTerm[i] == keyUpper[i] {
			commonPrefixLen++
		} else {
			break
		}
	}

	// Score based on percentage of common prefix
	if commonPrefixLen > 0 {
		return int(float64(commonPrefixLen) / float64(len(queryTerm)) * 50)
	}

	return 0
}

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
			// DefaultText: "./notfound.png",
			Usage: "Path to custom 'not found' image",
		},
		&cli.StringFlag{
			Name:  "cfd-config",
			Usage: "Cloudflare config for proxying the open port",
		},
		&cli.StringFlag{
			Name:  "cfd-uuid",
			Usage: "Cloudflare uuid for proxying the open port",
		},
	},
	Action: func(c *cli.Context) error {
		input := c.String("input")
		port := c.String("port")
		notFoundImage := c.String("notfound")
		config := c.String("cfd-config")
		cfduuid := c.String("cfd-uuid")
		db, err := bitcask.Open(input, bitcask.WithOpenReadonly(true), bitcask.WithAutoRecovery(false))
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		if config != "" && cfduuid != "" {
			cmd := exec.Command("cloudflared", "tunnel --config ", config, " --url http://localhost:"+port, "run ", cfduuid)
			if err := cmd.Start(); err != nil {
				log.Printf("Failed to start cloudflared: %v", err)
			} else {
				log.Println("Cloudflared started successfully")
				log.Printf("cmd: %s", cmd.String())
			}
		} else {
			log.Println("Cloudflare config not provided, skipping cloudflared setup")
		}

		http.HandleFunc("/img", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("q")
			// origin := r.URL.Headers.Get("origin")
			params := r.URL.Query().Get("p")
			if params == "" {
				params = "c_fit,h_300,w_300"
			}
			if query == "" {
				http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
				return
			}

			// Generate search terms based on progressive percentages
			searchTerms := generateSearchTerms(query)

			// Map to store unique results with their scores
			resultsMap := make(map[string]SearchResult)

			// Search using each term with progressive scoring
			for _, term := range searchTerms {
				err = db.Scan(bitcask.Key(term), bitcask.KeyFunc(func(k bitcask.Key) error {
					value, err := db.Get(k)
					if err != nil {
						return nil // Skip this result but continue scanning
					}

					keyStr := string(k)
					score := calculateScore(keyStr, query)

					// Only update if new score is higher or entry doesn't exist
					keyHash := string(k)
					existing, exists := resultsMap[keyHash]
					if !exists || score > existing.Score {
						resultsMap[keyHash] = SearchResult{
							Key:   k,
							Value: value,
							Score: score,
						}
					}

					return nil
				}))
				if err != nil {
					log.Printf("Error scanning with term %s: %v", term, err)
				}
			}

			// Convert map to slice and sort by score
			var results []SearchResult
			for _, result := range resultsMap {
				results = append(results, result)
			}

			sort.Slice(results, func(i, j int) bool {
				return results[i].Score > results[j].Score
			})

			// Handle case when no results found
			if len(results) == 0 {
				if notFoundImage != "" {
					w.Header().Set("Content-Type", "image/png")
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

			// Use the best match (highest score)
			bestMatch := results[0]

			uriparts := strings.Split(string(bestMatch.Value), "/upload/")
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
