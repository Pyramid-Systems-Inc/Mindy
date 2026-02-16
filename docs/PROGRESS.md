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

### Phase 1.2 Enhancements (Just Completed)
- [x] Enhanced TF-IDF with:
  - Stopword filtering (60+ common English words)
  - Simple stemming (15 common suffixes)
  - BM25 ranking support (optional)
  - Increased dimension (8192)
  - Document length tracking
  - Better IDF smoothing
- [x] Comprehensive documentation:
  - ARCHITECTURE.md - Detailed system design
  - SPEC.md - Technical specification
  - USAGE.md - User guide with examples
- [x] Extended test suite (11 tests for TF-IDF)

## Testing Status

| Component | Tests | Status |
|-----------|-------|--------|
| TF-IDF Embedder | 11 | ✓ Passing |
| Blob Store | 4 | ⚠ Needs disk space |
| Graph Store | 3 | ⚠ Needs disk space |
| Indexer | 4 | ⚠ Needs disk space |

Run tests: `go test ./...`

## Next Steps (Phase 1.3)
1. Add PDF support to indexer
2. Improve search API (filters, pagination)
3. Add better CLI usage/help
4. Add BM25 as primary algorithm

## Technology Stack
| Component | Implementation |
|-----------|---------------|
| Language | Go 1.21+ |
| Storage | Local filesystem + BadgerDB |
| Vector | Custom TF-IDF/BM25 (hash-based) |
| HTTP | Chi router |
| No external dependencies | ✓ |
