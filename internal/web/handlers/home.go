package handlers

import (
	"log"
	"net/http"

	"github.com/gaetanlhf/ZIMServer/internal/web/i18n"
	"github.com/gaetanlhf/ZIMServer/internal/web/services"
)

type HomeHandler struct {
	ArchiveService *services.ArchiveService
	Templates      TemplateRenderer
	Version        string
	I18n           *i18n.I18n
}

type HomeData struct {
	Archives       []*services.Archive
	Count          int
	Languages      []services.LanguageInfo
	Categories     []string
	Version        string
	Lang           string
	AvailableLangs []i18n.LanguageInfo
	Translations   interface{}
}

type TemplateRenderer interface {
	Render(w http.ResponseWriter, name string, data interface{}) error
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	archives := h.ArchiveService.ListArchives()
	lang := h.I18n.GetLanguage(r)

	data := HomeData{
		Archives:       archives,
		Count:          len(archives),
		Languages:      h.ArchiveService.GetLanguages(),
		Categories:     h.ArchiveService.GetCategories(),
		Version:        h.Version,
		Lang:           lang,
		AvailableLangs: h.I18n.GetAvailableLanguages(),
		Translations:   h.I18n.Translate(lang, "home"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", lang)

	if err := h.Templates.Render(w, "home", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}
