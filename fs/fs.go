package fs

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

	tr = otel.Tracer("fs")
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
func (m *StatikFS) Mkdir(context.Context, string, os.FileMode) error {
	// If fs.ErrPermission is used, gvfs retries the operation forever
	return fs.ErrNotExist
}

// RemoveAll implements webdav.FileSystem for StatikFS.
func (m *StatikFS) RemoveAll(context.Context, string) error { return errPermission }

// Rename implements webdav.FileSystem for StatikFS.
func (m *StatikFS) Rename(context.Context, string, string) error { return errPermission }

// OpenFile implements webdav.FileSystem for StatikFS.
func (m *StatikFS) OpenFile(
	ctx context.Context,
	name string,
	flag int,
	perm os.FileMode,
) (webdav.File, error) {
	ctx, span := tr.Start(ctx, "OpenFile")
	span.SetAttributes(
		attribute.String("name", name),
		attribute.Int("flag", flag),
	)
	defer span.End()

	// only allow read-only access
	if flag != os.O_RDONLY {
		return nil, fs.ErrPermission
	}

	statikPath := path.Dir(name)

	ctx, statikSpan := tr.Start(ctx, "GetStatik")
	statik, err := m.cache.Get(ctx, statikPath)
	if err != nil {
		statikSpan.RecordError(err)
		statikSpan.SetStatus(codes.Error, "requested statik.json for a path that doesn't exist")
		return nil, fs.ErrNotExist
	}
	statikSpan.End()

	if strings.HasSuffix(name, "/") {
		// we're opening a dir
		return statik, nil
	}

	name = path.Base(name)
	name = strings.TrimPrefix(name, "/")

	// we're opening a file
	for _, file := range statik.Files {
		if file.Name() == name {
			span.AddEvent("file found")
			return m.getFile(file), nil
		}
	}

	// we're opening a dir (??)
	for _, dir := range statik.Directories {
		if dir.Name() == name {
			span.AddEvent("dir found")
			redir := statikPath + "/" + name + "/"
			return m.OpenFile(ctx, redir, flag, perm)
		}
	}

	return nil, fs.ErrNotExist
}

func (m *StatikFS) getFile(file StatikFileInfo) webdav.File {

	if file.Mime == "text/statik-link" {
		return NewLinkFile(file)
	}

	populate := m.createFilePopulate(file)
	return NewLazyMemFile(file, populate)
}

func (m *StatikFS) createFilePopulate(file StatikFileInfo) func() (*bytes.Buffer, error) {
	return func() (*bytes.Buffer, error) {

		if file.Mime == "text/statik-link" {
			return bytes.NewBufferString(file.Url), nil
		}

		log.Debug().Str("url", file.Url).Msg("opening file")

		buf, found := m.openFiles.Get(file.Url)
		if found {
			// cache hit
			log.Debug().Str("url", file.Url).Msg("cache hit")
			return buf, nil
		}

		// cache miss
		log.Debug().Str("url", file.Url).Msg("cache miss")
		buf, err := fetchBytes(file)
		if err != nil {
			return nil, err
		}
		m.openFiles.Add(file.Url, buf) // populate cache

		return buf, nil
	}
}

func fetchBytes(i StatikFileInfo) (*bytes.Buffer, error) {
	resp, err := httpGet(context.Background(), i.Url)
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
func (m *StatikFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	statikPath := path.Dir(name)

	statik, err := m.cache.Get(ctx, statikPath)
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
