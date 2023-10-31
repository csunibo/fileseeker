package listfs

import (
	"context"
	"io/fs"
	"os"
	"strings"
	"time"

	"golang.org/x/net/webdav"
)

type (
	// ListFS is a webdav.FileSystem made by a list of files with names
	ListFS struct {
		names    []string
		modTimes time.Time
	}

	// listFile is a webdav.File that has a name and no children
	listFile struct {
		name    string
		modTime time.Time
	}

	// listRoot is a webdav.File that is the root directory of a ListFS
	listRoot struct {
		listFile
		children []string
	}
)

func NewListFS(names []string) webdav.FileSystem {
	return ListFS{names: names, modTimes: time.Now()}
}

func (f ListFS) Mkdir(context.Context, string, os.FileMode) error { return fs.ErrExist }      // Mkdir implements webdav.FileSystem for ListFS
func (f ListFS) RemoveAll(context.Context, string) error          { return fs.ErrPermission } // RemoveAll implements webdav.FileSystem for ListFS
func (f ListFS) Rename(context.Context, string, string) error     { return fs.ErrPermission } // Rename implements webdav.FileSystem for ListFS
func (f ListFS) OpenFile(_ context.Context, name string, _ int, _ os.FileMode) (webdav.File, error) {
	if name == "/" {
		return listRoot{
			listFile: listFile{name: "", modTime: f.modTimes},
			children: f.names,
		}, nil
	}

	name = strings.TrimPrefix(name, "/")
	for _, course := range f.names {
		if name == course {
			return listFile{name: name, modTime: f.modTimes}, nil
		}
	}

	return nil, fs.ErrNotExist
} // OpenFile implements webdav.FileSystem for ListFS
func (f ListFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	file, err := f.OpenFile(ctx, name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	return file.Stat()
} // Stat implements webdav.FileSystem for ListFS.

func (f ListFS) Seek(int64, int) (int64, error) { return 0, fs.ErrPermission } // Seek implements fs.File for ListFS
func (f ListFS) Write([]byte) (int, error)      { return 0, fs.ErrPermission } // Write implements fs.File for ListFS

func (l listFile) Name() string       { return l.name }     // Name implements fs.FileInfo for listFile
func (l listFile) Size() int64        { return 0 }          // Size implements fs.FileInfo for listFile
func (l listFile) Mode() fs.FileMode  { return fs.ModeDir } // Mode implements fs.FileInfo for listFile
func (l listFile) ModTime() time.Time { return l.modTime }  // ModTime implements fs.FileInfo for listFile
func (l listFile) IsDir() bool        { return true }       // IsDir implements fs.FileInfo for listFile
func (l listFile) Sys() any           { return nil }        // Sys implements fs.FileInfo for listFile

func (l listFile) Close() error                       { return nil }                   // Close implements fs.File for listFile
func (l listFile) Read([]byte) (int, error)           { return 0, fs.ErrPermission }   // Read implements fs.File for listFile
func (l listFile) Seek(int64, int) (int64, error)     { return 0, fs.ErrPermission }   // Seek implements fs.File for listFile
func (l listFile) Write([]byte) (int, error)          { return 0, fs.ErrPermission }   // Write implements fs.File for listFile
func (l listFile) Stat() (fs.FileInfo, error)         { return l, nil }                // Stat implements fs.File for listFile
func (l listFile) Readdir(int) ([]fs.FileInfo, error) { return nil, fs.ErrPermission } // Readdir implements fs.File for listFile

func (l listRoot) Readdir(count int) ([]fs.FileInfo, error) {
	if count <= 0 || count > len(l.children) {
		count = len(l.children)
	}

	files := make([]fs.FileInfo, count)
	for i := 0; i < count; i++ {
		files[i] = listFile{name: l.children[i], modTime: l.modTime}
	}

	return files, nil
} // Readdir implements fs.ReadDirFile for listRoot
