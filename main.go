package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/gorilla/handlers"
	"golang.org/x/net/webdav"
)

// StatikFs represents a virtual filesystem that is backed by a statik.json files in a remote server.
//
// cache is a map from path to Statik, where path is the path of the directory containing the statik.json file.
// cacheLock is a RW mutex that protects cache. It is used by getOrCacheStatik and cacheStatik.
type StatikFs struct {
	base string

	cache     map[string]Statik
	cacheLock sync.RWMutex
}

func NewStatikFs(base string) *StatikFs {
	return &StatikFs{
		base:      base,
		cacheLock: sync.RWMutex{},
		cache:     map[string]Statik{},
	}
}

func (m *StatikFs) cacheStatik(path string) (Statik, error) {
	if cache, ok := m.cache[path]; ok {
		return cache, nil
	}

	response, err := http.Get(m.base + path + "/statik.json")
	if err != nil {
		return Statik{}, fmt.Errorf("error getting statik.json: %w", err)
	}

	statik := Statik{}
	err = json.NewDecoder(response.Body).Decode(&statik)
	if err != nil {
		return Statik{}, fmt.Errorf("error decoding statik.json: %w", err)
	}

	m.cacheLock.Lock()
	m.cache[path] = statik
	m.cacheLock.Unlock()

	return statik, nil
}

func (m *StatikFs) getOrCacheStatik(path string) (Statik, error) {
	m.cacheLock.RLock()
	statik, ok := m.cache[path]
	m.cacheLock.RUnlock()

	if ok {
		return statik, nil
	} else {
		return m.cacheStatik(path)
	}
}

func (m *StatikFs) Mkdir(_ context.Context, _ string, _ os.FileMode) error {
	// If fs.ErrPermission is used, gvfs retries the operation forever
	return fs.ErrNotExist
}
func (m *StatikFs) RemoveAll(_ context.Context, _ string) error { return fs.ErrPermission }
func (m *StatikFs) Rename(_ context.Context, _, _ string) error { return fs.ErrPermission }

func (m *StatikFs) OpenFile(
	ctx context.Context,
	name string,
	flag int,
	perm os.FileMode,
) (webdav.File, error) {

	// only allow read-only access
	if flag != os.O_RDONLY {
		return nil, fs.ErrPermission
	}

	statikPath := path.Dir(name)
	statik, err := m.getOrCacheStatik(statikPath)
	if err != nil {
		slog.ErrorContext(ctx, "requested statik.json for a path that doesn't exist",
			"path", statikPath, "err", err)
		return nil, fs.ErrNotExist
	}

	if strings.HasSuffix(name, "/") {
		// we're opening a dir
		return statik, nil
	}

	name = path.Base(name)
	name = strings.TrimPrefix(name, "/")

	// we're opening a file
	for _, file := range statik.Files {
		if file.Name() == name {
			return newInMemFile(file)
		}
	}

	// we're opening a dir (??)
	for _, dir := range statik.Directories {
		if dir.Name() == name {
			redir := statikPath + "/" + name + "/"
			return m.OpenFile(ctx, redir, flag, perm)
		}
	}

	return nil, fs.ErrNotExist
}

func (m *StatikFs) Stat(_ context.Context, name string) (os.FileInfo, error) {
	statikPath := path.Dir(name)

	statik, err := m.getOrCacheStatik(statikPath)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(name, "/") {
		// we're opening a dir
		return statik, nil
	}

	name = path.Base(name)
	name = strings.TrimPrefix(name, "/")

	// we're opening a file
	for _, file := range statik.Files {
		if file.Name() == name {
			return file, nil
		}
	}

	for _, dir := range statik.Directories {
		if dir.Name() == name {
			return dir, nil
		}
	}

	return nil, fs.ErrNotExist
}

func main() {

	const basePath = "https://csunibo.github.io/"

	handler := &webdav.Handler{
		FileSystem: NewStatikFs(basePath + "programmazione/"),
		LockSystem: webdav.NewMemLS(),
	}

	http.Handle("/", handler)
	err := http.ListenAndServe(":8080",
		handlers.CombinedLoggingHandler(os.Stdout, http.DefaultServeMux))
	if err != nil {
		panic(err)
	}
}
