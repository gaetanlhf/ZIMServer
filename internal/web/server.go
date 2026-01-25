package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gaetanlhf/ZIMServer/internal/web/handlers"
	"github.com/gaetanlhf/ZIMServer/internal/web/services"
	"github.com/gaetanlhf/ZIMServer/internal/web/templates"
	"github.com/gaetanlhf/ZIMServer/internal/web/utils"
)

type Server struct {
	archiveService *services.ArchiveService
	faviconService *services.FaviconService
	searchService  *services.SearchService
	homeHandler    *handlers.HomeHandler
	viewerHandler  *handlers.ViewerHandler
	contentHandler *handlers.ContentHandler
	apiHandler     *handlers.APIHandler
}

func NewServer(version string) (*Server, error) {
	tmpl, err := templates.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	archiveService := services.NewArchiveService()
	faviconService := services.NewFaviconService()
	searchService := services.NewSearchService()

	return &Server{
		archiveService: archiveService,
		faviconService: faviconService,
		searchService:  searchService,
		homeHandler: &handlers.HomeHandler{
			ArchiveService: archiveService,
			Templates:      tmpl,
			Version:        version,
		},
		viewerHandler: &handlers.ViewerHandler{
			ArchiveService: archiveService,
			FaviconService: faviconService,
			Templates:      tmpl,
		},
		contentHandler: &handlers.ContentHandler{
			ArchiveService: archiveService,
			FaviconService: faviconService,
			Templates:      tmpl,
		},
		apiHandler: &handlers.APIHandler{
			ArchiveService: archiveService,
			SearchService:  searchService,
		},
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
	utils.LoggingMiddleware(http.HandlerFunc(s.serveHTTP)).ServeHTTP(w, r)
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/":
		s.homeHandler.ServeHTTP(w, r)
	case strings.HasPrefix(path, "/assets/"):
		http.StripPrefix("/assets/", http.FileServer(templates.GetAssetsFS())).ServeHTTP(w, r)
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
