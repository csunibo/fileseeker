package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"golang.org/x/exp/slog"
	"golang.org/x/net/webdav"

	"github.com/csunibo/fileseeker/fs"
)

type configType []struct {
	Years []struct {
		Teachings []struct {
			Url string `json:"url"`
		} `json:"teachings"`
	} `json:"years"`
}

var serverStart = time.Now()

func main() {

	const basePath = "https://csunibo.github.io/"

	var config configType
	configStr, err := os.ReadFile("config/courses.json")
	if err != nil {
		fmt.Println("Error reading config file:", err)
		os.Exit(1)
	}

	err = json.Unmarshal(configStr, &config)
	if err != nil {
		fmt.Println("Error parsing config file:", err)
		os.Exit(1)
	}

	teachings := make([]string, 0, len(config))
	for _, course := range config {
		for _, year := range course.Years {
			for _, teaching := range year.Teachings {
				url := teaching.Url
				teachings = append(teachings, url)
				slog.Info("creating handle for url", "url", url)
				http.Handle("/"+url+"/", &webdav.Handler{
					Prefix:     "/" + url,
					FileSystem: fs.NewStatikFS(basePath + url),
					LockSystem: webdav.NewMemLS(),
				})
			}
		}

	}

	slog.Info("creating handle for /", "url", "/")
	http.Handle("/", &webdav.Handler{
		FileSystem: listFS(teachings),
		LockSystem: webdav.NewMemLS(),
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}

	slog.Info("starting server", "addr", addr)
	err = http.ListenAndServe(addr,
		handlers.CombinedLoggingHandler(os.Stdout, http.DefaultServeMux))
	if err != nil {
		slog.Error("error while serving", "err", err)
		os.Exit(1)
	}
}
