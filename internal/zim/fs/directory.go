package fs

import (
	"io/fs"
	"os"
)

func (d *Directory) ReadDir(n int) ([]fs.DirEntry, error) {
	return d.entries, nil
}

func (d *Directory) Readdir(count int) ([]fs.FileInfo, error) {
	return nil, os.ErrInvalid
}

var _ fs.ReadDirFile = (*Directory)(nil)
