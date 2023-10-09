package fs

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/net/webdav"
)

const (
	StatikCachingTime = 5 * time.Minute // how long to cache statik.json files
	fileCacheSize     = 100             // number of files to cache
)

var (
	errNotADir    = errors.New("not a directory") // a directory operation is performed on a file
	errReadOnly   = errors.New("read only")       // a write operation is performed on a read-only file
	errPermission = fs.ErrPermission              // a write operation is performed on a read-only file
)

// StatikFS represents a virtual filesystem that is backed by a statik.json files
// in a remote server.
type StatikFS struct {
	baseUrl   string                            // base url of the remote server
	cache     *statikCache                      // cache of statik.json files
	openFiles *lru.Cache[string, *bytes.Buffer] // cache of open files (to avoid re-fetching them)
}

// NewStatikFS returns a new StatikFS that is backed by a statik.json file in the
// remote server at base url.
//
// The returned StatikFS is read-only. The returned StatikFS is goroutine-safe.
func NewStatikFS(base string) (*StatikFS, error) {
	fileCache, err := lru.New[string, *bytes.Buffer](fileCacheSize)
	if err != nil {
		return nil, err
	}
	sCache := newStatikCache(base)

	return &StatikFS{
		openFiles: fileCache,
		baseUrl:   base,
		cache:     sCache,
	}, nil
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
	statik, err := m.cache.Get(statikPath)
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

			populate := m.createFilePopulate(file)
			return NewLazyMemFile(file, populate), nil
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

func (m *StatikFS) createFilePopulate(file StatikFileInfo) func() (*bytes.Buffer, error) {
	return func() (*bytes.Buffer, error) {

		slog.Debug("opening file", "url", file.Url)

		buf, found := m.openFiles.Get(file.Url)
		if found {
			// cache hit
			slog.Debug("cache hit", "url", file.Url)
			return buf, nil
		}

		// cache miss
		slog.Debug("fetching file", "url", file.Url)
		buf, err := fetchBytes(file)
		if err != nil {
			return nil, err
		}
		m.openFiles.Add(file.Url, buf) // populate cache

		return buf, nil
	}
}

func fetchBytes(i StatikFileInfo) (*bytes.Buffer, error) {
	resp, err := http.Get(i.Url)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

// Stat implements webdav.FileSystem for StatikFS.
func (m *StatikFS) Stat(_ context.Context, name string) (os.FileInfo, error) {
	statikPath := path.Dir(name)

	statik, err := m.cache.Get(statikPath)
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
