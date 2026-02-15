# Mindy - Architecture

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Mindy                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Ingestion  │───▶│    Blob      │    │   HTTP API   │  │
│  │   (Watcher)  │    │   Store      │    │   (:9090)    │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│         │                                        │          │
│         ▼                                        ▼          │
│  ┌──────────────┐                       ┌──────────────┐   │
│  │   Indexer    │                       │   Query      │   │
│  │              │                       │   Engine     │   │
│  └──────────────┘                       └──────────────┘   │
│         │                                        │          │
│         ▼                                        ▼          │
│  ┌──────────────┐                       ┌──────────────┐   │
│  │   Vector     │◀─────────────────────▶│   Graph      │   │
│  │   Index      │                       │   Store      │   │
│  └──────────────┘                       └──────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. Ingestion Layer
- **File Watcher:** Monitors directories for file changes
- **File Processor:** Reads files, extracts metadata
- **Content Extractor:** Extracts text from various file types

### 2. Storage Layer
- **Blob Store:** Immutable content-addressable storage
- **Vector Index:** IVF-based ANN index for semantic search
- **Graph Store:** Entity/relationship storage using BadgerDB

### 3. Computation Layer
- **Indexer:** Coordinates extraction, embedding, indexing
- **Embedder:** Text to vector conversion (uses external API or local model)

### 4. Access Layer
- **HTTP Server:** REST API for queries
- **Query Engine:** Routes queries to vector/graph stores

## Data Models

### Blob
```
Key:   sha256(content)
Value: raw file content
```

### Vector Index
```
- IVF index with k-means clustering
- Vectors stored in partitions
- Metadata: document_id, chunk_id, source_path
```

### Graph
```
Nodes:
  - Document: file metadata, blob ref
  - Entity: extracted entity (person, org, etc.)
  - Chunk: text chunk from document

Edges:
  - HAS_CHUNK: Document → Chunk
  - HAS_ENTITY: Chunk → Entity
  - LINKS_TO: Entity → Entity
  - SAME_AS: Entity → Entity
```

## API Endpoints (Phase 1)

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/ingest | Trigger manual ingestion |
| GET | /api/v1/search | Semantic search |
| GET | /api/v1/graph/node/{id} | Get node by ID |
| GET | /api/v1/graph/traverse | Graph traversal |
| GET | /health | Health check |

## Configuration

- CLI flags for all options
- Optional YAML config file
- Environment variables support

## Technology Choices

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go | Fast dev, good concurrency |
| Blob | Local filesystem | Simple, reliable |
| Vector | Custom IVF | No external deps, local |
| Graph | BadgerDB | Fast, embedded, Go-native |
| HTTP | Chi router | Minimal, fast |
