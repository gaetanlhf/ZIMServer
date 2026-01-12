package handlers

import (
	"log"
	"net/http"

	"github.com/gaetanlhf/ZIMServer/internal/web/services"
)

type HomeHandler struct {
	ArchiveService *services.ArchiveService
	Templates      TemplateRenderer
	Version        string
}

type HomeData struct {
	Archives   []*services.Archive
	Count      int
	Languages  []services.LanguageInfo
	Categories []string
	Version    string
}

type TemplateRenderer interface {
	Render(w http.ResponseWriter, name string, data interface{}) error
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	archives := h.ArchiveService.ListArchives()

	data := HomeData{
		Archives:   archives,
		Count:      len(archives),
		Languages:  h.ArchiveService.GetLanguages(),
		Categories: h.ArchiveService.GetCategories(),
		Version:    h.Version,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.Templates.Render(w, "home", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}
