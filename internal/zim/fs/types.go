package fs

import (
	"github.com/gaetanlhf/ZIMServer/internal/zim/reader"
	"io"
	"io/fs"
	"time"
)

type ZIMFS struct {
	reader *reader.ZIMReader
}

type File struct {
	fileInfo *FileInfo
	reader   io.ReadSeeker
}

type FileInfo struct {
	isDir   bool
	modTime time.Time
	mode    fs.FileMode
	name    string
	size    int64
}

type Directory struct {
	File
	entries []fs.DirEntry
}
