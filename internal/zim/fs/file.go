package fs

import (
	"io"
	"io/fs"
	"os"
)

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := f.reader.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, fs.ErrInvalid
}

func (f *File) Close() error {
	return nil
}

func (f *File) Read(d []byte) (int, error) {
	return f.reader.Read(d)
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f.fileInfo, nil
}

func (f *File) Readdir(count int) ([]fs.FileInfo, error) {
	return nil, os.ErrInvalid
}

var _ fs.File = (*File)(nil)
var _ io.Seeker = (*File)(nil)
