package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type SearchService struct{}

type SearchResponse struct {
	Query   string         `json:"query"`
	Archive string         `json:"archive"`
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
	Time    string         `json:"time"`
}

type SearchResult struct {
	Title      string  `json:"title"`
	Path       string  `json:"path"`
	Namespace  string  `json:"namespace"`
	Score      float64 `json:"score"`
	URL        string  `json:"url"`
	IsRedirect bool    `json:"isRedirect"`
}

func NewSearchService() *SearchService {
	return &SearchService{}
}

func (s *SearchService) HandleSearch(w http.ResponseWriter, r *http.Request, archive *Archive) {
	if archive.IndexMgr == nil {
		http.Error(w, "Search not available for this archive", http.StatusServiceUnavailable)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	maxResults := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			maxResults = limit
		}
	}

	start := time.Now()
	results, err := archive.IndexMgr.Search(query, maxResults)
	elapsed := time.Since(start)

	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := SearchResponse{
		Query:   query,
		Archive: archive.Name,
		Results: make([]SearchResult, 0, len(results)),
		Count:   len(results),
		Time:    fmt.Sprintf("%.2fms", float64(elapsed.Microseconds())/1000.0),
	}

	for _, result := range results {
		response.Results = append(response.Results, SearchResult{
			Title:      result.Entry.GetTitle(),
			Path:       result.Entry.GetPath(),
			Namespace:  string(result.Entry.GetNamespace()),
			Score:      result.Score,
			URL:        fmt.Sprintf("/zim/%s/%s", archive.Name, result.Entry.GetPath()),
			IsRedirect: result.Entry.IsRedirect(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Search [%s]: '%s' -> %d results in %s", archive.Name, query, len(results), elapsed)
}
