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

const webUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mindy - Personal Knowledge Graph</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; color: #333; line-height: 1.6; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 2rem; text-align: center; }
        .header h1 { font-size: 2.5rem; margin-bottom: 0.5rem; }
        .container { max-width: 1200px; margin: 0 auto; padding: 2rem; }
        .search-box { background: white; border-radius: 12px; padding: 2rem; box-shadow: 0 4px 6px rgba(0,0,0,0.1); margin-bottom: 2rem; }
        .search-box h2 { margin-bottom: 1rem; color: #667eea; }
        .search-form { display: flex; gap: 1rem; flex-wrap: wrap; }
        .search-input { flex: 1; min-width: 250px; padding: 0.75rem 1rem; font-size: 1rem; border: 2px solid #e0e0e0; border-radius: 8px; }
        .search-input:focus { outline: none; border-color: #667eea; }
        .btn { padding: 0.75rem 1.5rem; font-size: 1rem; border: none; border-radius: 8px; cursor: pointer; }
        .btn-primary { background: #667eea; color: white; }
        .results { background: white; border-radius: 12px; padding: 1.5rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .result-item { padding: 1rem; border-bottom: 1px solid #e0e0e0; }
        .result-item:hover { background: #f9f9f9; }
        .result-score { display: inline-block; background: #667eea; color: white; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.875rem; margin-bottom: 0.5rem; }
        .result-meta { color: #666; font-size: 0.875rem; }
        .tabs { display: flex; gap: 0.5rem; margin-bottom: 1rem; border-bottom: 2px solid #e0e0e0; padding-bottom: 0.5rem; }
        .tab { padding: 0.5rem 1rem; border: none; background: none; cursor: pointer; font-size: 1rem; color: #666; }
        .tab.active { color: #667eea; background: rgba(102, 126, 234, 0.1); border-radius: 8px 8px 0 0; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .loading { text-align: center; padding: 2rem; color: #666; }
        .api-endpoints { background: white; border-radius: 12px; padding: 1.5rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-top: 2rem; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Mindy</h1>
        <p>Personal AI Memory & Knowledge Graph</p>
    </div>
    <div class="container">
        <div class="search-box">
            <h2>Search</h2>
            <form class="search-form" id="searchForm">
                <input type="text" class="search-input" id="searchInput" placeholder="Ask anything..." autocomplete="off">
                <button type="submit" class="btn btn-primary">Search</button>
            </form>
        </div>
        <div class="tabs">
            <button class="tab active" data-tab="results">Results</button>
            <button class="tab" data-tab="api">API</button>
        </div>
        <div class="tab-content active" id="results">
            <div class="results" id="resultsContainer"><p style="text-align: center; color: #666;">Enter a search query</p></div>
        </div>
        <div class="tab-content" id="api">
            <div class="api-endpoints">
                <h3>API Endpoints</h3>
                <p><strong>GET /health</strong> - Health check</p>
                <p><strong>POST /api/v1/ingest?path=&lt;path&gt;</strong> - Index file/directory</p>
                <p><strong>GET /api/v1/search?q=&lt;query&gt;&amp;k=10</strong> - Semantic search</p>
                <p><strong>GET /api/v1/stats</strong> - Index statistics</p>
            </div>
        </div>
    </div>
    <script>
        const API_BASE = window.location.origin;
        document.getElementById('searchForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const query = document.getElementById('searchInput').value;
            if (!query) return;
            const container = document.getElementById('resultsContainer');
            container.innerHTML = '<div class="loading">Searching...</div>';
            try {
                const response = await fetch(API_BASE + '/api/v1/search?q=' + encodeURIComponent(query) + '&k=20');
                const data = await response.json();
                if (data.results && data.results.length > 0) {
                    container.innerHTML = data.results.map(r => {
                        let meta = {};
                        try { meta = JSON.parse(r.meta); } catch (e) {}
                        return '<div class="result-item"><span class="result-score">' + (r.score * 100).toFixed(1) + '%</span><div class="result-meta"><strong>' + (meta.path || r.id) + '</strong></div></div>';
                    }).join('');
                } else {
                    container.innerHTML = '<p style="text-align: center; color: #666;">No results found</p>';
                }
            } catch (e) {
                container.innerHTML = '<p style="color: red;">Error: ' + e.message + '</p>';
            }
        });
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', () => {
                document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
                document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
                tab.classList.add('active');
                document.getElementById(tab.dataset.tab).classList.add('active');
            });
        });
    </script>
</body>
</html>`

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

	r.Get("/", s.serveWebUI)
	r.Get("/ui", s.serveWebUI)
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

func (s *Server) serveWebUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(webUIHTML))
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
