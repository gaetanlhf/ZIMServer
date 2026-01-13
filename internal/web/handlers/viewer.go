package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gaetanlhf/ZIMServer/internal/web/services"
)

type ViewerHandler struct {
	ArchiveService *services.ArchiveService
	FaviconService *services.FaviconService
	Templates      TemplateRenderer
}

type ViewerData struct {
	ArchiveName  string
	ArchiveTitle string
	EntryPath    string
	FaviconURL   string
	FaviconType  string
	HasIndex     bool
}

func (h *ViewerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	originalPath := r.URL.Path
	path := strings.TrimPrefix(originalPath, "/viewer/")

	if path != "" && !strings.Contains(path, "/") {
		if !strings.HasSuffix(originalPath, "/") {
			http.Redirect(w, r, originalPath+"/", http.StatusMovedPermanently)
			return
		}
	}

	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		http.Redirect(w, r, "/", http.StatusFound)
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

		mainPageURL := fmt.Sprintf("/viewer/%s/%s", archiveName, resolvedPage.GetPath())
		http.Redirect(w, r, mainPageURL, http.StatusFound)
		return
	}

	entryPath := parts[1]

	faviconURL, faviconType := h.FaviconService.GetFaviconInfo(archive, archiveName)

	hasIndex := archive.IndexMgr != nil

	data := ViewerData{
		ArchiveName:  archiveName,
		ArchiveTitle: archive.Metadata.Title,
		EntryPath:    entryPath,
		FaviconURL:   faviconURL,
		FaviconType:  faviconType,
		HasIndex:     hasIndex,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.Templates.Render(w, "viewer", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}
