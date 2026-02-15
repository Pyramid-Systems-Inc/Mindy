# Mindy - Progress

## MVP Phase 1 - Progress

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
- [x] Query endpoints (search, graph) - partial

## Completed

- Binary builds and runs
- Health endpoint works
- Data directory created at ~/.mindy/data

## Current Task

Testing the full pipeline (index + query)

## Notes

- Start with local file ingestion
- Build from bottom up: blob → vector → graph → api
- Test each component before moving to next
- Using random embedder for MVP (need real embeddings later)
