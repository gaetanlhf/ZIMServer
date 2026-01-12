package web

import (
	"net/http"

	"github.com/gaetanlhf/ZIMServer/internal/zim/fs"
	"github.com/gaetanlhf/ZIMServer/internal/zim/index"
	zimreader "github.com/gaetanlhf/ZIMServer/internal/zim/reader"
)

type Archive struct {
	Name     string
	Path     string
	Reader   *zimreader.ZIMReader
	FS       *fs.ZIMFS
	IndexMgr *index.Manager
	Metadata Metadata
}

type Metadata struct {
	Title        string
	Description  string
	Language     string
	LanguageCode string
	Creator      string
	Publisher    string
	Date         string
	Tags         string
	Category     string
	EntryCount   uint32
}

type LanguageInfo struct {
	Code string
	Name string
}

type HomeData struct {
	Archives   []*Archive
	Count      int
	Languages  []LanguageInfo
	Categories []string
	Version    string
}

type ViewerData struct {
	ArchiveName  string
	ArchiveTitle string
	EntryPath    string
	FaviconURL   string
	FaviconType  string
}

type TemplateRenderer interface {
	Render(w http.ResponseWriter, name string, data interface{}) error
}

type SearchResponse struct {
	Query   string         `json:"query"`
	Archive string         `json:"archive"`
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
	Time    string         `json:"time"`
}

type SearchResult struct {
	Title      string  `json:"title"`
	Path       string  `json:"path"`
	Namespace  string  `json:"namespace"`
	Score      float64 `json:"score"`
	URL        string  `json:"url"`
	IsRedirect bool    `json:"isRedirect"`
}

type APISearchResponse struct {
	Query   string            `json:"query"`
	Results []APISearchResult `json:"results"`
	Count   int               `json:"count"`
}

type APISearchResult struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

type APIRandomResponse struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}
