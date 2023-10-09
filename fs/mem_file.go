package fs

import (
	"io/fs"
	"time"

	"golang.org/x/net/webdav"
)

// newInMemFile returns a new inMemFile for the given StatikFileInfo. If the file
// is a link, a new inMemLinkFile is returned, otherwise a new inMemHttpFile is
// returned.
//
// The scope of this function is to abstract the creation of the inMemFile, so
// that every StatikFileInfo can be represented by the most appropriate
// inMemFile.
func newInMemFile(file StatikFileInfo) (webdav.File, error) {
	if file.Mime == "text/statik-link" {
		return newInMemLinkFile(file)
	} else {
		return newInMemHttpFile(file)
	}
}

type StatikFileInfo struct {
	NameRaw string    `json:"name"`
	Path    string    `json:"path"`
	Url     string    `json:"url"`
	Mime    string    `json:"mime"`
	SizeRaw string    `json:"size"`
	Time    time.Time `json:"time"`
}

func (f StatikFileInfo) Name() string       { return f.NameRaw }                  // Name implements fs.FileInfo for StatikFileInfo
func (f StatikFileInfo) Mode() fs.FileMode  { return 0666 }                       // Mode implements fs.FileInfo for StatikFileInfo
func (f StatikFileInfo) ModTime() time.Time { return f.Time }                     // ModTime implements fs.FileInfo for StatikFileInfo
func (f StatikFileInfo) IsDir() bool        { return false }                      // IsDir implements fs.FileInfo for StatikFileInfo
func (f StatikFileInfo) Sys() any           { return nil }                        // Sys implements fs.FileInfo for StatikFileInfo
func (f StatikFileInfo) Size() int64        { return parseSizeOrZero(f.SizeRaw) } // Size implements fs.FileInfo for StatikFileInfo
