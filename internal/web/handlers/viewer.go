package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gaetanlhf/ZIMServer/internal/web/i18n"
	"github.com/gaetanlhf/ZIMServer/internal/web/services"
)

type ViewerHandler struct {
	ArchiveService      *services.ArchiveService
	IllustrationService *services.IllustrationService
	FaviconService      *services.FaviconService
	Templates           TemplateRenderer
	I18n                *i18n.I18n
}

type ViewerData struct {
	ArchiveName     string
	ArchiveTitle    string
	EntryPath       string
	IllustrationURL string
	IllustrationType string
	FaviconURL      string
	FaviconType     string
	HasIndex        bool
	IsCatch         bool
	CatchURL        string
	CatchSrc        template.URL
	Lang            string
}

func (h *ViewerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	originalPath := r.URL.Path

	if strings.HasPrefix(originalPath, "/catch") {
		h.handleCatch(w, r)
		return
	}

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
		h.handle404(w, r, "", "")
		return
	}

	if len(parts) == 1 || parts[1] == "" {
		mainPage, err := archive.Reader.GetMainPage()
		if err != nil {
			h.handle404(w, r, archiveName, "")
			return
		}

		resolvedPage, err := archive.Reader.ResolveRedirect(mainPage)
		if err != nil {
			log.Printf("Failed to resolve main page for %s: %v", archiveName, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		mainPageURL := fmt.Sprintf("/viewer/%s/%s", archiveName, resolvedPage.GetPath())
		http.Redirect(w, r, mainPageURL, http.StatusFound)
		return
	}

	entryPath := parts[1]

	illustrationURL, illustrationType := h.IllustrationService.GetIllustration(archive, archiveName)
	if illustrationURL == "" {
		illustrationURL, illustrationType = h.FaviconService.GetFavicon(archive, archiveName)
	}
	
	faviconURL, faviconType := h.FaviconService.GetFavicon(archive, archiveName)

	hasIndex := archive.IndexMgr != nil
	lang := h.I18n.GetLanguage(r)

	data := ViewerData{
		ArchiveName:     archiveName,
		ArchiveTitle:    archive.Metadata.Title,
		EntryPath:       entryPath,
		IllustrationURL: illustrationURL,
		IllustrationType: illustrationType,
		FaviconURL:      faviconURL,
		FaviconType:     faviconType,
		HasIndex:        hasIndex,
		IsCatch:         false,
		Lang:            lang,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", lang)

	if err := h.Templates.Render(w, "viewer", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (h *ViewerHandler) handleCatch(w http.ResponseWriter, r *http.Request) {
	viewer := r.URL.Query().Get("viewer")
	catchURL := r.URL.Query().Get("url")
	lang := h.I18n.GetLanguage(r)

	if viewer == "" {
		data := struct {
			Url  string
			Lang string
		}{
			Url:  catchURL,
			Lang: lang,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Language", lang)
		if err := h.Templates.Render(w, "catch", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("Template error: %v", err)
		}
		return
	}

	archive, exists := h.ArchiveService.GetArchive(viewer)
	if !exists {
		h.handle404(w, r, "", "")
		return
	}

	illustrationURL, illustrationType := h.IllustrationService.GetIllustration(archive, viewer)
	if illustrationURL == "" {
		illustrationURL, illustrationType = h.FaviconService.GetFavicon(archive, viewer)
	}
	faviconURL, faviconType := h.FaviconService.GetFavicon(archive, viewer)
	
	hasIndex := archive.IndexMgr != nil

	data := ViewerData{
		ArchiveName:     viewer,
		ArchiveTitle:    archive.Metadata.Title,
		IllustrationURL: illustrationURL,
		IllustrationType: illustrationType,
		FaviconURL:      faviconURL,
		FaviconType:     faviconType,
		HasIndex:        hasIndex,
		IsCatch:         true,
		CatchURL:        catchURL,
		CatchSrc:        template.URL("/catch?url=" + url.QueryEscape(catchURL)),
		Lang:            lang,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", lang)

	if err := h.Templates.Render(w, "viewer", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (h *ViewerHandler) handle404(w http.ResponseWriter, r *http.Request, archiveName string, resourcePath string) {
	lang := h.I18n.GetLanguage(r)

	data := struct {
		Url     string
		HomeURL string
		Lang    string
	}{
		Url:  r.URL.Path,
		Lang: lang,
	}

	if archiveName != "" {
		archive, exists := h.ArchiveService.GetArchive(archiveName)
		if exists {
			mainPage, err := archive.Reader.GetMainPage()
			if err == nil {
				resolvedPage, err := archive.Reader.ResolveRedirect(mainPage)
				if err == nil {
					data.HomeURL = fmt.Sprintf("/viewer/%s/%s", archiveName, resolvedPage.GetPath())
				}
			}
		}
	}

	w.Header().Set("Content-Language", lang)
	w.WriteHeader(http.StatusNotFound)
	if err := h.Templates.Render(w, "404", data); err != nil {
		log.Printf("Template error: %v", err)
	}
}
