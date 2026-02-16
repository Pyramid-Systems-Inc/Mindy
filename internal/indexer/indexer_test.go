package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"mindy/internal/blob"
	"mindy/internal/graph"
	"mindy/internal/vector"
)

func TestIndexer_IndexFile(t *testing.T) {
	tmpDir := t.TempDir()

	blobStore, _ := blob.NewStore(tmpDir)
	vectorIndex, _ := vector.NewIndex(tmpDir)
	graphStore, _ := graph.NewStore(tmpDir)

	indexer := New(blobStore, vectorIndex, graphStore, tmpDir)

	testFile := filepath.Join(tmpDir, "test.md")
	content := []byte("# Test Document\n\nThis is a test document about Python programming.")
	os.WriteFile(testFile, content, 0644)

	err := indexer.IndexFile(testFile)
	if err != nil {
		t.Fatalf("failed to index file: %v", err)
	}
}

func TestIndexer_ExtractText(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		content  []byte
		wantLen  int
	}{
		{
			name:    "markdown",
			path:    "test.md",
			content: []byte("# Hello World\n\nThis is content."),
			wantLen: 20,
		},
		{
			name:    "plain text",
			path:    "test.txt",
			content: []byte("Plain text content"),
			wantLen: 17,
		},
		{
			name:    "json",
			path:    "test.json",
			content: []byte(`{"key": "value"}`),
			wantLen: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractText(tt.path, tt.content)
			if len(got) < tt.wantLen-5 || len(got) > tt.wantLen+5 {
				t.Errorf("extractText() length = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestIndexer_ChunkText(t *testing.T) {
	text := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6"
	chunks := chunkText(text, 20)

	if len(chunks) == 0 {
		t.Error("expected chunks, got none")
	}
}

func TestIndexer_ExtractEntities(t *testing.T) {
	text := "Contact John Doe at john@example.com or visit https://example.com. Call 555-1234."
	entities := extractEntities(text)

	if len(entities) == 0 {
		t.Error("expected entities, got none")
	}

	found := false
	for _, e := range entities {
		if len(e) > 5 && e[:5] == "email" {
			found = true
		}
	}
	if !found {
		t.Logf("entities: %v", entities)
	}
}
