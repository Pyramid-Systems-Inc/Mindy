package embedder

import (
	"fmt"
	"testing"
)

func TestTFIDF_Dimension(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	if tfidf.Dimension() != 8192 {
		t.Errorf("expected dimension 8192, got %d", tfidf.Dimension())
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

	if len(vec) != 8192 {
		t.Errorf("expected vector size 8192, got %d", len(vec))
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
	if len(vec) != 8192 {
		t.Errorf("expected vector size 8192, got %d", len(vec))
	}
}

func TestTFIDF_Stopwords(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	terms := tfidf.tokenize("the quick brown fox")
	
	found := false
	for _, term := range terms {
		if term == "the" {
			found = true
			break
		}
	}
	if found {
		t.Error("stopwords should be filtered out")
	}
}

func TestTFIDF_Stemming(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	terms := tfidf.tokenize("programming development testing")
	
	if len(terms) == 0 {
		t.Error("expected terms after stemming")
	}
}

func TestTFIDF_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	tfidf.AddDocument("doc1", "test content")
	tfidf.AddDocument("doc2", "more test content")

	stats := tfidf.GetStats()
	
	if stats["doc_count"].(int) != 2 {
		t.Errorf("expected doc_count 2, got %v", stats["doc_count"])
	}
}

func TestTFIDF_EmbedWithWeights(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	tfidf.AddDocument("doc1", "python programming")

	vec, weights, err := tfidf.EmbedWithWeights("python tutorial")
	if err != nil {
		t.Fatalf("failed to embed with weights: %v", err)
	}

	if len(vec) != 8192 {
		t.Errorf("expected vector size 8192, got %d", len(vec))
	}

	if len(weights) == 0 {
		t.Error("expected term weights")
	}
}

func TestTFIDF_MultipleDocuments(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	docs := []string{
		"python programming basics",
		"javascript web development",
		"python advanced topics",
		"data science with python",
	}

	for i, doc := range docs {
		err = tfidf.AddDocument(fmt.Sprintf("doc%d", i), doc)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
	}

	results, err := tfidf.Search("python", 5)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("expected at least 2 results, got %d", len(results))
	}
}

func TestTFIDF_EmptyDocument(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfidf, err := NewTFIDF(tmpDir)
	if err != nil {
		t.Fatalf("failed to create TF-IDF: %v", err)
	}
	defer tfidf.Close()

	err = tfidf.AddDocument("doc1", "")
	if err != nil {
		t.Fatalf("failed to add empty document: %v", err)
	}

	results, err := tfidf.Search("test", 5)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 0 {
		t.Error("expected no results for empty document")
	}
}
