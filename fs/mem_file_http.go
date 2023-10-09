package fs

import (
	"bytes"
	fs2 "io/fs"
	"net/http"
)

// inMemHttpFile represents a file that is retrieved from a GET request to a URL.
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

func (f *inMemHttpFile) Stat() (fs2.FileInfo, error)           { return f.i, nil }         // Stat implements fs.File for inMemHttpFile
func (f *inMemHttpFile) Readdir(_ int) ([]fs2.FileInfo, error) { return nil, errNotADir }  // Readdir implements fs.File for inMemHttpFile
func (f *inMemHttpFile) Write(_ []byte) (int, error)           { return 0, errPermission } // Write implements fs.File for inMemHttpFile
func (f *inMemHttpFile) Close() error                          { return nil }              // Close implements fs.File for inMemHttpFile
func (f *inMemHttpFile) Read(b []byte) (int, error) {
	err := f.open()
	if err != nil {
		return 0, err
	}

	return f.b.Read(b)
} // Read implements fs.File for inMemHttpFile
func (f *inMemHttpFile) Seek(offset int64, whence int) (int64, error) {
	err := f.open()
	if err != nil {
		return 0, err
	}

	return f.b.Seek(offset, whence)
} // Seek implements fs.File for inMemHttpFile
