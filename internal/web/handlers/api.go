package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gaetanlhf/ZIMServer/internal/web/services"
)

type APIHandler struct {
	ArchiveService *services.ArchiveService
	SearchService  *services.SearchService
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

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	archiveName := parts[0]
	action := parts[1]

	archive, exists := h.ArchiveService.GetArchive(archiveName)
	if !exists {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "search":
		h.handleSearch(w, r, archive)
	case "random":
		h.handleRandom(w, r, archive)
	default:
		http.NotFound(w, r)
	}
}

func (h *APIHandler) handleSearch(w http.ResponseWriter, r *http.Request, archive *services.Archive) {
	if archive.IndexMgr == nil {
		http.Error(w, "Search not available", http.StatusServiceUnavailable)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l == -1 {
				limit = -1
			} else if l > 0 {
				limit = l
			}
		}
	}

	results, err := archive.IndexMgr.Search(query, limit)
	if err != nil {
		log.Printf("Search error: %v", err)
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := APISearchResponse{
		Query:   query,
		Results: make([]APISearchResult, 0, len(results)),
		Count:   len(results),
	}

	for _, result := range results {
		response.Results = append(response.Results, APISearchResult{
			Title: result.Entry.GetTitle(),
			Path:  result.Entry.GetPath(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *APIHandler) handleRandom(w http.ResponseWriter, r *http.Request, archive *services.Archive) {
	if archive.IndexMgr == nil {
		log.Printf("Random failed: IndexMgr is nil for archive %s", archive.Name)
		http.Error(w, "Random not available for this archive", http.StatusServiceUnavailable)
		return
	}

	entry, err := archive.IndexMgr.GetRandomArticle()
	if err != nil {
		log.Printf("Random error for archive %s: %v", archive.Name, err)
		http.Error(w, fmt.Sprintf("Random failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := APIRandomResponse{
		Title: entry.GetTitle(),
		Path:  entry.GetPath(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
