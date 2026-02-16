# Mindy - Specification

## Project Overview

- **Name:** Mindy
- **Type:** Personal AI memory / knowledge graph system
- **Goal:** Ingest local files, build vector + graph memory, query semantically
- **Target Users:** Single-user local-first

## Core Features

### Phase 1 (MVP) - Complete

1. **File Ingestion**
   - Watch directories for new/changed files (polling every 5s)
   - Manual ingest via API
   - Support: .txt, .md, .html, .json

2. **Blob Store**
   - Store raw file content by content-hash (SHA256)
   - Immutable storage
   - Located at `~/.mindy/data/blobs`

3. **Vector Index**
   - TF-IDF with hash-based vectorization (4096-dim)
   - Cosine similarity search
   - Located at `~/.mindy/data/vector`

4. **Graph Store**
   - Extract entities and relationships
   - Store in BadgerDB
   - Located at `~/.mindy/data/graph`

5. **Query API**
   - Semantic search via TF-IDF vectors
   - Graph traversal (BFS)
   - HTTP API on port 9090

### Phase 1.1 (Enhancements)

1. **Entity Extraction**
   - Emails (regex)
   - URLs (regex)
   - Phone numbers (regex)
   - Dates (regex)
   - Capitalized words (proper nouns)

2. **Persistence**
   - TF-IDF vectors saved to disk
   - Vocabulary and IDF scores persisted

3. **Better Search**
   - Normalized TF-IDF vectors
   - Cosine similarity ranking

## Configuration

| Flag | Config Key | Default | Description |
|------|------------|---------|-------------|
| `--watch` | `watch_paths` | none | Directories to watch (comma-separated) |
| `--port` | `http_port` | 9090 | API server port |
| `--data-dir` | `data_dir` | ~/.mindy/data | Data storage location |
| `--config` | - | none | Config file path |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |
| POST | /api/v1/ingest?path=<filepath> | Index file or directory |
| GET | /api/v1/search?q=<query>&k=<n> | Semantic search (default k=10) |
| GET | /api/v1/graph/node/{id} | Get node by ID |
| GET | /api/v1/graph/traverse?start=<id>&type=<edge>&depth=<n> | Graph traversal |
| GET | /api/v1/blob/{hash} | Get raw blob content |

## Data Flow

```
File → Blob Store (sha256) → Text Extract → TF-IDF Vector → Vector Index
                              ↘→ Entity Extract → Graph Store
```

## Query Flow

```
Semantic Query → TF-IDF Query Vector → Cosine Similarity → Ranked Results
                                                       ↓
                                              Return metadata + blob refs

Graph Query → BFS Traversal → Node/Edge Results
```

## Out of Scope (Phase 1)

- PDF/DOCX parsing (future)
- Web crawler (Phase 2)
- Connectors (GitHub, Gmail) (Phase 2)
- Multi-node coordination (Phase 3)
- Multi-tenant / auth (Phase 3)
- Web UI (future)

## Technology

| Component | Implementation |
|-----------|---------------|
| Language | Go 1.21+ |
| Storage | Local filesystem + BadgerDB |
| Vector | Custom TF-IDF (hash-based) |
| HTTP | Chi router |
| No external dependencies | ✓ |
