package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type teachings []struct {
	Url string `json:"url"`
}

type statikNode struct {
	statikDirectory
	Directories []statikDirectory `json:"directories"`
	Files       []statikFile      `json:"files"`
}

type statikDirectory struct {
	Url         string    `json:"url"`
	Time        time.Time `json:"time"`
	GeneratedAt time.Time `json:"generated_at"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	SizeRaw     string    `json:"size"`
}

type statikFile struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Url     string    `json:"url"`
	Mime    string    `json:"mime"`
	SizeRaw string    `json:"size"`
	Time    time.Time `json:"time"`
}

var (
	dataDir = flag.String("d", "data", "data directory")
)

func main() {
	flag.Parse()

	fmt.Println("Starting fileseeker...")

	fmt.Println("Loading teachings...")
	teachingData, err := loadTeachings("config/teachings.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load teachings: %v\n", err)
		os.Exit(1)
	}

	urlQueue := make([]string, 0)

	const rootUrl = "https://csunibo.github.io"

	// enqueue teachings
	for _, teaching := range teachingData {
		url := fmt.Sprintf("%s/%s", rootUrl, teaching.Url)
		urlQueue = append(urlQueue, url)
	}
	fmt.Println("Enqueued teachings", len(urlQueue))

	// walk the tree

	for len(urlQueue) > 0 {
		statikUrl := urlQueue[0]
		urlQueue = urlQueue[1:]

		// get statik.json
		node, err := getStatik(fmt.Sprintf("%s/statik.json", statikUrl))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get statik.json: %v\n", err)
			continue
		}

		// enqueue directories
		for _, d := range node.Directories {
			subUrl := fmt.Sprintf("%s/%s", statikUrl, d.Name)
			urlQueue = append(urlQueue, subUrl)
		}

		// download files
		for _, f := range node.Files {
			time.Sleep(2 * time.Millisecond)

			url := fmt.Sprintf("%s/%s", statikUrl, f.Path)

			path := strings.TrimPrefix(url, rootUrl)
			path = filepath.Join(*dataDir, path)

			fmt.Println("Downloading", url, "to", path)

			// create folder if not exists
			dir := filepath.Dir(path)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				if err := os.MkdirAll(dir, os.ModePerm); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to create directory: %v\n", dir)
					continue
				}
			}

			// create file
			fp, err := os.Create(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create file: %v\n", path)
				continue
			}

			// write file
			resp, err := http.Get(f.Url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to fetch file: %v\n", f.Url)
				continue
			}

			_, err = fp.ReadFrom(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write file: %v\n", path)
				continue
			}

			err = fp.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to close file: %v\n", path)
				continue
			}
		}
	}

}

func loadTeachings(teachingsFile string) (teachings, error) {
	f, err := os.Open(teachingsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", teachingsFile, err)
	}

	var config teachings
	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode file %s: %w", teachingsFile, err)
	}
	return config, nil
}

func getStatik(url string) (statikNode, error) {
	resp, err := http.Get(url)
	if err != nil {
		return statikNode{}, fmt.Errorf("failed to fetch statik.json %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return statikNode{}, fmt.Errorf("failed to fetch statik.json %s: %s", url, resp.Status)
	}

	var statik statikNode
	err = json.NewDecoder(resp.Body).Decode(&statik)
	if err != nil {
		return statikNode{}, fmt.Errorf("failed to decode statik.json: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return statikNode{}, fmt.Errorf("failed to close response body: %w", err)
	}

	return statik, nil
}
