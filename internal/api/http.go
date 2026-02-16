package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"mindy/internal/blob"
	"mindy/internal/graph"
	"mindy/internal/indexer"
	"mindy/internal/vector"
	"mindy/pkg/embedder"
)

type Server struct {
	port        int
	blobStore   *blob.Store
	vectorIndex *vector.Index
	graphStore  *graph.Store
	indexer    *indexer.Indexer
	embedder   embedder.Embedder
	httpServer  *http.Server
}

func NewServer(port int, blobStore *blob.Store, vectorIndex *vector.Index, graphStore *graph.Store, idx *indexer.Indexer) *Server {
	var tfidf *embedder.TFIDF
	if idx != nil {
		tfidf = idx.GetEmbedder()
	}
	return &Server{
		port:        port,
		blobStore:   blobStore,
		vectorIndex: vectorIndex,
		graphStore:  graphStore,
		indexer:     idx,
		embedder:    tfidf,
	}
}

func (s *Server) Start() error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", s.health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/ingest", s.ingest)
		r.Post("/reindex", s.reindex)
		r.Get("/search", s.search)
		r.Get("/stats", s.stats)
		r.Get("/graph/node/{id}", s.getNode)
		r.Get("/graph/traverse", s.traverse)
		r.Get("/graph/search", s.searchNodes)
		r.Get("/blob/{hash}", s.getBlob)
	})

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: r,
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() {
	if s.httpServer != nil {
		s.httpServer.Close()
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if info.IsDir() {
		var indexed int
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				if s.indexer != nil {
					go s.indexer.IndexFile(p)
				}
				indexed++
			}
			return nil
		})
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"message": "Directory queued for indexing",
			"files":   indexed,
		})
		return
	}

	if s.indexer != nil {
		if err := s.indexer.IndexFile(path); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"path":   path,
	})
}

func (s *Server) reindex(w http.ResponseWriter, r *http.Request) {
	if s.indexer != nil {
		go s.indexer.ReindexAll()
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "Reindex started in background",
		})
		return
	}
	
	http.Error(w, "indexer not available", http.StatusServiceUnavailable)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "q (query) required", http.StatusBadRequest)
		return
	}

	k := 10
	if kStr := r.URL.Query().Get("k"); kStr != "" {
		if parsed, err := strconv.Atoi(kStr); err == nil && parsed > 0 && parsed <= 100 {
			k = parsed
		}
	}

	offset := 0
	if offStr := r.URL.Query().Get("offset"); offStr != "" {
		if parsed, err := strconv.Atoi(offStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	fileType := r.URL.Query().Get("type")
	pathFilter := r.URL.Query().Get("path")

	queryVec, err := s.embedder.Embed(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	allResults, err := s.vectorIndex.Search(queryVec, k+offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var filteredResults []SearchResult
	
	for _, result := range allResults {
		if len(filteredResults) >= k {
			break
		}
		
		if fileType != "" {
			if !strings.Contains(result.Meta, `"file_type":"`+fileType) && 
			   !strings.Contains(result.Meta, `"content_type":"`+fileType) {
				continue
			}
		}
		
		if pathFilter != "" && !strings.Contains(result.Meta, pathFilter) {
			continue
		}
		
		filteredResults = append(filteredResults, SearchResult{
			ID:    result.ID,
			Score: result.Score,
			Meta:  result.Meta,
		})
	}

	if offset > len(filteredResults) {
		filteredResults = []SearchResult{}
	} else if offset > 0 && offset < len(filteredResults) {
		filteredResults = filteredResults[offset:]
	}

	response := SearchResponse{
		Query:      query,
		Results:    filteredResults,
		Total:     len(allResults),
		Offset:    offset,
		Limit:     k,
		Page:      offset/k + 1,
	}
	
	if len(allResults) > offset+k {
		response.NextOffset = offset + k
	}

	json.NewEncoder(w).Encode(response)
}

type SearchResponse struct {
	Query      string          `json:"query"`
	Results    []SearchResult  `json:"results"`
	Total      int             `json:"total"`
	Offset     int             `json:"offset"`
	Limit      int             `json:"limit"`
	Page       int             `json:"page"`
	NextOffset int             `json:"next_offset,omitempty"`
}

type SearchResult struct {
	ID    string  `json:"id"`
	Score float32 `json:"score"`
	Meta  string  `json:"meta"`
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})
	
	if s.indexer != nil {
		stats = s.indexer.GetStats()
	}
	
	stats["indexer"] = map[string]interface{}{
		"files_indexed": s.indexer.GetFileCount(),
	}
	
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) searchNodes(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "q (query) required", http.StatusBadRequest)
		return
	}

	limit := 20
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	nodeType := r.URL.Query().Get("type")

	typeFilter := map[string]string{
		"document": "Document",
		"chunk":    "Chunk",
		"entity":   "Entity",
	}
	
	if nodeType != "" {
		if t, ok := typeFilter[nodeType]; ok {
			nodeType = t
		}
	}

	nodes := s.graphStore.SearchNodes(nodeType, query, limit)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"query": query,
		"nodes": nodes,
		"count": len(nodes),
	})
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	node, err := s.graphStore.GetNode(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(node)
}

func (s *Server) traverse(w http.ResponseWriter, r *http.Request) {
	start := r.URL.Query().Get("start")
	if start == "" {
		http.Error(w, "start required", http.StatusBadRequest)
		return
	}

	edgeType := r.URL.Query().Get("type")
	depth := 3
	if dStr := r.URL.Query().Get("depth"); dStr != "" {
		if parsed, err := strconv.Atoi(dStr); err == nil && parsed > 0 && parsed <= 10 {
			depth = parsed
		}
	}

	nodes, err := s.graphStore.Traverse(start, edgeType, depth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"start": start,
		"nodes": nodes,
		"count": len(nodes),
	})
}

func (s *Server) getBlob(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if hash == "" {
		http.Error(w, "hash required", http.StatusBadRequest)
		return
	}

	data, err := s.blobStore.Get(hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
}
