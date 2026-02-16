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
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │    TF-IDF    │    │    Graph     │    │   Query      │  │
│  │   Embedder   │    │   Store      │    │   Engine     │  │
│  │ (4096-dim)   │    │ (BadgerDB)   │    │              │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
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

### 3. Immutable Storage
- Blobs are never modified
- Updates create new entries with new hashes
- History is preserved through graph relationships

## Components

### 1. Ingestion Layer

**File Watcher**
- Monitors directories for new/changed files
- Polls every 5 seconds
- Filters by file extension (.txt, .md, .html, .json, .xml, .csv, .log)
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
| `.csv` | CSV | Direct pass-through |
| `.log` | Log | Direct pass-through |

### 2. Storage Layer

#### Blob Store (`~/.mindy/data/blobs/`)
Content-addressable storage using SHA256:

```
Structure:
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
Custom IVF (Inverted File) index with TF-IDF:

```
~/.mindy/data/vector/
├── centroids.bin    # IVF cluster centroids
└── ...

~/.mindy/data/tfidf/
├── vocab.json      # Term → index mapping
├── idf.json        # Inverse document frequencies
├── vectors.json    # Document vectors
└── meta.json       # Document count, stats
```

- **Dimension**: 4096 (hash-based mapping)
- **Index type**: IVF (Inverted File with k-means)
- **Similarity**: Cosine similarity

#### Graph Store (`~/.mindy/data/graph/`)
BadgerDB-based graph storage:

- **Nodes**: Documents, Chunks, Entities
- **Edges**: Relationships between nodes
- **Indexes**: Outgoing edges per node, incoming edges per node

### 3. Computation Layer

#### Indexer Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│                    Indexer Pipeline                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  File Input                                                 │
│      │                                                      │
│      ▼                                                      │
│  ┌──────────────┐                                          │
│  │ Blob Store   │──▶ SHA256 → Content-addressable           │
│  └──────────────┘       storage                             │
│      │                                                      │
│      ▼                                                      │
│  ┌──────────────┐                                          │
│  │ Text Extract │──▶ File-type specific extraction          │
│  └──────────────┘       (strip HTML, parse JSON, etc.)      │
│      │                                                      │
│      ├──────────────────────┐                               │
│      ▼                      ▼                               │
│  ┌──────────────┐     ┌──────────────┐                     │
│  │ TF-IDF Index │     │ Entity Extract│                    │
│  │              │     │              │                     │
│  │ 1. Tokenize  │     │ 1. Regex     │                     │
│  │ 2. TF calc   │     │    - emails  │                     │
│  │ 3. IDF calc  │     │    - URLs    │                     │
│  │ 4. Vector    │     │    - phones  │                     │
│  │    build     │     │    - dates    │                     │
│  │ 5. Cluster   │     │ 2. Capitalized│                    │
│  └──────────────┘     │    words      │                     │
│                       └──────────────┘                     │
│                             │                               │
│                             ▼                               │
│                       ┌──────────────┐                      │
│                       │ Graph Store  │                      │
│                       │              │                      │
│                       │ - Add nodes  │                      │
│                       │ - Add edges  │                      │
│                       └──────────────┘                      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### TF-IDF Implementation Details

**Tokenization**:
1. Lowercase conversion
2. Alphanumeric extraction
3. Word boundary detection
4. Minimum length filter (2 chars)

**Term Frequency (TF)**:
```
TF(t,d) = 1 + log(TF_raw(t,d))
```
- Logarithmic scaling to reduce impact of frequent terms

**Inverse Document Frequency (IDF)**:
```
IDF(t) = log((N + 1) / (df(t) + 1))
```
- Smooth IDF to avoid division by zero
- N = total documents, df = documents containing term

**Vectorization**:
- Hash-based mapping: `position = FNV32a(term) % 4096`
- Each term maps to a fixed position in 4096-dim space
- Vector = TF × IDF for each term position
- L2 normalization for cosine similarity

**Search**:
1. Tokenize query
2. Build query vector (same process as indexing)
3. Search IVF clusters (top N probes)
4. Compute cosine similarity with candidates
5. Return top-K results

#### Entity Extraction

| Type | Pattern | Node Prefix |
|------|---------|-------------|
| Email | `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}` | `email:` |
| URL | `https?://[^\s<>"{}|\\^\`]+` | `url:` |
| Phone | `(\+?1?[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}` | `phone:` |
| Date | `\d{1,2}[/-]\d{1,2}[/-]\d{2,4}` or month names | `date:` |
| Proper Noun | Capitalized words (len > 2) | (none) |

### 4. Access Layer

#### HTTP Server (port 9090)

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
| LINKS_TO | Entity | Entity | Entity relationship (future) |
| SAME_AS | Entity | Entity | Entity equivalence (future) |

## Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go 1.21+ | Fast, good concurrency, single binary |
| Blob Store | Local filesystem | Simple, reliable, content-addressable |
| Vector Index | Custom IVF + TF-IDF | No external deps, efficient |
| Graph Store | BadgerDB | Fast, embedded, Go-native |
| HTTP Server | Chi router | Minimal, fast |
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

### Time Complexity
| Operation | Complexity |
|-----------|------------|
| Index document | O(T × D) where T = terms, D = vocab size |
| Search | O(P × C) where P = probes, C = avg cluster size |
| Graph traversal | O(V + E) where V = nodes, E = edges |

### Space Complexity
| Component | Complexity |
|-----------|------------|
| Blob store | O(total content size) |
| Vector index | O(D × K) where D = docs, K = 4096 |
| Graph store | O(V + E) |

## Future Enhancements

### Phase 1.2 (Near)
- PDF/DOCX parsing
- BM25 ranking (alternative to TF-IDF)
- Better CLI with interactive mode

### Phase 2 (Medium-term)
- Web crawler
- Connectors (GitHub, Gmail, Slack)
- Multi-node coordination

### Phase 3 (Long-term)
- Distributed CRDT frontier
- MCP integration
- Agent framework
