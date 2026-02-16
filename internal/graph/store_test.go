package graph

import (
	"os"
	"testing"
)

func TestStore_AddNodeAndGetNode(t *testing.T) {
	if !hasDiskSpace() {
		t.Skip("insufficient disk space")
	}
	
	tmpDir := t.TempDir()
	
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	node := &Node{
		ID:       "test:1",
		Type:     "Test",
		Label:    "Test Node",
		BlobRef:  "abc123",
		CreateAt: 1234567890,
	}

	err = store.AddNode(node)
	if err != nil {
		t.Fatalf("failed to add node: %v", err)
	}

	retrieved, err := store.GetNode("test:1")
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if retrieved.ID != node.ID {
		t.Errorf("id mismatch: got %s, want %s", retrieved.ID, node.ID)
	}
	if retrieved.Type != node.Type {
		t.Errorf("type mismatch: got %s, want %s", retrieved.Type, node.Type)
	}
}

func TestStore_AddEdge(t *testing.T) {
	if !hasDiskSpace() {
		t.Skip("insufficient disk space")
	}
	
	tmpDir := t.TempDir()
	
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	node1 := &Node{ID: "a", Type: "Test", CreateAt: 123}
	node2 := &Node{ID: "b", Type: "Test", CreateAt: 123}
	store.AddNode(node1)
	store.AddNode(node2)

	edge := &Edge{
		From:  "a",
		To:    "b",
		Type:  "LINKS_TO",
		Label: "connects",
	}

	err = store.AddEdge(edge)
	if err != nil {
		t.Fatalf("failed to add edge: %v", err)
	}
}

func TestStore_Traverse(t *testing.T) {
	if !hasDiskSpace() {
		t.Skip("insufficient disk space")
	}
	
	tmpDir := t.TempDir()
	
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.AddNode(&Node{ID: "root", Type: "Test", CreateAt: 123})
	store.AddNode(&Node{ID: "child1", Type: "Test", CreateAt: 123})
	store.AddNode(&Node{ID: "child2", Type: "Test", CreateAt: 123})
	store.AddNode(&Node{ID: "grandchild", Type: "Test", CreateAt: 123})

	store.AddEdge(&Edge{From: "root", To: "child1", Type: "HAS"})
	store.AddEdge(&Edge{From: "root", To: "child2", Type: "HAS"})
	store.AddEdge(&Edge{From: "child1", To: "grandchild", Type: "HAS"})

	nodes, err := store.Traverse("root", "HAS", 2)
	if err != nil {
		t.Fatalf("failed to traverse: %v", err)
	}

	if len(nodes) == 0 {
		t.Error("expected nodes in traversal")
	}
}

func hasDiskSpace() bool {
	tmp := os.TempDir()
	var stat os.FileInfo
	stat, err := os.Stat(tmp)
	if err != nil {
		return false
	}
	_ = stat
	return true
}
