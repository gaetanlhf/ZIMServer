package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gaetanlhf/ZIMServer/internal/web/handlers"
	"github.com/gaetanlhf/ZIMServer/internal/web/i18n"
	"github.com/gaetanlhf/ZIMServer/internal/web/services"
	"github.com/gaetanlhf/ZIMServer/internal/web/templates"
	"github.com/gaetanlhf/ZIMServer/internal/web/utils"
)

type Server struct {
	archiveService      *services.ArchiveService
	illustrationService *services.IllustrationService
	faviconService      *services.FaviconService
	homeHandler *handlers.HomeHandler
	viewerHandler       *handlers.ViewerHandler
	contentHandler      *handlers.ContentHandler
	apiHandler          *handlers.APIHandler
	assetsHandler       http.Handler
	i18n                *i18n.I18n
}

func NewServer(version string) (*Server, error) {
	i18n, err := i18n.New()
	if err != nil {
		return nil, fmt.Errorf("failed to load i18n: %w", err)
	}

	tmpl, err := templates.Load(i18n)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	illustrationService := services.NewIllustrationService()
	faviconService := services.NewFaviconService()
	archiveService := services.NewArchiveService(illustrationService, faviconService)
	homeHandler := &handlers.HomeHandler{
		ArchiveService: archiveService,
		Templates:      tmpl,
		Version:        version,
		I18n:           i18n,
	}

	viewerHandler := &handlers.ViewerHandler{
		ArchiveService:      archiveService,
		IllustrationService: illustrationService,
		FaviconService:      faviconService,
		Templates:           tmpl,
		I18n:                i18n,
	}

	contentHandler := &handlers.ContentHandler{
		ArchiveService:      archiveService,
		IllustrationService: illustrationService,
		FaviconService:      faviconService,
		Templates:           tmpl,
		I18n:                i18n,
	}

	apiHandler := &handlers.APIHandler{
		ArchiveService: archiveService,
	}

	return &Server{
		archiveService:      archiveService,
		illustrationService: illustrationService,
		faviconService:      faviconService,
		homeHandler: homeHandler,
		viewerHandler:       viewerHandler,
		contentHandler:      contentHandler,
		apiHandler:          apiHandler,
		assetsHandler:       http.StripPrefix("/assets/", http.FileServer(templates.GetAssetsFS())),
		i18n:                i18n,
	}, nil
}

func (s *Server) LoadZIM(path string) error {
	return s.archiveService.LoadZIM(path)
}

func (s *Server) UnloadZIM(name string) error {
	return s.archiveService.UnloadZIM(name)
}

func (s *Server) ListArchives() []*services.Archive {
	return s.archiveService.ListArchives()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	withServerHeader(utils.LoggingMiddleware(http.HandlerFunc(s.serveHTTP))).ServeHTTP(w, r)
}

func withServerHeader(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "ZIMServer")
		h.ServeHTTP(w, r)
	})
}


func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/":
		s.homeHandler.ServeHTTP(w, r)
	case strings.HasPrefix(path, "/assets/"):
		s.assetsHandler.ServeHTTP(w, r)
	case strings.HasPrefix(path, "/viewer/"):
		s.viewerHandler.ServeHTTP(w, r)
	case strings.HasPrefix(path, "/content/"):
		s.contentHandler.ServeHTTP(w, r)
	case strings.HasPrefix(path, "/api/"):
		s.apiHandler.ServeHTTP(w, r)
	case strings.HasPrefix(path, "/catch"):
		s.viewerHandler.ServeHTTP(w, r)
	default:
		s.contentHandler.ServeHTTP(w, r)
	}
}
