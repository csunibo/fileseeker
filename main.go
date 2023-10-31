package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"golang.org/x/net/webdav"

	"github.com/csunibo/fileseeker/fs"
	"github.com/csunibo/fileseeker/listfs"
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

				statikFS, err := fs.NewStatikFS(basePath + url)
				if err != nil {
					slog.Error("error creating statik fs", "err", err, "url", url)
					os.Exit(1)
				}

				http.Handle("/"+url+"/", &webdav.Handler{
					Prefix:     "/" + url,
					FileSystem: statikFS,
					LockSystem: webdav.NewMemLS(),
				})
			}
		}

	}

	slog.Info("creating handle for /", "url", "/")
	http.Handle("/", &webdav.Handler{
		FileSystem: listfs.NewListFS(teachings),
		LockSystem: webdav.NewMemLS(),
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}

	slog.Info("creating logging handler")
	handler := handlers.CombinedLoggingHandler(os.Stdout, http.DefaultServeMux)

	PROXY := os.Getenv("PROXY")
	if PROXY == "true" {
		slog.Warn("proxy handling enabled. If you are not behind a proxy, set PROXY=false")
		handler = handlers.ProxyHeaders(handler)
	}

	slog.Info("starting server", "addr", addr)
	err = http.ListenAndServe(addr, handler)
	if err != nil {
		slog.Error("error while serving", "err", err)
		os.Exit(1)
	}
}
