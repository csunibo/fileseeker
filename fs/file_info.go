package fs

import (
	"io/fs"
	"time"
)

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
