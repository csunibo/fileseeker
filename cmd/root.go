package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/spf13/cobra"
	"golang.org/x/net/webdav"

	"github.com/csunibo/fileseeker/fs"
	"github.com/csunibo/fileseeker/listfs"
	"github.com/csunibo/fileseeker/telemetry"
)

type configType []struct {
	Years []struct {
		Teachings []struct {
			Url string `json:"url"`
		} `json:"teachings"`
	} `json:"years"`
}

const (
	serviceName = "fileseeker"
	serviceVer  = "0.1.0"
)

var (
	RootCmd = &cobra.Command{
		Use:   serviceName,
		Short: "a webdav proxy for csunibo",
		Long: "a webdav server that serves files " +
			"from a statik.json file tree, as produced by statik.",
		Run: Execute,
	}
	configFile   string
	grpcEndpoint string
	grpcSecure   bool
	basePath     string
)

func init() {
	RootCmd.Flags().StringVarP(&configFile, "config", "c", "config/courses.json", "path to config file")
	RootCmd.Flags().StringVar(&grpcEndpoint, "otel", "", "endpoint of the grpc server for OpenTelemetry")
	RootCmd.Flags().BoolVar(&grpcSecure, "otelsecure", false, "use secure connection for OpenTelemetry")

	RootCmd.Flags().StringVarP(&basePath, "basepath", "b", "", "base path for the static files")
	_ = RootCmd.MarkFlagRequired("basepath")
}

func Execute(*cobra.Command, []string) {
	var config configType
	configStr, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		os.Exit(1)
	}

	err = json.Unmarshal(configStr, &config)
	if err != nil {
		fmt.Println("Error parsing config file:", err)
		os.Exit(1)
	}

	// Setup telemetry
	if grpcEndpoint != "" {
		shutdown, err := telemetry.SetupOTelSDK(context.Background(),
			serviceName, serviceVer,
			grpcEndpoint, grpcSecure)
		if err != nil {
			slog.Error("error while setting up telemetry", "err", err)
			os.Exit(1)
		}

		defer func() {
			if err := shutdown(context.Background()); err != nil {
				slog.Error("error while shutting down telemetry", "err", err)
			}
		}()
	}

	mux := http.NewServeMux()

	teachings := make([]string, 0, len(config))
	for _, course := range config {
		for _, year := range course.Years {
			for _, teaching := range year.Teachings {
				url := teaching.Url
				teachings = append(teachings, url)
				slog.Info("creating handle for url", "url", url)
				handleTeaching(mux, url)
			}
		}
	}

	slog.Info("creating handle for /", "url", "/")
	mux.Handle("/", &webdav.Handler{
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

func handleTeaching(mux *http.ServeMux, url string) {
	statikFS, err := fs.NewStatikFS(basePath + url)
	if err != nil {
		slog.Error("error creating statik fs", "err", err, "url", url)
		os.Exit(1)
	}

	handler := &webdav.Handler{
		Prefix:     "/" + url,
		FileSystem: statikFS,
		LockSystem: webdav.NewMemLS(),
	}

	mux.Handle("/"+url+"/", handler)
}
