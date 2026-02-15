package indexer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mindy/internal/blob"
	"mindy/internal/graph"
	"mindy/internal/vector"
	"mindy/pkg/embedder"
)

type Indexer struct {
	blobStore  *blob.Store
	vectorIndex *vector.Index
	graphStore *graph.Store
	embedder   embedder.Embedder
}

func New(blobStore *blob.Store, vectorIndex *vector.Index, graphStore *graph.Store) *Indexer {
	return &Indexer{
		blobStore:   blobStore,
		vectorIndex: vectorIndex,
		graphStore:  graphStore,
		embedder:    embedder.NewRandom(384),
	}
}

func (i *Indexer) IndexFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	blobHash, err := i.blobStore.Put(content)
	if err != nil {
		return fmt.Errorf("failed to store blob: %w", err)
	}

	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	docID := fmt.Sprintf("doc:%s", blobHash)
	docNode := &graph.Node{
		ID:       docID,
		Type:     "Document",
		Label:    filepath.Base(path),
		BlobRef:  blobHash,
		Props: map[string]interface{}{
			"path":       path,
			"size":       stat.Size(),
			"modified":   stat.ModTime().Unix(),
			"content_type": getContentType(path),
		},
		CreateAt: time.Now().Unix(),
	}

	if err := i.graphStore.AddNode(docNode); err != nil {
		return fmt.Errorf("failed to add document node: %w", err)
	}

	text := extractText(path, content)
	chunks := chunkText(text, 512)

	for idx, chunk := range chunks {
		chunkID := fmt.Sprintf("chunk:%s:%d", blobHash, idx)
		chunkHash := sha256.Sum256([]byte(chunk))
		chunkHashStr := hex.EncodeToString(chunkHash[:])

		vec, err := i.embedder.Embed(chunk)
		if err != nil {
			continue
		}

		meta := fmt.Sprintf(`{"doc_id":"%s","chunk":%d,"path":"%s"}`, docID, idx, path)
		if err := i.vectorIndex.Add(chunkID+":"+chunkHashStr, vec, meta); err != nil {
			continue
		}

		chunkNode := &graph.Node{
			ID:       chunkID,
			Type:     "Chunk",
			Label:    fmt.Sprintf("Chunk %d", idx),
			BlobRef:  blobHash,
			Props: map[string]interface{}{
				"text":   chunk,
				"index":  idx,
				"doc_id": docID,
			},
			CreateAt: time.Now().Unix(),
		}
		i.graphStore.AddNode(chunkNode)

		i.graphStore.AddEdge(&graph.Edge{
			From:  docID,
			To:    chunkID,
			Type:  "HAS_CHUNK",
			Label: "",
		})

		entities := extractEntities(chunk)
		for _, entity := range entities {
			entityID := fmt.Sprintf("entity:%s", strings.ToLower(strings.ReplaceAll(entity, " ", "_")))
			entityNode := &graph.Node{
				ID:    entityID,
				Type:  "Entity",
				Label: entity,
				Props: map[string]interface{}{
					"name": entity,
				},
				CreateAt: time.Now().Unix(),
			}
			i.graphStore.AddNode(entityNode)

			i.graphStore.AddEdge(&graph.Edge{
				From:  chunkID,
				To:    entityID,
				Type:  "HAS_ENTITY",
				Label: "mentions",
			})
		}
	}

	return nil
}

func extractText(path string, content []byte) string {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".txt", ".md", ".markdown":
		return string(content)
	case ".html", ".htm":
		return stripHTML(string(content))
	case ".json":
		return string(content)
	default:
		return string(content)
	}
}

func stripHTML(html string) string {
	var buf bytes.Buffer
	inTag := false
	for _, c := range html {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
		} else if !inTag {
			buf.WriteRune(c)
		}
	}
	return buf.String()
}

func chunkText(text string, size int) []string {
	if len(text) <= size {
		return []string{text}
	}

	var chunks []string
	lines := strings.Split(text, "\n")
	var current strings.Builder

	for _, line := range lines {
		if current.Len()+len(line) > size && current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

func extractEntities(text string) []string {
	var entities []string
	words := strings.Fields(text)

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if len(word) > 3 && isCapitalized(word) {
			entities = append(entities, word)
		}
	}

	return entities
}

func isCapitalized(word string) bool {
	if len(word) == 0 {
		return false
	}
	first := word[0]
	return first >= 'A' && first <= 'Z'
}

func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt":
		return "text/plain"
	case ".md", ".markdown":
		return "text/markdown"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
