# Mindy - Usage Guide

## Quick Start

```bash
# Start mindy with a watched directory
mindy.exe --watch "C:\Users\You\Documents" --port 9090

# Or use a config file
mindy.exe --config mindy.yaml

# Check if it's running
curl http://localhost:9090/health
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
# Basic search (returns top 10 by default)
curl "http://localhost:9090/api/v1/search?q=python+programming"

# Search with custom limit
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
    },
    {
      "id": "chunk:def456:1",
      "score": 0.72,
      "meta": "{\"doc_id\":\"doc:def456\",\"chunk\":1,\"path\":\"C:\\Users\\You\\Docs\\tutorial.md\"}"
    }
  ]
}
```

### Get Node by ID

```bash
# Get document node
curl "http://localhost:9090/api/v1/graph/node/doc:abc123"
```

Response:
```json
{
  "id": "doc:abc123",
  "type": "Document",
  "label": "python.md",
  "blob_ref": "abc123def456",
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
# Get all chunks from a document
curl "http://localhost:9090/api/v1/graph/traverse?start=doc:abc123&type=HAS_CHUNK&depth=1"

# Get all entities from a document (depth 2)
curl "http://localhost:9090/api/v1/graph/traverse?start=doc:abc123&type=HAS_ENTITY&depth=2"

# Full graph exploration
curl "http://localhost:9090/api/v1/graph/traverse?start=doc:abc123&depth=3"
```

Response:
```json
{
  "start": "doc:abc123",
  "nodes": [
    {"id": "doc:abc123", "type": "Document", "label": "python.md"},
    {"id": "chunk:abc123:0", "type": "Chunk", "label": "Chunk 0"},
    {"id": "chunk:abc123:1", "type": "Chunk", "label": "Chunk 1"},
    {"id": "entity:python", "type": "Entity", "label": "Python"},
    {"id": "entity:programming", "type": "Entity", "label": "Programming"}
  ]
}
```

### Get Raw Content

```bash
# Get content by blob hash
curl "http://localhost:9090/api/v1/blob/abc123def456"
```

Returns the raw file content (text/plain).

## Understanding Search Results

### Interpreting Scores

The search returns cosine similarity scores (0.0 to 1.0):
- **0.85+**: Very relevant
- **0.70-0.85**: Relevant
- **0.50-0.70**: Somewhat relevant
- **< 0.50**: May not be relevant

### Result Metadata

Each result includes metadata about the source chunk:
```json
{
  "doc_id": "doc:abc123",   // Parent document ID
  "chunk": 0,               // Chunk index within document
  "path": "C:\\docs\\file.md"  // Original file path
}
```

### Building a Complete Answer

To get full context from search results:

1. **Search** for relevant chunks
2. **Get document** from `doc_id` in metadata
3. **Get blob** using `blob_ref` from document node

```bash
# 1. Search
curl "http://localhost:9090/api/v1/search?q=machine+learning"

# 2. Get document (use doc_id from results)
curl "http://localhost:9090/api/v1/graph/node/doc:abc123"

# 3. Get full content (use blob_ref from document)
curl "http://localhost:9090/api/v1/blob/abc123def456"
```

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

## Entity Types Extracted

Mindy automatically extracts these entity types:

| Type | Example | Search by ID |
|------|---------|---------------|
| Emails | `john@example.com` | `entity:email:john@example.com` |
| URLs | `https://example.com` | `entity:url:https://example.com` |
| Phones | `555-123-4567` | `entity:phone:555-123-4567` |
| Dates | `2024-01-15` | `entity:date:2024-01-15` |
| Proper Nouns | `Python`, `John` | `entity:python` |

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
│   ├── 000000.vlog
│   └── 000000.sst
├── vector/         # IVF vector index
│   └── centroids.bin
└── tfidf/          # TF-IDF index
    ├── vocab.json      # Term vocabulary
    ├── idf.json        # Inverse document frequencies
    ├── vectors.json    # Document vectors
    └── meta.json      # Index statistics
```

## Common Use Cases

### Use Case 1: Personal Knowledge Base

```yaml
# mindy.yaml
watch_paths:
  - C:\Users\You\Documents
  - C:\Users\You\Notes
  - C:\Users\You\Research
http_port: 9090
```

```bash
# Start
mindy.exe --config mindy.yaml

# Search your knowledge base
curl "http://localhost:9090/api/v1/search?q=project+management+notes"
```

### Use Case 2: Code Documentation Search

```bash
# Index code docs
curl -X POST "http://localhost:9090/api/v1/ingest?path=C:\Projects\docs"

# Search
curl "http://localhost:9090/api/v1/search?q=authentication+setup"
```

### Use Case 3: Research Paper Analyzer

```bash
# Index papers
curl -X POST "http://localhost:9090/api/v1/ingest?path=C:\Research\papers"

# Find related entities
curl "http://localhost:9090/api/v1/graph/traverse?start=entity:machine_learning&depth=2"
```

## Programming Examples

### Python

```python
import requests

BASE = "http://localhost:9090"

class MindyClient:
    def __init__(self, base_url=BASE):
        self.base = base_url
    
    def health(self):
        return requests.get(f"{self.base}/health").json()
    
    def search(self, query, k=10):
        return requests.get(
            f"{self.base}/api/v1/search",
            params={"q": query, "k": k}
        ).json()
    
    def ingest(self, path):
        return requests.post(
            f"{self.base}/api/v1/ingest",
            params={"path": path}
        ).json()
    
    def get_node(self, node_id):
        return requests.get(
            f"{self.base}/api/v1/graph/node/{node_id}"
        ).json()
    
    def traverse(self, start, edge_type="", depth=3):
        return requests.get(
            f"{self.base}/api/v1/graph/traverse",
            params={"start": start, "type": edge_type, "depth": depth}
        ).json()
    
    def get_blob(self, hash):
        return requests.get(f"{self.base}/api/v1/blob/{hash}").text

