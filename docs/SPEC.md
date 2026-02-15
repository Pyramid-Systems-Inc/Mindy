# Mindy - Specification

## Project Overview

- **Name:** Mindy
- **Type:** Personal AI memory / knowledge graph system
- **Goal:** Ingest local files, build vector + graph memory, query semantically
- **Target Users:** Single-user local-first, sellable later as multi-tenant

## Core Features

### Phase 1 (MVP)
1. **File Ingestion**
   - Watch directories for new/changed files
   - Support all file types: Markdown, PDF, TXT, HTML, etc.
   - Content-addressable storage (SHA256)

2. **Blob Store**
   - Store raw file content by content-hash
   - Immutable storage
   - Located at `~/.mindy/data/blobs`

3. **Vector Index**
   - Custom IVF index built in Go
   - Extract text from files, embed, store
   - Located at `~/.mindy/data/vector`

4. **Graph Store**
   - Extract entities and relationships
   - Store in BadgerDB
   - Located at `~/.mindy/data/graph`

5. **Query API**
   - Semantic search via vectors
   - Graph traversal
   - HTTP API on port 9090

## Configuration

| Flag | Config Key | Default | Description |
|------|------------|---------|-------------|
| `--watch` | `watch_paths` | none | Directories to watch (comma-separated) |
| `--port` | `http_port` | 9090 | API server port |
| `--data-dir` | `data_dir` | ~/.mindy/data | Data storage location |
| `--config` | - | none | Config file path |

## Data Flow

```
File → Blob Store (sha256) → Text Extract → Vector Index
                              ↘→ Graph Store (entities)
                                 
Query → Vector Search → Results
      → Graph Traversal → Results
```

## Out of Scope (Phase 1)
- Web crawler
- Connectors (GitHub, Gmail)
- Multi-node / CRDT frontier
- Multi-tenant / auth
- GUI (web UI later)
