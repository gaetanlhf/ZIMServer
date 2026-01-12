package reader

import "io"

const (
	MagicNumber        = 0x44D495A
	NamespaceContent   = 'C'
	NamespaceMetadata  = 'M'
	NamespaceWellKnown = 'W'
	NamespaceIndex     = 'X'
)

type CompressionType byte

const (
	CompressionNone  CompressionType = 1
	CompressionLZMA2 CompressionType = 4
	CompressionZstd  CompressionType = 5
)

type Header struct {
	MagicNumber   uint32
	MajorVersion  uint16
	MinorVersion  uint16
	UUID          [16]byte
	EntryCount    uint32
	ClusterCount  uint32
	PathPtrPos    uint64
	TitlePtrPos   uint64
	ClusterPtrPos uint64
	MimeListPos   uint64
	MainPage      uint32
	LayoutPage    uint32
	ChecksumPos   uint64
}

type DirectoryEntry interface {
	GetNamespace() byte
	GetPath() string
	GetTitle() string
	IsRedirect() bool
}

type ContentEntry struct {
	MimeType      uint16
	Namespace     byte
	Revision      uint32
	ClusterNumber uint32
	BlobNumber    uint32
	Path          string
	Title         string
}

func (e *ContentEntry) GetNamespace() byte { return e.Namespace }
func (e *ContentEntry) GetPath() string    { return e.Path }
func (e *ContentEntry) GetTitle() string {
	if e.Title == "" {
		return e.Path
	}
	return e.Title
}
func (e *ContentEntry) IsRedirect() bool { return false }

type RedirectEntry struct {
	Namespace     byte
	Revision      uint32
	RedirectIndex uint32
	Path          string
	Title         string
}

func (e *RedirectEntry) GetNamespace() byte { return e.Namespace }
func (e *RedirectEntry) GetPath() string    { return e.Path }
func (e *RedirectEntry) GetTitle() string {
	if e.Title == "" {
		return e.Path
	}
	return e.Title
}
func (e *RedirectEntry) IsRedirect() bool { return true }

type Cluster struct {
	Compression CompressionType
	Extended    bool
	reader      io.ReaderAt
	offset      uint64
	size        uint64
}

type ZIMReader struct {
	file          io.ReaderAt
	header        *Header
	mimeTypes     []string
	pathPointers  []uint64
	titlePointers []uint32
	clusterPtrs   []uint64
}
