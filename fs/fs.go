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
	"time"

	"golang.org/x/net/webdav"
)

var StatikCachingTime = 5 * time.Minute

// StatikFS represents a virtual filesystem that is backed by a statik.json files
// in a remote server.
//
// cache is a map from path to Statik, where path is the path of the directory
// containing the statik.json file. cacheLock is a RW mutex that protects cache.
// It is used by GetStatik and cacheStatik.
type StatikFS struct {
	baseUrl string

	cache     map[string]statikCache
	cacheLock sync.RWMutex
}

// NewStatikFS returns a new StatikFS that is backed by a statik.json file in the
// remote server at base url.
//
// The returned StatikFS is read-only. The returned StatikFS is goroutine-safe.
func NewStatikFS(base string) *StatikFS {
	return &StatikFS{
		baseUrl:   base,
		cacheLock: sync.RWMutex{},
		cache:     map[string]statikCache{},
	}
}

// statikCache is a struct that represents a cached statik.json file and its
// expiration time.
type statikCache struct {
	statik Statik
	exp    time.Time
}

// GetStatik returns the Statik struct for the statik.json file in the directory
// specified by path.
//
// If the statik.json file is not cached, it is fetched from the remote server,
// cached and returned.
//
// The function is safe for concurrent use, as it uses a RW mutex to protect the
// cache.
func (m *StatikFS) GetStatik(path string) (Statik, error) {
	// check cache
	m.cacheLock.RLock()
	cache, contentOk := m.cache[path]
	m.cacheLock.RUnlock()

	if contentOk && cache.exp.After(time.Now()) {
		// cache hit
		return cache.statik, nil
	}

	if contentOk {
		// cache expired
		m.cacheLock.Lock()
		delete(m.cache, path)
		m.cacheLock.Unlock()
	}

	// cache miss
	slog.Debug("caching statik.json for path", "path", path)

	response, err := http.Get(m.baseUrl + path + "/statik.json")
	if err != nil {
		return Statik{}, fmt.Errorf("error getting statik.json: %w", err)
	}

	statik := Statik{}
	err = json.NewDecoder(response.Body).Decode(&statik)
	if err != nil {
		return Statik{}, fmt.Errorf("error decoding statik.json: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return Statik{}, fmt.Errorf("error closing response body: %w", err)
	}

	// populate cache
	m.cacheLock.Lock()
	m.cache[path] = statikCache{statik, time.Now().Add(StatikCachingTime)}
	m.cacheLock.Unlock()

	return statik, nil
}

// Mkdir implements webdav.FileSystem for StatikFS.
func (m *StatikFS) Mkdir(_ context.Context, _ string, _ os.FileMode) error {
	// If fs.ErrPermission is used, gvfs retries the operation forever
	return fs.ErrNotExist
}

// RemoveAll implements webdav.FileSystem for StatikFS.
func (m *StatikFS) RemoveAll(_ context.Context, _ string) error { return errPermission }

// Rename implements webdav.FileSystem for StatikFS.
func (m *StatikFS) Rename(_ context.Context, _, _ string) error { return errPermission }

// OpenFile implements webdav.FileSystem for StatikFS.
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
	statik, err := m.GetStatik(statikPath)
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

// Stat implements webdav.FileSystem for StatikFS.
func (m *StatikFS) Stat(_ context.Context, name string) (os.FileInfo, error) {
	statikPath := path.Dir(name)

	statik, err := m.GetStatik(statikPath)
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
