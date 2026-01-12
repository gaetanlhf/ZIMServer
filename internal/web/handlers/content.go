package handlers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaetanlhf/ZIMServer/internal/web/services"
	"github.com/gaetanlhf/ZIMServer/internal/web/utils"
	zimreader "github.com/gaetanlhf/ZIMServer/internal/zim/reader"
)

type ContentHandler struct {
	ArchiveService *services.ArchiveService
	FaviconService *services.FaviconService
}

var timeZero = time.Time{}

func (h *ContentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	originalPath := r.URL.Path
	path := strings.TrimPrefix(originalPath, "/content/")
	path = strings.TrimSuffix(path, "/")

	if path != "" && !strings.Contains(path, "/") {
		if !strings.HasSuffix(originalPath, "/") {
			http.Redirect(w, r, originalPath+"/", http.StatusMovedPermanently)
			return
		}
	}

	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 0 {
		http.NotFound(w, r)
		return
	}

	archiveName := parts[0]
	archive, exists := h.ArchiveService.GetArchive(archiveName)
	if !exists {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 1 || parts[1] == "" {
		mainPage, err := archive.Reader.GetMainPage()
		if err != nil {
			http.Error(w, "No main page found", http.StatusNotFound)
			return
		}

		resolvedPage, err := archive.Reader.ResolveRedirect(mainPage)
		if err != nil {
			http.Error(w, "Failed to resolve main page", http.StatusInternalServerError)
			return
		}

		mainPageURL := fmt.Sprintf("/content/%s/%s", archiveName, resolvedPage.GetPath())
		http.Redirect(w, r, mainPageURL, http.StatusFound)
		return
	}

	if parts[1] == "favicon.ico" {
		h.handleFavicon(w, r, archive)
		return
	}

	resourcePath := parts[1]
	h.handleResource(w, r, archive, resourcePath)
}

func (h *ContentHandler) handleResource(w http.ResponseWriter, r *http.Request, archive *services.Archive, resourcePath string) {
	entry, err := archive.FS.GetEntry(resourcePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if entry.IsRedirect() {
		resolvedEntry, err := archive.Reader.ResolveRedirect(entry)
		if err != nil {
			http.Error(w, "Failed to resolve redirect", http.StatusInternalServerError)
			log.Printf("Redirect resolution error for %s: %v", resourcePath, err)
			return
		}

		targetPath := resolvedEntry.GetPath()
		redirectURL := fmt.Sprintf("/content/%s/%s", archive.Name, targetPath)

		log.Printf("Redirect: %s -> %s", resourcePath, targetPath)

		http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
		return
	}

	file, err := archive.FS.Open(resourcePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	mimeType := utils.GuessMimeType(resourcePath)
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}

	http.ServeContent(w, r, filepath.Base(resourcePath), timeZero, file.(http.File))
}

func (h *ContentHandler) handleFavicon(w http.ResponseWriter, r *http.Request, archive *services.Archive) {
	faviconPaths := []struct {
		namespace byte
		path      string
	}{
		{zimreader.NamespaceWellKnown, "favicon"},
		{zimreader.NamespaceWellKnown, "favicon.png"},
		{zimreader.NamespaceWellKnown, "favicon.ico"},
		{zimreader.NamespaceContent, "favicon"},
		{zimreader.NamespaceContent, "favicon.png"},
		{zimreader.NamespaceContent, "favicon.ico"},
		{zimreader.NamespaceMetadata, "Illustration_48x48@1"},
		{zimreader.NamespaceMetadata, "Illustration_96x96@2"},
	}

	for _, fp := range faviconPaths {
		entry, err := archive.Reader.GetEntryByURL(fp.namespace, fp.path)
		if err != nil {
			continue
		}

		content, err := archive.Reader.GetContent(entry)
		if err != nil {
			continue
		}

		mimeType, _ := archive.Reader.GetMimeType(entry)
		if mimeType == "" {
			mimeType = "image/png"
		}

		w.Header().Set("Content-Type", mimeType)
		w.Write(content)
		return
	}

	http.NotFound(w, r)
}
