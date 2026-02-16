# Mindy - Progress

## Phase 1 Complete ✓

### Core Features
- [x] Project initialization (go.mod)
- [x] Configuration system (CLI + config file)
- [x] Blob store implementation
- [x] Graph store (BadgerDB)
- [x] Vector index (custom IVF)
- [x] Text extraction (multiple file types)
- [x] Embedder interface
- [x] File watcher / ingestion
- [x] Indexer pipeline
- [x] HTTP API server
- [x] Query endpoints (search, graph)

### Phase 1.1 Enhancements
- [x] Fixed /ingest API to actually call indexer
- [x] Added vector persistence (save on add/close)
- [x] Implemented TF-IDF embedder (local-only, no external deps)
- [x] Improved entity extraction with regex patterns (email, URL, phone, date)

### Phase 1.2 Enhancements
- [x] Enhanced TF-IDF with:
  - Stopword filtering (60+ common English words)
  - Simple stemming (15 common suffixes)
  - BM25 ranking support (optional)
  - Increased dimension (8192)
  - Document length tracking
  - Better IDF smoothing

### Phase 1.3 Enhancements (Core System) - COMPLETE ✓
- [x] **PDF/DOCX Support** - Text extraction from PDF and DOCX files
- [x] **Incremental Indexing** - File tracker tracks hashes, skips unchanged files
- [x] **Search Filters** - Filter by file type and path
- [x] **Pagination** - Offset/limit support with next_offset
- [x] **Metadata Extraction** - File properties, content types
- [x] **CLI Improvements** - --help, --version flags, better output
- [x] **Graceful Shutdown** - Proper cleanup on signals (10s timeout)
- [x] **Index Stats API** - GET /api/v1/stats endpoint
- [x] **Reindex API** - POST /api/v1/reindex for full reindex
- [x] **Graph Search** - GET /api/v1/graph/search for node lookup
- [x] **Web UI** - Built-in web interface at /ui

### Phase 1.4 Enhancements (Search Quality) - COMPLETE ✓
- [x] **N-gram Indexing** - Bigrams and trigrams for phrase matching (1-3 range)
- [x] **Code-aware Tokenization** - Splits camelCase, snake_case, kebab-case, PascalCase
- [x] **BM25 by Default** - Enabled BM25 ranking as default algorithm
- [x] **Synonym Expansion** - Local synonym map with 500+ programming terms
- [x] **Fuzzy Matching** - Levenshtein distance for typo tolerance (threshold: 2)
- [x] **Auto-reindex** - Version tracking triggers automatic reindex on config changes

## New API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check with timestamp |
| POST | /api/v1/ingest | Index file/directory |
| POST | /api/v1/reindex | Reindex all tracked files |
| GET | /api/v1/search | Semantic search with filters |
| GET | /api/v1/stats | Index statistics |
| GET | /api/v1/graph/search | Search nodes by type/label |
| GET | /api/v1/graph/node/{id} | Get node by ID |
| GET | /api/v1/graph/traverse | Graph traversal |
| GET | /api/v1/blob/{hash} | Get blob content |

## Search Filters
```
GET /api/v1/search?q=<query>&k=<n>&offset=<n>&type=<type>&path=<path>
```

## Testing Status

| Component | Tests | Status |
|-----------|-------|--------|
| TF-IDF Embedder | 14 | ✓ Passing |
| Blob Store | 4 | ⚠ Needs disk space |
| Graph Store | 3 | ⚠ Needs disk space |
| Indexer | 4 | ⚠ Needs disk space |
| Extractor | 6 | ✓ Passing |

Run tests: `go test ./...`

## What's Left (Future)
- [ ] Batch import/export
- [ ] Web crawler (Phase 2)
- [ ] Connectors (GitHub, Gmail) (Phase 2)

## Technology Stack
| Component | Implementation |
|-----------|---------------|
| Language | Go 1.21+ |
| Storage | Local filesystem + BadgerDB |
| Vector | Custom TF-IDF/BM25 (hash-based) |
| HTTP | Chi router |
| No external dependencies | ✓ |
