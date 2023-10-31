package fs

import (
	"bytes"
	"fmt"
	"io/fs"
)

type inMemLinkFile struct {
	i StatikFileInfo
	*bytes.Reader
}

const linkFileTemplate = `You can find this file at %s`

func newInMemLinkFile(info StatikFileInfo) (*inMemLinkFile, error) {
	content := fmt.Sprintf(linkFileTemplate, info.Url)
	return &inMemLinkFile{i: info, Reader: bytes.NewReader([]byte(content))}, nil
}

func (f inMemLinkFile) Stat() (fs.FileInfo, error)           { return f.i, nil }         // Stat implements fs.File for inMemLinkFile
func (f inMemLinkFile) Close() error                         { return nil }              // Close implements fs.File for inMemLinkFile
func (f inMemLinkFile) Readdir(_ int) ([]fs.FileInfo, error) { return nil, errNotADir }  // Readdir implements fs.File for inMemLinkFile
func (f inMemLinkFile) Write(_ []byte) (int, error)          { return 0, errPermission } // Write implements fs.File for inMemLinkFile
