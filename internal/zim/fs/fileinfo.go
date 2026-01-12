package fs

import (
	"io/fs"
	"time"
)

func (i *FileInfo) IsDir() bool {
	return i.isDir
}

func (i *FileInfo) ModTime() time.Time {
	return i.modTime
}

func (i *FileInfo) Mode() fs.FileMode {
	return i.mode
}

func (i *FileInfo) Name() string {
	return i.name
}

func (i *FileInfo) Size() int64 {
	return i.size
}

func (i *FileInfo) Sys() any {
	return nil
}

var _ fs.FileInfo = (*FileInfo)(nil)
