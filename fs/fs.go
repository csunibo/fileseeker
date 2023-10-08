package fs

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

	"golang.org/x/net/webdav"
)

// StatikFS represents a virtual filesystem that is backed by a statik.json files in a remote server.
//
// cache is a map from path to Statik, where path is the path of the directory containing the statik.json file.
// cacheLock is a RW mutex that protects cache. It is used by getOrCacheStatik and cacheStatik.
type StatikFS struct {
	baseUrl string

	cache     map[string]Statik
	cacheLock sync.RWMutex
}

func NewStatikFs(base string) *StatikFS {
	return &StatikFS{
		baseUrl:   base,
		cacheLock: sync.RWMutex{},
		cache:     map[string]Statik{},
	}
}

func (m *StatikFS) cacheStatik(path string) (Statik, error) {
	response, err := http.Get(m.baseUrl + path + "/statik.json")
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

func (m *StatikFS) getOrCacheStatik(path string) (Statik, error) {
	m.cacheLock.RLock()
	statik, ok := m.cache[path]
	m.cacheLock.RUnlock()

	if !ok {
		return m.cacheStatik(path)
	} else {
		return statik, nil
	}
}

func (m *StatikFS) Mkdir(_ context.Context, _ string, _ os.FileMode) error {
	// If fs.ErrPermission is used, gvfs retries the operation forever
	return fs.ErrNotExist
}
func (m *StatikFS) RemoveAll(_ context.Context, _ string) error { return errPermission }
func (m *StatikFS) Rename(_ context.Context, _, _ string) error { return errPermission }

func (m *StatikFS) OpenFile(
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

func (m *StatikFS) Stat(_ context.Context, name string) (os.FileInfo, error) {
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
