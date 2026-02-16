package embedder

import (
	"testing"
)

func TestTFIDF_Dimension(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	if tfidf.Dimension() != 4096 {
		t.Errorf("expected dimension 4096, got %d", tfidf.Dimension())
	}
}

func TestTFIDF_AddAndSearch(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	err = tfidf.AddDocument("doc1", "python programming language tutorial")
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	err = tfidf.AddDocument("doc2", "javascript web development guide")
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	results, err := tfidf.Search("python tutorial", 2)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}

	if results[0].ID != "doc1" {
		t.Errorf("expected doc1 as top result, got %s", results[0].ID)
	}
}

func TestTFIDF_Embed(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	vec, err := tfidf.Embed("test query")
	if err != nil {
		t.Fatalf("failed to embed: %v", err)
	}

	if len(vec) != 4096 {
		t.Errorf("expected vector size 4096, got %d", len(vec))
	}
}

func TestTFIDF_Tokenize(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	terms := tfidf.tokenize("Hello World Test 123")
	if len(terms) == 0 {
		t.Error("expected tokens, got none")
	}
}

func TestTFIDF_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf1, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}

	err = tfidf1.AddDocument("doc1", "test content for persistence")
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}
	tfidf1.Close()

	tfidf2, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf2.Close()

	vec, ok := tfidf2.GetVector("doc1")
	if !ok {
		t.Error("expected document to be loaded from persistence")
	}
	if len(vec) != 4096 {
		t.Errorf("expected vector size 4096, got %d", len(vec))
	}
}
