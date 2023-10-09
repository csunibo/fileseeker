package fs

import (
	"bytes"
	"io/fs"
)

type LazyMemFile struct {
	r *bytes.Reader
	i StatikFileInfo
	p func() (*bytes.Buffer, error)
}

func NewLazyMemFile(info StatikFileInfo, populate func() (*bytes.Buffer, error)) *LazyMemFile {
	return &LazyMemFile{p: populate, i: info}
}

func (m *LazyMemFile) Close() error                         { return nil }
func (m *LazyMemFile) Readdir(_ int) ([]fs.FileInfo, error) { return nil, errNotADir }
func (m *LazyMemFile) Stat() (fs.FileInfo, error)           { return m.i, nil }
func (m *LazyMemFile) Write(_ []byte) (n int, err error)    { return 0, errReadOnly }
func (m *LazyMemFile) Read(p []byte) (int, error) {
	if m.r == nil {
		err := m.load()
		if err != nil {
			return 0, err
		}
	}

	return m.r.Read(p)
}
func (m *LazyMemFile) Seek(offset int64, whence int) (int64, error) {
	if m.r == nil {
		err := m.load()
		if err != nil {
			return 0, err
		}
	}

	return m.r.Seek(offset, whence)
}

func (m *LazyMemFile) load() error {
	buf, err := m.p()
	if err != nil {
		return err
	}

	m.r = bytes.NewReader(buf.Bytes())
	return nil
}