# Usage
client = MindyClient()
results = client.search("python tutorial")
for r in results["results"]:
    print(f"{r['score']:.2f} - {r['meta']}")

doc = client.get_node("doc:abc123")
print(f"Document: {doc['label']}")
```

### JavaScript/Node.js

```javascript
const BASE = "http://localhost:9090";

class MindyClient {
  constructor(baseUrl = BASE) {
    this.base = baseUrl;
  }

  async health() {
    const r = await fetch(`${this.base}/health`);
    return r.json();
  }

  async search(query, k = 10) {
    const r = await fetch(
      `${this.base}/api/v1/search?q=${encodeURIComponent(query)}&k=${k}`
    );
    return r.json();
  }

  async ingest(path) {
    const r = await fetch(
      `${this.base}/api/v1/ingest?path=${encodeURIComponent(path)}`,
      { method: "POST" }
    );
    return r.json();
  }

  async getNode(nodeId) {
    const r = await fetch(`${this.base}/api/v1/graph/node/${nodeId}`);
    return r.json();
  }

  async traverse(start, edgeType = "", depth = 3) {
    const r = await fetch(
      `${this.base}/api/v1/graph/traverse?start=${start}&type=${edgeType}&depth=${depth}`
    );
    return r.json();
  }
}

// Usage
const client = new MindyClient();
const results = await client.search("python tutorial");
console.log(results.results);
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
)

type MindyClient struct {
    Base string
}

func NewMindyClient(port int) *MindyClient {
    return &MindyClient{Base: fmt.Sprintf("http://localhost:%d", port)}
}

func (c *MindyClient) Search(query string, k int) (map[string]interface{}, error) {
    resp, err := http.Get(c.Base + "/api/v1/search?" + 
        url.Values{"q": []string{query}, "k": []string{fmt.Sprint(k)}}.Encode())
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}

func main() {
    client := NewMindyClient(9090)
    result, _ := client.Search("python", 10)
    fmt.Printf("%+v\n", result)
}
```

### cURL Commands

```bash
# Health check
curl http://localhost:9090/health

# Search
curl "http://localhost:9090/api/v1/search?q=machine+learning"
curl "http://localhost:9090/api/v1/search?q=python&k=5"

# Index a file
curl -X POST "http://localhost:9090/api/v1/ingest?path=C:\docs\readme.md"

# Index a directory
curl -X POST "http://localhost:9090/api/v1/ingest?path=C:\docs"

# Get node
curl "http://localhost:9090/api/v1/graph/node/doc:abc123"

# Traverse from document
curl "http://localhost:9090/api/v1/graph/traverse?start=doc:abc123&depth=2"

# Get entity connections
curl "http://localhost:9090/api/v1/graph/traverse?start=entity:python&depth=1"

# Get raw content
curl "http://localhost:9090/api/v1/blob/abc123def456"
```

## Troubleshooting

### Port Already in Use

Find and kill the process:
```cmd
netstat -ano | findstr :9090
taskkill /PID <PID> /F

# Or use a different port
mindy.exe --port 9091
```

### Data Directory Issues

```bash
# Check directory permissions
dir "%USERPROFILE%\.mindy"

# Use a custom data directory
mindy.exe --data-dir "C:\mindy-data"
```

### No Search Results

1. **Verify documents are indexed**:
```bash
curl "http://localhost:9090/api/v1/graph/traverse?start=doc:*&depth=1"
```

2. **Check for entities**:
```bash
curl "http://localhost:9090/api/v1/graph/traverse?start=entity:*&depth=1"
```

3. **Re-index**:
```bash
curl -X POST "http://localhost:9090/api/v1/ingest?path=C:\your\docs"
```

### Watcher Not Detecting Changes

- Files must match supported extensions
- Check log output for errors
- Try manual re-ingest:
```bash
curl -X POST "http://localhost:9090/api/v1/ingest?path=C:\changed\file.md"
```

### Check Logs

Mindy logs to stdout. On Windows, capture logs:

```cmd
mindy.exe --watch "C:\docs" > mindy.log 2>&1
```

### View Stored Data

```cmd
# List blobs
dir "%USERPROFILE%\.mindy\data\blobs"

# Check TF-IDF vocabulary
type "%USERPROFILE%\.mindy\data\tfidf\vocab.json"

# Check document count
type "%USERPROFILE%\.mindy\data\tfidf\meta.json"
```

## Performance Tips

1. **Watch fewer directories** - More directories = more polling overhead
2. **Use exclusions** - Don't watch temp folders or caches
3. **Batch large directories** - Use the ingest API for bulk operations
4. **Limit search results** - Use `k` parameter to reduce processing
5. **Shallow traversal** - Use `depth=1` when possible

## Security Notes

- Mindy runs locally; data stays on your machine
- No authentication by default (local use only)
- Blob access is content-addressed (read-only)
- No network exposure by default

## Limitations

- **No PDF/DOCX support** (yet)
- **No semantic understanding** - TF-IDF is keyword-based
- **Single user** - No multi-tenant support
- **No real-time sync** - File watcher polls every 5 seconds
