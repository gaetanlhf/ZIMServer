package fs

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	zimreader "github.com/gaetanlhf/ZIMServer/internal/zim/reader"
)

func New(reader *zimreader.ZIMReader) *ZIMFS {
	return &ZIMFS{reader: reader}
}

func (zfs *ZIMFS) GetEntry(name string) (zimreader.DirectoryEntry, error) {
	return zfs.searchEntryFromURL(name)
}

func (zfs *ZIMFS) Open(name string) (fs.File, error) {
	switch name {
	case ".":
		return zfs.serveDirectory(name)
	case "index.html":
		return zfs.serveIndex()
	default:
		return zfs.serveZimEntry(name)
	}
}

func (zfs *ZIMFS) serveIndex() (fs.File, error) {
	mainPage, err := zfs.reader.GetMainPage()
	if err != nil {
		return nil, os.ErrNotExist
	}

	return zfs.serveZimEntry(mainPage.GetPath())
}

func (zfs *ZIMFS) serveDirectory(name string) (fs.File, error) {
	zimFile := &Directory{
		File: File{
			fileInfo: &FileInfo{
				isDir:   true,
				modTime: time.Time{},
				mode:    fs.ModeDir | 0555,
				name:    name,
				size:    0,
			},
			reader: bytes.NewReader(nil),
		},
		entries: make([]fs.DirEntry, 0),
	}

	return zimFile, nil
}

func (zfs *ZIMFS) serveZimEntry(name string) (fs.File, error) {
	entry, err := zfs.searchEntryFromURL(name)
	if err != nil {
		return nil, os.ErrNotExist
	}
	return zfs.OpenEntry(entry, name)
}

func (zfs *ZIMFS) OpenEntry(entry zimreader.DirectoryEntry, name string) (fs.File, error) {
	resolvedEntry, err := zfs.reader.ResolveRedirect(entry)
	if err != nil {
		return nil, err
	}
	readerAt, size, err := zfs.reader.GetContentReader(resolvedEntry)
	if err != nil {
		return nil, err
	}

	return &File{
		fileInfo: &FileInfo{
			isDir:   false,
			modTime: time.Time{},
			mode:    0444,
			name:    filepath.Base(name),
			size:    size,
		},
		reader: io.NewSectionReader(readerAt, 0, size),
	}, nil
}

func (zfs *ZIMFS) searchEntryFromURL(url string) (zimreader.DirectoryEntry, error) {
	if entry, err := zfs.reader.FindEntry(url); err == nil {
		return entry, nil
	}
	if entry, err := zfs.reader.FindAuxiliaryEntry(url); err == nil {
		return entry, nil
	}
	return nil, os.ErrNotExist
}

var _ fs.FS = (*ZIMFS)(nil)
