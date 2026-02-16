package indexer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mindy/internal/blob"
	"mindy/internal/extractor"
	"mindy/internal/graph"
	"mindy/internal/vector"
	"mindy/pkg/embedder"
)

type Indexer struct {
	blobStore   *blob.Store
	vectorIndex *vector.Index
	graphStore  *graph.Store
	embedder    *embedder.TFIDF
	dataDir     string
	extractor  *extractor.Extractor
	fileTracker *FileTracker
}

type FileTracker struct {
	dataDir string
	files   map[string]FileInfo
}

type FileInfo struct {
	Hash       string `json:"hash"`
	Modified   int64  `json:"modified"`
	IndexedAt  int64  `json:"indexed_at"`
	BlobRef    string `json:"blob_ref"`
	ChunkCount int    `json:"chunk_count"`
}

func New(blobStore *blob.Store, vectorIndex *vector.Index, graphStore *graph.Store, dataDir string) *Indexer {
	tfidf, _ := embedder.NewTFIDF(dataDir)
	tracker := NewFileTracker(dataDir)
	
	return &Indexer{
		blobStore:   blobStore,
		vectorIndex: vectorIndex,
		graphStore:  graphStore,
		embedder:    tfidf,
		dataDir:     dataDir,
		extractor:   extractor.New(),
		fileTracker: tracker,
	}
}

func NewFileTracker(dataDir string) *FileTracker {
	ft := &FileTracker{
		dataDir: dataDir,
		files:   make(map[string]FileInfo),
	}
	ft.load()
	return ft
}

func (ft *FileTracker) load() {
	path := filepath.Join(ft.dataDir, "file_tracker.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &ft.files)
}

func (ft *FileTracker) save() {
	path := filepath.Join(ft.dataDir, "file_tracker.json")
	data, _ := json.Marshal(ft.files)
	os.WriteFile(path, data, 0644)
}

func (ft *FileTracker) Get(path string) (FileInfo, bool) {
	info, ok := ft.files[path]
	return info, ok
}

func (ft *FileTracker) Set(path string, info FileInfo) {
	ft.files[path] = info
	ft.save()
}

func (ft *FileTracker) Remove(path string) {
	delete(ft.files, path)
	ft.save()
}

func (ft *FileTracker) Count() int {
	return len(ft.files)
}

func (i *Indexer) GetEmbedder() *embedder.TFIDF {
	return i.embedder
}

func (i *Indexer) IndexFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	currentHash := sha256ToString(content)
	
	if info, ok := i.fileTracker.Get(path); ok {
		if info.Hash == currentHash && info.Modified == stat.ModTime().Unix() {
			return nil
		}
	}

	blobHash, err := i.blobStore.Put(content)
	if err != nil {
		return fmt.Errorf("failed to store blob: %w", err)
	}

	docID := fmt.Sprintf("doc:%s", blobHash)
	
	existingDoc, _ := i.graphStore.GetNode(docID)
	if existingDoc != nil {
		i.removeDocumentFromIndex(docID)
	}

	metadata, _ := extractor.ExtractMetadata(path, content)
	metadata["content_type"] = i.extractor.GetContentType(path)
	metadata["file_type"] = i.extractor.GetFileType(path)

	docNode := &graph.Node{
		ID:       docID,
		Type:     "Document",
		Label:    filepath.Base(path),
		BlobRef:  blobHash,
		Props: map[string]interface{}{
			"path":         path,
			"size":         stat.Size(),
			"modified":     stat.ModTime().Unix(),
			"content_type": metadata["content_type"],
			"file_type":    metadata["file_type"],
		},
		CreateAt: time.Now().Unix(),
	}

	if err := i.graphStore.AddNode(docNode); err != nil {
		return fmt.Errorf("failed to add document node: %w", err)
	}

	text, err := i.extractor.Extract(path, content)
	if err != nil {
		return fmt.Errorf("failed to extract text: %w", err)
	}

	if i.embedder != nil {
		if err := i.embedder.AddDocument(docID, text); err != nil {
			fmt.Printf("Warning: failed to add document to TF-IDF: %v\n", err)
		}
	}

	chunks := chunkText(text, 512)
	chunkCount := 0

	for idx, chunk := range chunks {
		chunkID := fmt.Sprintf("chunk:%s:%d", blobHash, idx)
		chunkHash := sha256ToString([]byte(chunk))

		vec, err := i.embedder.Embed(chunk)
		if err != nil {
			continue
		}

		meta := fmt.Sprintf(`{"doc_id":"%s","chunk":%d,"path":"%s"}`, docID, idx, path)
		if err := i.vectorIndex.Add(chunkID+":"+chunkHash, vec, meta); err != nil {
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
		chunkCount++
	}

	i.fileTracker.Set(path, FileInfo{
		Hash:       currentHash,
		Modified:   stat.ModTime().Unix(),
		IndexedAt:  time.Now().Unix(),
		BlobRef:    blobHash,
		ChunkCount: chunkCount,
	})

	i.vectorIndex.Save()

	return nil
}

func (i *Indexer) removeDocumentFromIndex(docID string) {
	edges, _ := i.graphStore.GetNodeEdges(docID)
	
	for _, edge := range edges {
		if edge.Type == "HAS_CHUNK" {
			chunkID := edge.To
			chunkNode, _ := i.graphStore.GetNode(chunkID)
			if chunkNode != nil {
				if text, ok := chunkNode.Props["text"].(string); ok {
					i.embedder.AddDocument(chunkID+"_removed", text)
				}
			}
		}
	}
}

func (i *Indexer) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["embedder"] = i.embedder.GetStats()
	stats["file_tracker"] = map[string]interface{}{
		"tracked_files": i.fileTracker.Count(),
	}
	
	return stats
}

func (i *Indexer) ReindexAll() error {
	for path := range i.fileTracker.files {
		if err := i.IndexFile(path); err != nil {
			fmt.Printf("Error reindexing %s: %v\n", path, err)
		}
	}
	return nil
}

func (i *Indexer) GetFileCount() int {
	return i.fileTracker.Count()
}

func sha256ToString(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
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
	seen := make(map[string]bool)

	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	urlRegex := regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `]+`)
	phoneRegex := regexp.MustCompile(`(\+?1?[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`)
	dateRegex := regexp.MustCompile(`\b\d{1,2}[/-]\d{1,2}[/-]\d{2,4}\b|\b(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{1,2},?\s+\d{4}\b`)

	emails := emailRegex.FindAllString(text, -1)
	for _, e := range emails {
		if !seen[e] {
			seen[e] = true
			entities = append(entities, "email:"+e)
		}
	}

	urls := urlRegex.FindAllString(text, -1)
	for _, u := range urls {
		if !seen[u] {
			seen[u] = true
			entities = append(entities, "url:"+u)
		}
	}

	phones := phoneRegex.FindAllString(text, -1)
	for _, p := range phones {
		if !seen[p] {
			seen[p] = true
			entities = append(entities, "phone:"+p)
		}
	}

	dates := dateRegex.FindAllString(text, -1)
	for _, d := range dates {
		if !seen[d] {
			seen[d] = true
			entities = append(entities, "date:"+d)
		}
	}

	words := strings.Fields(text)
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if len(word) > 2 && isCapitalized(word) && !seen[word] {
			seen[word] = true
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
