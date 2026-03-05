package handlers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaetanlhf/ZIMServer/internal/web/i18n"
	"github.com/gaetanlhf/ZIMServer/internal/web/services"
	"github.com/gaetanlhf/ZIMServer/internal/web/utils"
)

type ContentHandler struct {
	ArchiveService      *services.ArchiveService
	IllustrationService *services.IllustrationService
	FaviconService      *services.FaviconService
	Templates           TemplateRenderer
	I18n                *i18n.I18n
}

var timeZero = time.Time{}

func (h *ContentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	originalPath := r.URL.Path
	path := strings.TrimPrefix(originalPath, "/content/")

	if path != "" && !strings.Contains(path, "/") {
		if !strings.HasSuffix(originalPath, "/") {
			http.Redirect(w, r, originalPath+"/", http.StatusMovedPermanently)
			return
		}
	}

	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 0 {
		h.handle404(w, r, "", "")
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
			http.Error(w, "Failed to resolve main page", http.StatusInternalServerError)
			return
		}

		mainPageURL := fmt.Sprintf("/content/%s/%s", archiveName, resolvedPage.GetPath())
		http.Redirect(w, r, mainPageURL, http.StatusFound)
		return
	}

	if parts[1] == "favicon.ico" {
		h.handleIllustration(w, r, archive, archiveName)
		return
	}

	resourcePath := parts[1]
	h.handleResource(w, r, archive, resourcePath)
}

func (h *ContentHandler) handleResource(w http.ResponseWriter, r *http.Request, archive *services.Archive, resourcePath string) {
	entry, err := archive.FS.GetEntry(resourcePath)
	if err != nil {
		h.handle404(w, r, archive.Name, resourcePath)
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

		log.Printf("%s%s%s %sRedirect:%s %s -> %s",
			utils.ColorBlue, utils.SymbolRedirect, utils.ColorReset,
			utils.ColorGray, utils.ColorReset,
			resourcePath, targetPath)

		http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
		return
	}

	file, err := archive.FS.Open(resourcePath)
	if err != nil {
		h.handle404(w, r, archive.Name, resourcePath)
		return
	}
	defer file.Close()

	mimeType, _ := archive.Reader.GetMimeType(entry)
	if mimeType == "" {
		mimeType = utils.GuessMimeType(resourcePath)
	}

	if mimeType != "" {
		if mimeType == "text/html" && !strings.Contains(mimeType, "charset") {
			mimeType += "; charset=utf-8"
		}
		w.Header().Set("Content-Type", mimeType)
	}

	http.ServeContent(w, r, filepath.Base(resourcePath), timeZero, file.(http.File))
}

func (h *ContentHandler) handleIllustration(w http.ResponseWriter, r *http.Request, archive *services.Archive, archiveName string) {
	illustrationURL, mimeType := h.IllustrationService.GetIllustration(archive, archiveName)
	
	if illustrationURL == "" {
		illustrationURL, mimeType = h.FaviconService.GetFavicon(archive, archiveName)
	}

	if illustrationURL != "" {
		path := strings.TrimPrefix(illustrationURL, "/content/"+archiveName+"/")
		entry, err := archive.FS.GetEntry(path)
		if err == nil {
			content, err := archive.Reader.GetContent(entry)
			if err == nil {
				w.Header().Set("Content-Type", mimeType)
				w.Write(content)
				return
			}
		}
	}

	// Fallback to serving a default illustration or a 404
	http.NotFound(w, r)
}

func (h *ContentHandler) handle404(w http.ResponseWriter, r *http.Request, archiveName string, resourcePath string) {
	w.WriteHeader(http.StatusNotFound)
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
					data.HomeURL = fmt.Sprintf("/content/%s/%s", archiveName, resolvedPage.GetPath())
				}
			}
		}
	}

	w.Header().Set("Content-Language", lang)
	if err := h.Templates.Render(w, "404", data); err != nil {
		log.Printf("Template error: %v", err)
	}
}
