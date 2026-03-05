package services

import (
	"fmt"
	"strings"

	zimreader "github.com/gaetanlhf/ZIMServer/internal/zim/reader"
)

type FaviconService struct{}

func NewFaviconService() *FaviconService {
	return &FaviconService{}
}

func (s *FaviconService) GetFavicon(archive *Archive, archiveName string) (string, string) {
	faviconPaths := []struct {
		namespace byte
		path      string
	}{
		{zimreader.NamespaceWellKnown, "favicon"},
		{zimreader.NamespaceContent, "favicon"},
		{zimreader.NamespaceWellKnown, "favicon.png"},
		{zimreader.NamespaceContent, "favicon.png"},
		{zimreader.NamespaceWellKnown, "favicon.ico"},
		{zimreader.NamespaceContent, "favicon.ico"},
	}

	for _, fp := range faviconPaths {
		entry, err := archive.Reader.GetEntryByURL(fp.namespace, fp.path)
		if err != nil {
			continue
		}

		faviconURL := fmt.Sprintf("/content/%s/%s", archiveName, entry.GetPath())

		mimeType, _ := archive.Reader.GetMimeType(entry)
		if mimeType == "" || !strings.HasPrefix(mimeType, "image/") {
			content, err := archive.Reader.GetContent(entry)
			if err != nil {
				continue
			}
			detectedMime := detectIllustrationMimeType(content, fp.path)
			if detectedMime != "" {
				mimeType = detectedMime
			}
		}

		if mimeType == "" || !strings.HasPrefix(mimeType, "image/") {
			continue
		}

		return faviconURL, mimeType
	}

	return "", ""
}
