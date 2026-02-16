# Mindy - Progress

## MVP Phase 1 - Complete

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

## Phase 1.1 Fixes (Completed)

- [x] Fixed /ingest API to actually call indexer
- [x] Added vector persistence (save on add/close)
- [x] Implemented TF-IDF embedder (local-only, no external deps)
- [x] Improved entity extraction with regex patterns (email, URL, phone, date)

## Current Task

Testing the full pipeline (ingest → index → search)

## Technology

| Component | Implementation |
|-----------|---------------|
| Embedding | TF-IDF with hash-based vectorization |
| Storage | Local filesystem + BadgerDB |
| API | Chi router |

## Next Steps

1. Test full pipeline end-to-end
2. Add more file type support (PDF, DOCX)
3. Consider coordination plane (Phase 2)
