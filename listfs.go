package main

import (
	"context"
	"io/fs"
	"os"
	"strings"
	"time"

	"golang.org/x/net/webdav"
)

// listFS is a webdav.FileSystem that is a list of strings.
type listFS []string

func (l listFS) Seek(_ int64, _ int) (int64, error)                     { return 0, fs.ErrPermission }
func (l listFS) Write(_ []byte) (int, error)                            { return 0, fs.ErrPermission }
func (l listFS) Mkdir(_ context.Context, _ string, _ os.FileMode) error { return fs.ErrExist }
func (l listFS) RemoveAll(_ context.Context, _ string) error            { return fs.ErrPermission }
func (l listFS) Rename(_ context.Context, _, _ string) error            { return fs.ErrPermission }
func (l listFS) OpenFile(_ context.Context, name string, _ int, _ os.FileMode) (webdav.File, error) {

	if name == "/" {
		return listRoot{listFile: "", children: l}, nil
	}

	name = strings.TrimPrefix(name, "/")
	for _, course := range l {
		if name == course {
			return listFile(course), nil
		}
	}

	return nil, fs.ErrNotExist
}
func (l listFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	file, err := l.OpenFile(ctx, name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	return file.Stat()
}

// listFile is a webdav.File that is a string.
type listFile string

func (l listFile) Name() string                         { return string(l) }
func (l listFile) Size() int64                          { return 0 }
func (l listFile) Mode() fs.FileMode                    { return fs.ModeDir }
func (l listFile) ModTime() time.Time                   { return serverStart }
func (l listFile) IsDir() bool                          { return true }
func (l listFile) Sys() any                             { return nil }
func (l listFile) Close() error                         { return nil }
func (l listFile) Read(_ []byte) (n int, err error)     { return 0, fs.ErrPermission }
func (l listFile) Seek(_ int64, _ int) (int64, error)   { return 0, fs.ErrPermission }
func (l listFile) Write(_ []byte) (n int, err error)    { return 0, fs.ErrPermission }
func (l listFile) Stat() (fs.FileInfo, error)           { return l, nil }
func (l listFile) Readdir(_ int) ([]fs.FileInfo, error) { return nil, fs.ErrPermission }

// listRoot is a webdav.File that is the root directory of a listFS.
type listRoot struct {
	listFile
	children []string
}

func (l listRoot) Readdir(count int) ([]fs.FileInfo, error) {
	if count <= 0 || count > len(l.children) {
		count = len(l.children)
	}

	files := make([]fs.FileInfo, count)
	for i := 0; i < count; i++ {
		files[i] = listFile(l.children[i])
	}

	return files, nil
}
