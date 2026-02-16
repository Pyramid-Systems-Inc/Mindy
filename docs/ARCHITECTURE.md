# Mindy - Architecture

## System Overview

Mindy is a personal AI memory / knowledge graph system that:
- Ingests local files (watched or manually added)
- Builds semantic memory using TF-IDF/BM25 vectorization
- Stores entities and relationships in a graph
- Provides query API for semantic search and graph traversal
- Includes a built-in Web UI

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Mindy                                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  File        │───▶│    Blob      │    │   Web UI    │  │
│  │  Watcher    │    │   Store      │    │   (:9090)   │  │
│  │  (polling)  │    │  (SHA256)    │    │              │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│         │                     │                   │             │
│         ▼                     │                   │             │
│  ┌──────────────┐          │                   │             │
│  │   Indexer    │          │                   │             │
│  │              │          │                   │             │
│  └──────────────┘          │                   │             │
│         │                  │                   │             │
│         ├──────────────────┼──────────────────┘             │
│         ▼                  ▼                                │
│  ┌──────────────┐    ┌──────────────┐                   │
│  │    TF-IDF    │    │    Graph     │                   │
│  │   Embedder    │    │   Store      │                   │
│  │ (8192-dim)   │    │ (BadgerDB)   │                   │
│  └──────────────┘    └──────────────┘                   │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                   HTTP API (:9090)                      │  │
│  │  /health  /api/v1/search  /api/v1/ingest           │  │
│  │  /api/v1/stats  /api/v1/graph/*  /api/v1/reindex │  │
│  └─────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## Design Principles

### 1. Local-First
- All data stays on your machine
- No external dependencies (no cloud APIs, no LLMs)
- Single binary deployment

### 2. Content-Addressable
- Every piece of content is stored by its hash (SHA256)
- Deduplication is automatic
- Content integrity is guaranteed

### 3. Incremental Indexing
- File tracker monitors content hashes
- Skips re-indexing unchanged files
- Tracks modified times for change detection

### 4. Privacy-First
- No data leaves your machine
- No authentication required (local use)
- Content-addressed access only

## Components

### 1. Ingestion Layer

**File Watcher**
- Monitors directories for new/changed files
- Polls every 5 seconds
- Filters by file extension
- Non-blocking: files are queued for async processing

**Manual Ingest**
- REST API: `POST /api/v1/ingest?path=<filepath>`
- Supports single files or entire directories

**Supported File Types**
| Extension | Handler | Description |
|-----------|---------|-------------|
| `.txt` | Plain text | Direct pass-through |
| `.md`, `.markdown` | Markdown | Direct pass-through |
| `.html`, `.htm` | HTML stripper | Removes tags, keeps text |
| `.json` | JSON | Direct pass-through |
| `.xml` | XML | Direct pass-through |
| `.csv` | CSV | Field extraction |
| `.log` | Log | Direct pass-through |
| `.pdf` | PDF extractor | Text extraction |
| `.docx` | DOCX extractor | XML parsing |

### 2. Storage Layer

#### Blob Store (`~/.mindy/data/blobs/`)
Content-addressable storage using SHA256:
```
~/.mindy/data/blobs/
├── ab/
│   └── cdef123456789...
├── cd/
│   └── efgh789012345...
└── ...
```
- **Key**: SHA256(content)
- **Directory structure**: First 2 chars of hash / remaining chars
- **Benefits**: Automatic deduplication, integrity verification

#### Vector Index (`~/.mindy/data/vector/`)
Custom IVF index with TF-IDF/BM25:
```
~/.mindy/data/vector/
├── centroids.bin    # IVF cluster centroids

~/.mindy/data/tfidf/
├── vocab.json      # Term → index mapping
├── idf.json       # Inverse document frequencies
├── vectors.json   # Document vectors
├── meta.json      # Document count, stats
└── file_tracker.json  # File hash tracking
```
- **Dimension**: 8192 (hash-based mapping)
- **Index type**: IVF (Inverted File with k-means)
- **Similarity**: Cosine similarity
- **Algorithms**: TF-IDF (default), BM25 (optional)

#### Graph Store (`~/.mindy/data/graph/`)
BadgerDB-based graph storage:
- **Nodes**: Documents, Chunks, Entities
- **Edges**: Relationships between nodes
- **Search**: Full-text search on labels and properties

### 3. Computation Layer

#### Indexer Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│                    Indexer Pipeline                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  File Input                                                 │
│      │                                                      │
│      ▼                                                      │
│  ┌──────────────┐                                          │
│  │ Blob Store   │──▶ SHA256 → Content-addressable         │
│  └──────────────┘       storage                             │
│      │                                                      │
│      ▼                                                      │
│  ┌──────────────┐                                          │
│  │ Text Extract │──▶ File-type specific extraction         │
│  │   (10+ types)│     (PDF, DOCX, HTML, etc.)            │
│  └──────────────┘                                          │
│      │                                                      │
│      ├──────────────────────┐                               │
│      ▼                      ▼                               │
│  ┌──────────────┐     ┌──────────────┐                  │
│  │ TF-IDF/BM25  │     │ Entity Extract│                  │
│  │   Index      │     │              │                  │
│  │              │     │ - emails     │                  │
│  │ - tokenize   │     │ - URLs       │                  │
│  │ - stopwords  │     │ - phones     │                  │
│  │ - stemming    │     │ - dates      │                  │
│  │ - IDF        │     │ - proper nouns│                  │
│  └──────────────┘     └──────────────┘                  │
│                             │                               │
│                             ▼                               │
│                       ┌──────────────┐                    │
│                       │ Graph Store  │                    │
│                       │              │                    │
│                       │ - Add nodes  │                    │
│                       │ - Add edges  │                    │
│                       └──────────────┘                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### TF-IDF Implementation Details

**Tokenization**:
1. Lowercase conversion
2. Alphanumeric extraction
3. Stopword filtering (60+ common English words)
4. Simple stemming (15 common suffixes)
5. Minimum length filter (2 chars)

**Term Frequency (TF)**:
```
TF(t,d) = 1 + log(TF_raw(t,d))
```

**Inverse Document Frequency (IDF)**:
```
IDF(t) = log((N + 1) / (df(t) + 1)) + 1
```

**Vectorization**:
- Hash-based mapping: `position = FNV32a(term) % 8192`
- Each term maps to a fixed position in 8192-dim space
- Vector = TF × IDF for each term position
- L2 normalization for cosine similarity

**BM25 (Optional)**:
- Parameters: k1=1.5, b=0.75
- Document length normalization

#### Entity Extraction

| Type | Pattern | Node ID Prefix |
|------|---------|----------------|
| Email | `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}` | `email:` |
| URL | `https?://...` | `url:` |
| Phone | `\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}` | `phone:` |
| Date | `\d{1,2}[/-]\d{1,2}[/-]\d{2,4}` | `date:` |
| Proper Noun | Capitalized word, len > 2 | (none) |

### 4. Access Layer

#### HTTP Server (port 9090)

| Method | Path | Description |
|--------|------|-------------|
| GET | / | Web UI |
| GET | /ui | Web UI |
| GET | /health | Health check with timestamp |
| POST | /api/v1/ingest | Index file/directory |
| POST | /api/v1/reindex | Reindex all files |
| GET | /api/v1/search | Semantic search with filters |
| GET | /api/v1/stats | Index statistics |
| GET | /api/v1/graph/node/{id} | Get node by ID |
| GET | /api/v1/graph/traverse | Graph traversal |
| GET | /api/v1/graph/search | Search nodes |
| GET | /api/v1/blob/{hash} | Get raw content |

#### Web UI
- Built-in HTML/CSS/JS interface
- Search bar with results display
- Tab navigation (Results, API docs)
- Graph exploration

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
    "content_type": "text/markdown",
    "file_type": "markdown"
  }
}
```

### Chunk Node
```json
{
  "id": "chunk:<doc-hash>:0",
  "type": "Chunk",
  "label": "Chunk 0",
  "blob_ref": "<sha256>",
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
  "id": "entity:john_doe",
  "type": "Entity",
  "label": "John Doe",
  "props": {
    "name": "John Doe",
    "kind": "capitalized"
  }
}
```

### Edges
| Type | From | To | Description |
|------|------|----|-------------|
| HAS_CHUNK | Document | Chunk | Document contains chunk |
| HAS_ENTITY | Chunk | Entity | Chunk mentions entity |

## Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go 1.21+ | Fast, good concurrency, single binary |
| Blob Store | Local filesystem | Simple, reliable, content-addressable |
| Vector Index | Custom IVF + TF-IDF/BM25 | No external deps, efficient |
| Graph Store | BadgerDB | Fast, embedded, Go-native |
| HTTP Server | Chi router | Minimal, fast |
| Web UI | Embedded HTML | No separate frontend needed |
| No external APIs | ✓ | Privacy-first |

## Configuration

### CLI Flags
```bash
mindy [options]

Options:
  --config <path>    Config file path (YAML)
  --watch <paths>    Comma-separated directories to watch
  --port <n>         HTTP server port (default: 9090)
  --data-dir <path>  Data directory (default: ~/.mindy/data)
  --help             Show help message
  --version           Show version information
```

### Config File (YAML)
```yaml
watch_paths:
  - /path/to/docs
  - /path/to/notes
http_port: 9090
data_dir: ~/.mindy/data
```

## Performance Characteristics

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Index document | O(T) | T = number of terms |
| Search | O(P × C) | P = probes, C = avg cluster size |
| Graph traversal | O(V + E) | BFS |
| Blob read | O(1) | Content-addressed |
| Incremental index | O(1) | If hash unchanged |

## Version History

### v1.0.0 (Current)
- Initial release
- TF-IDF/BM25 vector search
- Graph-based entity storage
- Web UI
- Incremental indexing
- PDF/DOCX support
