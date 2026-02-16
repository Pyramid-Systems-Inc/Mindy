# Mindy - Specification

## Project Overview

- **Name:** Mindy
- **Type:** Personal AI memory / knowledge graph system
- **Goal:** Ingest local files, build vector + graph memory, query semantically
- **Target Users:** Single-user local-first

## Design Goals

1. **Local-First**: All data stays on the user's machine
2. **Privacy**: No external APIs, no cloud dependencies
3. **Simplicity**: Single binary, minimal setup
4. **Extensibility**: Plugin architecture for future features

## Core Features

### Phase 1 (MVP) - Complete

1. **File Ingestion**
   - Watch directories for new/changed files (polling every 5s)
   - Manual ingest via API
   - Support: .txt, .md, .html, .json, .xml, .csv, .log, .pdf, .docx

2. **Blob Store**
   - Store raw file content by content-hash (SHA256)
   - Immutable storage
   - Located at `~/.mindy/data/blobs`
   - Two-level directory structure for filesystem efficiency

3. **Vector Index**
   - TF-IDF with hash-based vectorization (8192-dim)
   - IVF (Inverted File) index for fast search
   - Cosine similarity search
   - Located at `~/.mindy/data/vector`

4. **Graph Store**
   - Extract entities and relationships
   - Store in BadgerDB (embedded key-value store)
   - Located at `~/.mindy/data/graph`
   - Node and edge types with properties

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
   - Vector index saved on close

3. **Better Search**
   - Normalized TF-IDF vectors
   - Cosine similarity ranking

## Technical Specification

### Vector Index

**TF-IDF Implementation**:
- **Dimension**: 8192 (fixed)
- **Mapping**: Hash-based (FNV32a)
- **TF formula**: `1 + log(TF_raw)`
- **IDF formula**: `log((N + 1) / (df + 1))`
- **Normalization**: L2 (Euclidean)
- **Search**: IVF with cosine similarity

**Storage**:
```
~/.mindy/data/tfidf/
├── vocab.json     # term → index mapping
├── idf.json       # term → IDF score
├── vectors.json   # doc_id → vector (sparse format)
└── meta.json      # document count
```

### Graph Store

**Schema**:
```
Nodes:
  - Document: file metadata, blob reference
  - Chunk: text chunk from document
  - Entity: extracted entity (email, URL, person, etc.)

Edges:
  - HAS_CHUNK: Document → Chunk
  - HAS_ENTITY: Chunk → Entity
```

**Storage**: BadgerDB (embedded)

### Entity Extraction

| Type | Pattern | Node ID Prefix |
|------|---------|----------------|
| Email | `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}` | `email:` |
| URL | `https?://...` | `url:` |
| Phone | `\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}` | `phone:` |
| Date | `\d{1,2}[/-]\d{1,2}[/-]\d{2,4}` | `date:` |
| Proper Noun | Capitalized word, len > 2 | (none) |

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
| GET | /ui | Web UI |
| POST | /api/v1/ingest?path=<filepath> | Index file or directory |
| POST | /api/v1/reindex | Reindex all tracked files |
| GET | /api/v1/search?q=<query>&k=<n> | Semantic search (default k=10) |
| GET | /api/v1/stats | Index statistics |
| GET | /api/v1/graph/search?q=<query> | Search nodes by label |
| GET | /api/v1/graph/node/{id} | Get node by ID |
| GET | /api/v1/graph/traverse?start=<id>&type=<edge>&depth=<n> | Graph traversal |
| GET | /api/v1/blob/{hash} | Get raw blob content |

### Search Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| q | string | required | Search query |
| k | int | 10 | Number of results |
| offset | int | 0 | Pagination offset |
| limit | int | k | Results per page |
| type | string | "" | Filter by file extension |
| path | string | "" | Filter by path prefix |

### Traverse Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| start | string | required | Starting node ID |
| type | string | "" | Edge type filter (empty = all) |
| depth | int | 3 | Traversal depth |

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

## Performance Characteristics

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Index document | O(T) | T = number of terms |
| Search | O(P × C) | P = probes, C = avg cluster size |
| Graph traversal | O(V + E) | BFS |
| Blob read | O(1) | Content-addressed |

## Out of Scope (Phase 1)

- Web crawler (Phase 2)
- Connectors (GitHub, Gmail) (Phase 2)
- Multi-node coordination (Phase 3)
- Multi-tenant / auth (Phase 3)
- Real-time sync (future)

## Technology

| Component | Implementation |
|-----------|---------------|
| Language | Go 1.21+ |
| Storage | Local filesystem + BadgerDB |
| Vector | Custom TF-IDF (hash-based) |
| HTTP | Chi router |
| No external dependencies | ✓ |

## Future Specifications (Planned)

### Phase 2 - Distributed / Crawler

### Web Crawler
- Index web pages automatically
- Respect robots.txt
- Configurable crawl depth

### Connectors
- GitHub (repos, issues, PRs)
- Gmail (emails)

### BM25 Ranking
Alternative ranking algorithm with better performance for keyword search:
- Parameters: k1 (term frequency saturation), b (document length normalization)
- Default: k1=1.5, b=0.75

### N-gram Indexing
Capture phrase information:
- Unigrams, bigrams, trigrams
- Configurable n-gram range
