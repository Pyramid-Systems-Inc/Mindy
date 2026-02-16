package blob

import (
	"path/filepath"
	"testing"
)

func TestStore_PutAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	content := []byte("hello world test content")
	
	hash, err := store.Put(content)
	if err != nil {
		t.Fatalf("failed to put: %v", err)
	}

	if hash == "" {
		t.Fatal("hash should not be empty")
	}

	retrieved, err := store.Get(hash)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	if string(retrieved) != string(content) {
		t.Errorf("content mismatch: got %s, want %s", string(retrieved), string(content))
	}
}

func TestStore_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	content := []byte("same content")
	
	hash1, _ := store.Put(content)
	hash2, _ := store.Put(content)

	if hash1 != hash2 {
		t.Errorf("same content should produce same hash: %s != %s", hash1, hash2)
	}
}

func TestStore_Has(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	content := []byte("test content")
	hash, _ := store.Put(content)

	exists, err := store.Has(hash)
	if err != nil {
		t.Fatalf("failed to check has: %v", err)
	}
	if !exists {
		t.Error("expected hash to exist")
	}

	exists, _ = store.Has("nonexistent")
	if exists {
		t.Error("expected hash to not exist")
	}
}

func TestStore_Path(t *testing.T) {
	tmpDir := t.TempDir()
	
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	content := []byte("test")
	hash, _ := store.Put(content)

	path := store.Path(hash)
	if !filepath.IsAbs(path) {
		t.Error("expected absolute path")
	}
}
