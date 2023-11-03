package fs

import (
	"bytes"
	"io/fs"
)

type LazyMemFile struct {
	reader   *bytes.Reader
	info     StatikFileInfo
	populate func() (*bytes.Buffer, error)
}

func NewLazyMemFile(info StatikFileInfo, populate func() (*bytes.Buffer, error)) *LazyMemFile {
	return &LazyMemFile{populate: populate, info: info}
}

func (m *LazyMemFile) Close() error                       { return nil }
func (m *LazyMemFile) Readdir(int) ([]fs.FileInfo, error) { return nil, errNotADir }
func (m *LazyMemFile) Stat() (fs.FileInfo, error)         { return m.info, nil }
func (m *LazyMemFile) Write([]byte) (int, error)          { return 0, errReadOnly }
func (m *LazyMemFile) Read(p []byte) (int, error) {
	if m.reader == nil {
		err := m.load()
		if err != nil {
			return 0, err
		}
	}

	return m.reader.Read(p)
}
func (m *LazyMemFile) Seek(offset int64, whence int) (int64, error) {
	if m.reader == nil {
		err := m.load()
		if err != nil {
			return 0, err
		}
	}

	return m.reader.Seek(offset, whence)
}

func (m *LazyMemFile) load() error {
	buf, err := m.populate()
	if err != nil {
		return err
	}

	m.reader = bytes.NewReader(buf.Bytes())
	return nil
}
