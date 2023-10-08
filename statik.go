package main

import (
	"errors"
	"io/fs"
	"time"
)

type Statik struct {
	StatikDirInfo
	Directories []StatikDirInfo  `json:"directories"`
	Files       []StatikFileInfo `json:"files"`
}

func (s Statik) Seek(_ int64, _ int) (int64, error) { return 0, errors.New("read only") }
func (s Statik) Write(_ []byte) (n int, err error)  { return 0, errors.New("read only") }
func (s Statik) Read(_ []byte) (_ int, _ error)     { return 0, errors.New("read only") }
func (s Statik) Close() error                       { return nil }
func (s Statik) Stat() (fs.FileInfo, error)         { return s, nil }

func (s Statik) Readdir(count int) ([]fs.FileInfo, error) {

	if count > 0 {
		count = min(count, len(s.Directories)+len(s.Files))
	} else {
		count = len(s.Directories) + len(s.Files)
	}

	infos := make([]fs.FileInfo, count)

	for i, dir := range s.Directories {
		if i == count {
			return infos, nil
		}
		infos[i] = dir
	}

	for i, file := range s.Files {
		arrIdx := i + len(s.Directories)
		if arrIdx == count {
			return infos, nil
		}
		infos[arrIdx] = file
	}

	return infos, nil
}

// StatikDirInfo represents a directory in Statik
type StatikDirInfo struct {
	Url         string    `json:"url"`
	Time        time.Time `json:"time"`
	GeneratedAt time.Time `json:"generated_at"`
	NameRaw     string    `json:"name"`
	Path        string    `json:"path"`
	SizeRaw     string    `json:"size"`
}

func (d StatikDirInfo) Mode() fs.FileMode  { return fs.ModeDir } // Mode implements fs.FileInfo for StatikDirInfo and Statik
func (d StatikDirInfo) ModTime() time.Time { return d.Time }     // ModTime implements fs.FileInfo for StatikDirInfo and Statik
func (d StatikDirInfo) IsDir() bool        { return true }       // IsDir implements fs.FileInfo for StatikDirInfo and Statik
func (d StatikDirInfo) Sys() any           { return nil }        // Sys implements fs.FileInfo for StatikDirInfo and Statik
func (d StatikDirInfo) Name() string       { return d.NameRaw }  // Name implements fs.FileInfo for StatikDirInfo and Statik
