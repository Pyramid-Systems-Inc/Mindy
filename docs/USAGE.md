# Mindy - Usage Guide

## Quick Start

```bash
# Start mindy with a watched directory
mindy.exe --watch "C:\Users\You\Documents" --port 9090

# Or use a config file
mindy.exe --config mindy.yaml
```

## Installation

### From Binary

1. Download `mindy.exe` from releases
2. Run `mindy.exe --help` to verify

### From Source

```bash
go build -o mindy.exe ./cmd/mindy
```

## Configuration

### Command Line

```bash
mindy [options]

Options:
  --config <path>    Config file path (YAML)
  --watch <paths>    Directories to watch (comma-separated)
  --port <n>         HTTP server port (default: 9090)
  --data-dir <path>  Data directory (default: ~/.mindy/data)
  --help             Show help
```

### Config File

Create `mindy.yaml`:

```yaml
watch_paths:
  - C:\Users\You\Documents
  - C:\Users\You\Notes
http_port: 9090
data_dir: C:\Users\You\.mindy\data
```

Then run:

```bash
mindy.exe --config mindy.yaml
```

## API Usage

### Health Check

```bash
curl http://localhost:9090/health
```

Response:
```json
{"status": "ok"}
```

### Ingest a File

```bash
# Single file
curl -X POST "http://localhost:9090/api/v1/ingest?path=C:\Users\You\Documents\notes.md"
```

Response:
```json
{"status": "ok", "path": "C:\\Users\\You\\Documents\\notes.md"}
```

### Ingest a Directory

```bash
curl -X POST "http://localhost:9090/api/v1/ingest?path=C:\Users\You\Documents"
```

Response:
```json
{"status": "ok", "message": "Directory queued for indexing", "files": 42}
```

### Semantic Search

```bash
curl "http://localhost:9090/api/v1/search?q=python+programming&k=5"
```

Response:
```json
{
  "query": "python programming",
  "results": [
    {
      "id": "chunk:abc123:0",
      "score": 0.85,
      "meta": "{\"doc_id\":\"doc:abc123\",\"chunk\":0,\"path\":\"C:\\Users\\You\\Docs\\python.md\"}"
    }
  ]
}
```

### Get Node by ID

```bash
curl "http://localhost:9090/api/v1/graph/node/doc:abc123"
```

Response:
```json
{
  "id": "doc:abc123",
  "type": "Document",
  "label": "python.md",
  "blob_ref": "abc123...",
  "props": {
    "path": "C:\\Users\\You\\Docs\\python.md",
    "size": 1234,
    "modified": 1700000000,
    "content_type": "text/markdown"
  }
}
```

### Graph Traversal

```bash
curl "http://localhost:9090/api/v1/graph/traverse?start=doc:abc123&type=HAS_CHUNK&depth=2"
```

Response:
```json
{
  "start": "doc:abc123",
  "nodes": [
    {"id": "doc:abc123", "type": "Document", ...},
    {"id": "chunk:abc123:0", "type": "Chunk", ...},
    {"id": "entity:python", "type": "Entity", ...}
  ]
}
```

### Get Raw Content

```bash
curl "http://localhost:9090/api/v1/blob/abc123..."
```

Returns the raw file content.

## Supported File Types

| Extension | Type | Extraction |
|-----------|------|------------|
| `.txt` | Plain text | Direct |
| `.md` | Markdown | Direct |
| `.markdown` | Markdown | Direct |
| `.html` | HTML | Strip tags |
| `.htm` | HTML | Strip tags |
| `.json` | JSON | Direct |
| `.xml` | XML | Direct |
| `.csv` | CSV | Direct |
| `.log` | Log | Direct |

## Data Storage

Mindy stores all data in the data directory (default: `~/.mindy/data`):

```
~/.mindy/data/
├── blobs/           # Raw file content (SHA256 addresses)
│   ├── ab/
│   │   └── cdef1234...
│   └── cd/
│       └── efgh5678...
├── graph/          # BadgerDB graph store
├── vector/         # TF-IDF vectors and centroids
└── tfidf/          # TF-IDF vocabulary and IDF scores
    ├── vocab.json
    ├── idf.json
    ├── vectors.json
    └── meta.json
```

## Programming Examples

### Python

```python
import requests

BASE = "http://localhost:9090"

# Search
def search(query, k=10):
    r = requests.get(f"{BASE}/api/v1/search", params={"q": query, "k": k})
    return r.json()

# Ingest
def ingest(path):
    r = requests.post(f"{BASE}/api/v1/ingest", params={"path": path})
    return r.json()

# Get document
def get_node(node_id):
    r = requests.get(f"{BASE}/api/v1/graph/node/{node_id}")
    return r.json()

# Traverse graph
def traverse(start, edge_type="", depth=3):
    r = requests.get(f"{BASE}/api/v1/graph/traverse", 
                    params={"start": start, "type": edge_type, "depth": depth})
    return r.json()
```

### JavaScript/Node.js

```javascript
const BASE = "http://localhost:9090";

async function search(query, k = 10) {
  const r = await fetch(`${BASE}/api/v1/search?q=${encodeURIComponent(query)}&k=${k}`);
  return r.json();
}

async function ingest(path) {
  const r = await fetch(`${BASE}/api/v1/ingest?path=${encodeURIComponent(path)}`, 
    { method: "POST" });
  return r.json();
}
```

### cURL

```bash
# Health check
curl http://localhost:9090/health

# Search
curl "http://localhost:9090/api/v1/search?q=machine+learning"

# Index a file
curl -X POST "http://localhost:9090/api/v1/ingest?path=C:\docs\readme.md"

# Get node
curl "http://localhost:9090/api/v1/graph/node/doc:abc123"

# Traverse from document
curl "http://localhost:9090/api/v1/graph/traverse?start=doc:abc123&depth=2"
```

## Troubleshooting

### Port Already in Use

```bash
# Use a different port
mindy.exe --port 9091
```

### Data Directory Issues

```bash
# Use a custom data directory
mindy.exe --data-dir "C:\mindy-data"
```

### Check Logs

Mindy logs to stdout. On Windows, you can capture logs:

```cmd
mindy.exe --watch "C:\docs" > mindy.log 2>&1
```

### View Stored Data

```bash
# List blobs
dir "%USERPROFILE%\.mindy\data\blobs"

# View graph (requires tools)
# (Future: add graph inspection CLI)
```

## Performance Tips

1. **Watch fewer directories** - More directories = more polling overhead
2. **Use exclusions** - Don't watch temp folders or caches
3. **Batch large directories** - Use the ingest API for bulk operations

## Security Notes

- Mindy runs locally; data stays on your machine
- No authentication by default (local use only)
- Blob access is content-addressed (read-only)
