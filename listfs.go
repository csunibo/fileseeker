package main

import (
	"context"
	"io/fs"
	"os"
	"strings"
	"time"

	"golang.org/x/net/webdav"
)

type (
	// listFS is a webdav.FileSystem made by a list of files with names
	listFS []string

	// listFile is a webdav.File that has a name and no children
	listFile string

	// listRoot is a webdav.File that is the root directory of a listFS
	listRoot struct {
		listFile
		children []string
	}
)

func (l listFS) Mkdir(_ context.Context, _ string, _ os.FileMode) error { return fs.ErrExist }      // Mkdir implements webdav.FileSystem for listFS
func (l listFS) RemoveAll(_ context.Context, _ string) error            { return fs.ErrPermission } // RemoveAll implements webdav.FileSystem for listFS
func (l listFS) Rename(_ context.Context, _, _ string) error            { return fs.ErrPermission } // Rename implements webdav.FileSystem for listFS
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
} // OpenFile implements webdav.FileSystem for listFS
func (l listFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	file, err := l.OpenFile(ctx, name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	return file.Stat()
} // Stat implements webdav.FileSystem for listFS.

func (l listFS) Seek(_ int64, _ int) (int64, error) { return 0, fs.ErrPermission } // Seek implements fs.File for listFS
func (l listFS) Write(_ []byte) (int, error)        { return 0, fs.ErrPermission } // Write implements fs.File for listFS

func (l listFile) Name() string       { return string(l) }   // Name implements fs.FileInfo for listFile
func (l listFile) Size() int64        { return 0 }           // Size implements fs.FileInfo for listFile
func (l listFile) Mode() fs.FileMode  { return fs.ModeDir }  // Mode implements fs.FileInfo for listFile
func (l listFile) ModTime() time.Time { return serverStart } // ModTime implements fs.FileInfo for listFile
func (l listFile) IsDir() bool        { return true }        // IsDir implements fs.FileInfo for listFile
func (l listFile) Sys() any           { return nil }         // Sys implements fs.FileInfo for listFile

func (l listFile) Close() error                         { return nil }                   // Close implements fs.File for listFile
func (l listFile) Read(_ []byte) (int, error)           { return 0, fs.ErrPermission }   // Read implements fs.File for listFile
func (l listFile) Seek(_ int64, _ int) (int64, error)   { return 0, fs.ErrPermission }   // Seek implements fs.File for listFile
func (l listFile) Write(_ []byte) (int, error)          { return 0, fs.ErrPermission }   // Write implements fs.File for listFile
func (l listFile) Stat() (fs.FileInfo, error)           { return l, nil }                // Stat implements fs.File for listFile
func (l listFile) Readdir(_ int) ([]fs.FileInfo, error) { return nil, fs.ErrPermission } // Readdir implements fs.File for listFile

func (l listRoot) Readdir(count int) ([]fs.FileInfo, error) {
	if count <= 0 || count > len(l.children) {
		count = len(l.children)
	}

	files := make([]fs.FileInfo, count)
	for i := 0; i < count; i++ {
		files[i] = listFile(l.children[i])
	}

	return files, nil
} // Readdir implements fs.ReadDirFile for listRoot
