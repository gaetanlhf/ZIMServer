package fs

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	zimreader "github.com/gaetanlhf/ZIMServer/internal/zim/reader"
)

func New(reader *zimreader.ZIMReader) *ZIMFS {
	return &ZIMFS{reader: reader}
}

// GetEntry récupère une entrée ZIM sans la résoudre (pour détecter les redirections)
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

	resolvedEntry, err := zfs.reader.ResolveRedirect(entry)
	if err != nil {
		return nil, err
	}

	content, err := zfs.reader.GetContent(resolvedEntry)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(name)

	zimFile := &File{
		fileInfo: &FileInfo{
			isDir:   false,
			modTime: time.Time{},
			mode:    0444,
			name:    filename,
			size:    int64(len(content)),
		},
		reader: bytes.NewReader(content),
	}

	return zimFile, nil
}

func (zfs *ZIMFS) searchEntryFromURL(url string) (zimreader.DirectoryEntry, error) {
	entry, err := zfs.reader.GetEntryByURL(zimreader.NamespaceContent, url)
	if err == nil {
		return entry, nil
	}

	contentNamespaces := []byte{
		zimreader.NamespaceWellKnown,
		zimreader.NamespaceMetadata,
		zimreader.NamespaceIndex,
	}

	for _, ns := range contentNamespaces {
		entry, err := zfs.reader.GetEntryByURL(ns, url)
		if err == nil {
			return entry, nil
		}
	}

	entries, err := zfs.reader.ListEntriesByNamespace(zimreader.NamespaceContent)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.GetPath() == url {
			return e, nil
		}
	}

	wellKnownEntries, err := zfs.reader.ListEntriesByNamespace(zimreader.NamespaceWellKnown)
	if err == nil {
		for _, e := range wellKnownEntries {
			if e.GetPath() == url {
				return e, nil
			}
		}
	}

	return nil, os.ErrNotExist
}

var _ fs.FS = (*ZIMFS)(nil)
