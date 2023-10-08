package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"golang.org/x/net/webdav"
)

type StatikFileInfo struct {
	NameRaw string    `json:"name"`
	Path    string    `json:"path"`
	Url     string    `json:"url"`
	Mime    string    `json:"mime"`
	SizeRaw string    `json:"size"`
	Time    time.Time `json:"time"`
}

var (
	errNotADir    = errors.New("not a directory")
	errPermission = fs.ErrPermission
)

// fs.FileInfo interface implementation

func (f StatikFileInfo) Name() string       { return f.NameRaw }
func (f StatikFileInfo) Mode() fs.FileMode  { return 0666 }
func (f StatikFileInfo) ModTime() time.Time { return f.Time }
func (f StatikFileInfo) IsDir() bool        { return false }
func (f StatikFileInfo) Sys() any           { return nil }

func newInMemFile(file StatikFileInfo) (webdav.File, error) {
	if file.Mime == "text/statik-link" {
		return newInMemLinkFile(file)
	} else {
		return newInMemHttpFile(file)
	}
}

type inMemLinkFile struct {
	i StatikFileInfo
	r *bytes.Reader
}

func (f inMemLinkFile) Stat() (fs.FileInfo, error)                   { return f.i, nil }
func (f inMemLinkFile) Close() error                                 { return nil }
func (f inMemLinkFile) Read(p []byte) (n int, err error)             { return f.r.Read(p) }
func (f inMemLinkFile) Seek(offset int64, whence int) (int64, error) { return f.r.Seek(offset, whence) }
func (f inMemLinkFile) Readdir(_ int) ([]fs.FileInfo, error)         { return nil, errNotADir }
func (f inMemLinkFile) Write(_ []byte) (n int, err error)            { return 0, errPermission }

const linkFileTemplate = `You can find this file at %s`

func newInMemLinkFile(info StatikFileInfo) (*inMemLinkFile, error) {
	content := fmt.Sprintf(linkFileTemplate, info.Url)
	return &inMemLinkFile{i: info, r: bytes.NewReader([]byte(content))}, nil
}

// inMemLinkFile represents a file that is retrieved from a GET request to a URL.
// The request is lazily performed when the file is first opened.
type inMemHttpFile struct {
	i StatikFileInfo
	r *http.Request
	b *bytes.Reader
}

func newInMemHttpFile(file StatikFileInfo) (*inMemHttpFile, error) {
	req, err := http.NewRequest("GET", file.Url, nil)
	if err != nil {
		return nil, err
	}

	return &inMemHttpFile{i: file, r: req}, nil
}

func (f *inMemHttpFile) open() error {
	if f.b != nil {
		return nil
	}

	resp, err := http.DefaultClient.Do(f.r)
	if err != nil {
		return err
	}

	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	f.b = bytes.NewReader(buf.Bytes())
	return nil
}

func (f *inMemHttpFile) Seek(offset int64, whence int) (int64, error) {
	err := f.open()
	if err != nil {
		return 0, err
	}

	return f.b.Seek(offset, whence)
}
func (f *inMemHttpFile) Read(p []byte) (int, error) {
	err := f.open()
	if err != nil {
		return 0, err
	}

	return f.b.Read(p)
}
func (f *inMemHttpFile) Stat() (fs.FileInfo, error)           { return f.i, nil }
func (f *inMemHttpFile) Readdir(_ int) ([]fs.FileInfo, error) { return nil, errNotADir }
func (f *inMemHttpFile) Write(_ []byte) (n int, err error)    { return 0, errPermission }
func (f *inMemHttpFile) Close() error                         { return nil }
