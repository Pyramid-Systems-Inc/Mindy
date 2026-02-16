# Mindy - Architecture

## System Overview

Mindy is a personal AI memory / knowledge graph system that:
- Ingests local files (watched or manually added)
- Builds semantic memory using TF-IDF vectorization
- Stores entities and relationships in a graph
- Provides query API for semantic search and graph traversal

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Mindy                                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  File        │───▶│    Blob      │    │   HTTP API   │  │
│  │  Watcher    │    │   Store      │    │   (:9090)    │  │
│  │  (polling)  │    │  (SHA256)    │    │              │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│         │                     │                   │          │
│         ▼                     │                   ▼          │
│  ┌──────────────┐            │           ┌──────────────┐   │
│  │   Indexer    │            │           │   Search    │   │
│  │              │            │           │   Query      │   │
│  └──────────────┘            │           └──────────────┘   │
│         │                    │                   │           │
│         ▼                    ▼                   ▼           │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐   │
│  │    TF-IDF    │    │    Graph     │    │   Query      │   │
│  │   Embedder   │    │   Store      │    │   Engine     │   │
│  │ (4096-dim)   │    │ (BadgerDB)   │    │              │   │
│  └──────────────┘    └──────────────┘    └──────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. Ingestion Layer

**File Watcher**
- Monitors directories for new/changed files
- Polls every 5 seconds
- Filters by file extension

**Manual Ingest**
- REST API: `POST /api/v1/ingest?path=<filepath>`

### 2. Storage Layer

**Blob Store** (`~/.mindy/data/blobs/`)
- Content-addressable storage
- Key: SHA256(content)
- Two-level directory: `xx/yyyy...` (first 2 chars / rest)

**Vector Index** (`~/.mindy/data/vector/`)
- Custom IVF (Inverted File) index
- 4096-dimensional TF-IDF vectors
- Hash-based term mapping for efficiency

**Graph Store** (`~/.mindy/data/graph/`)
- BadgerDB (embedded key-value store)
- Nodes: Document, Chunk, Entity
- Edges: HAS_CHUNK, HAS_ENTITY, LINKS_TO, SAME_AS

### 3. Computation Layer

**Indexer Pipeline**
```
File → Blob Store → Text Extract → TF-IDF Vector
                              ↘→ Entity Extract → Graph Store
```

**Text Extraction**
- Plain text (.txt, .md)
- HTML stripping
- JSON passthrough

**Entity Extraction**
- Regex-based: emails, URLs, phones, dates
- Capitalized word detection for proper nouns

### 4. Access Layer

**HTTP Server** (port 9090)

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |
| POST | /api/v1/ingest | Trigger indexing |
| GET | /api/v1/search | Semantic search |
| GET | /api/v1/graph/node/{id} | Get node by ID |
| GET | /api/v1/graph/traverse | Graph traversal |
| GET | /api/v1/blob/{hash} | Get raw content |

## Data Models

### Blob
```
Key:   sha256(content)
Value: raw file content
Path:  ~/.mindy/data/blobs/xx/yyyy...
```

### Document Node
```json
{
  "id": "doc:<hash>",
  "type": "Document",
  "label": "filename.md",
  "blob_ref": "<sha256>",
  "props": {
    "path": "/full/path/to/file.md",
    "size": 1234,
    "modified": 1700000000,
    "content_type": "text/markdown"
  }
}
```

### Chunk Node
```json
{
  "id": "chunk:<doc-hash>:0",
  "type": "Chunk",
  "props": {
    "text": "extracted text...",
    "index": 0,
    "doc_id": "doc:<hash>"
  }
}
```

### Entity Node
```json
{
  "id": "entity:some_entity_name",
  "type": "Entity",
  "label": "Some Entity Name",
  "props": {
    "name": "Some Entity Name",
    "kind": "email|url|phone|date|capitalized"
  }
}
```

### Edges
- `HAS_CHUNK`: Document → Chunk
- `HAS_ENTITY`: Chunk → Entity
- `LINKS_TO`: Entity → Entity (future)
- `SAME_AS`: Entity → Entity (future)

## Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go | Fast, good concurrency, single binary |
| Blob Store | Local filesystem | Simple, reliable |
| Vector Index | Custom IVF + TF-IDF | No external deps |
| Graph Store | BadgerDB | Fast, embedded, Go-native |
| HTTP Server | Chi router | Minimal, fast |

## Configuration

### CLI Flags
```bash
mindy [options]

Options:
  --config <path>    Config file path (YAML)
  --watch <paths>    Comma-separated directories to watch
  --port <n>         HTTP server port (default: 9090)
  --data-dir <path>  Data directory (default: ~/.mindy/data)
```

### Config File (YAML)
```yaml
watch_paths:
  - /path/to/docs
  - /path/to/notes
http_port: 9090
data_dir: ~/.mindy/data
```

## Future Enhancements

### Phase 2 (Planned)
- PDF/DOCX parsing
- Better entity extraction (NER)
- BM25 ranking
- Web UI
- CLI improvements

### Phase 3 (Future)
- Web crawler
- Connectors (GitHub, etc.)
- Multi-node coordination
- MCP integration
