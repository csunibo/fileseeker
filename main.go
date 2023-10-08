package main

import (
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"golang.org/x/net/webdav"

	"github.com/csunibo/fileseeker/fs"
)

func main() {

	const basePath = "https://csunibo.github.io/"

	handler := &webdav.Handler{
		FileSystem: fs.NewStatikFs(basePath + "programmazione/"),
		LockSystem: webdav.NewMemLS(),
	}

	http.Handle("/", handler)
	err := http.ListenAndServe(":8080",
		handlers.CombinedLoggingHandler(os.Stdout, http.DefaultServeMux))
	if err != nil {
		panic(err)
	}
}
