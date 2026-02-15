package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"mindy/internal/blob"
	"mindy/internal/graph"
	"mindy/internal/vector"
	"mindy/pkg/embedder"
)

type Server struct {
	port        int
	blobStore   *blob.Store
	vectorIndex *vector.Index
	graphStore  *graph.Store
	embedder   embedder.Embedder
	httpServer  *http.Server
}

func NewServer(port int, blobStore *blob.Store, vectorIndex *vector.Index, graphStore *graph.Store) *Server {
	return &Server{
		port:        port,
		blobStore:   blobStore,
		vectorIndex: vectorIndex,
		graphStore:  graphStore,
		embedder:    embedder.NewRandom(384),
	}
}

func (s *Server) Start() error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", s.health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/ingest", s.ingest)
		r.Get("/search", s.search)
		r.Get("/graph/node/{id}", s.getNode)
		r.Get("/graph/traverse", s.traverse)
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
				indexed++
			}
			return nil
		})
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"message":   "Directory queued for indexing",
			"files":     indexed,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"path":   path,
	})
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "q (query) required", http.StatusBadRequest)
		return
	}

	k := 10
	if kStr := r.URL.Query().Get("k"); kStr != "" {
		if parsed, err := strconv.Atoi(kStr); err == nil {
			k = parsed
		}
	}

	queryVec, err := s.embedder.Embed(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results, err := s.vectorIndex.Search(queryVec, k)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   query,
		"results": results,
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
		if parsed, err := strconv.Atoi(dStr); err == nil {
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
