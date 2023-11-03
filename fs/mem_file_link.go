package fs

import (
	"bytes"
	"fmt"
	"io/fs"
)

type LinkFile struct {
	i StatikFileInfo
	*bytes.Reader
}

func (f LinkFile) Stat() (fs.FileInfo, error)         { return f.i, nil }         // Stat implements fs.File for LinkFile
func (f LinkFile) Close() error                       { return nil }              // Close implements fs.File for LinkFile
func (f LinkFile) Readdir(int) ([]fs.FileInfo, error) { return nil, errNotADir }  // Readdir implements fs.File for LinkFile
func (f LinkFile) Write([]byte) (int, error)          { return 0, errPermission } // Write implements fs.File for LinkFile

const linkFileTemplate = `[Desktop Entry]
Type=Link
Version=1.0
Name=%s
URL=%s
Icon=text-html
`

func NewLinkFile(info StatikFileInfo) *LinkFile {
	content := fmt.Sprintf(linkFileTemplate, info.NameRaw, info.Url)
	info.SizeRaw = fmt.Sprintf("%d", len(content))
	info.NameRaw += ".desktop"

	return &LinkFile{info, bytes.NewReader([]byte(content))}
}
