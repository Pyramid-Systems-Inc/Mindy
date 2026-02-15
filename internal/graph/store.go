package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
)

type Node struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Label    string                 `json:"label"`
	Props    map[string]interface{} `json:"props,omitempty"`
	BlobRef  string                 `json:"blob_ref,omitempty"`
	CreateAt int64                  `json:"created_at"`
}

type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Type     string `json:"type"`
	Label    string `json:"label,omitempty"`
	Props    map[string]interface{} `json:"props,omitempty"`
	Weight   float32 `json:"weight,omitempty"`
}

type Store struct {
	db *badger.DB
}

func NewStore(dataDir string) (*Store, error) {
	baseDir := filepath.Join(dataDir, "graph")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions(baseDir)
	opts.IndexCacheSize = 100 << 20

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) AddNode(node *Node) error {
	return s.db.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(node)
		if err != nil {
			return err
		}
		key := []byte("node:" + node.ID)
		return txn.Set(key, data)
	})
}

func (s *Store) GetNode(id string) (*Node, error) {
	var node Node
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("node:" + id))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &node)
		})
	})
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (s *Store) AddEdge(edge *Edge) error {
	return s.db.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(edge)
		if err != nil {
			return err
		}
		edgeKey := fmt.Sprintf("edge:%s:%s:%s", edge.From, edge.Type, edge.To)
		if err := txn.Set([]byte(edgeKey), data); err != nil {
			return err
		}

		fromEdgesKey := []byte("out:" + edge.From)
		var fromEdges []byte
		item, err := txn.Get(fromEdgesKey)
		if err == nil {
			err = item.Value(func(val []byte) error {
				fromEdges = make([]byte, len(val))
				copy(fromEdges, val)
				return nil
			})
			if err != nil && err.Error() != "Key not found" {
				return err
			}
		}
		fromEdges = append(fromEdges, []byte(edgeKey+"\n")...)
		if err := txn.Set(fromEdgesKey, fromEdges); err != nil {
			return err
		}

		toEdgesKey := []byte("in:" + edge.To)
		var toEdges []byte
		item2, err := txn.Get(toEdgesKey)
		if err == nil {
			err = item2.Value(func(val []byte) error {
				toEdges = make([]byte, len(val))
				copy(toEdges, val)
				return nil
			})
			if err != nil && err.Error() != "Key not found" {
				return err
			}
		}
		toEdges = append(toEdges, []byte(edgeKey+"\n")...)
		if err := txn.Set(toEdgesKey, toEdges); err != nil {
			return err
		}

		return nil
	})
}

func (s *Store) GetNodeEdges(nodeID string) ([]*Edge, error) {
	var edges []*Edge

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("out:" + nodeID))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			keys := splitKeys(val)
			for _, key := range keys {
				edgeStr := string(key)
				item, err := txn.Get([]byte(edgeStr))
				if err != nil {
					continue
				}
				var edge Edge
				if err := item.Value(func(v []byte) error {
					return json.Unmarshal(v, &edge)
				}); err != nil {
					continue
				}
				edges = append(edges, &edge)
			}
			return nil
		})
	})

	return edges, err
}

func (s *Store) Traverse(start string, edgeType string, depth int) ([]*Node, error) {
	visited := make(map[string]bool)
	var result []*Node
	queue := []string{start}

	for len(queue) > 0 && depth > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		node, err := s.GetNode(current)
		if err == nil {
			result = append(result, node)
		}

		edges, _ := s.GetNodeEdges(current)
		for _, edge := range edges {
			if edgeType == "" || edge.Type == edgeType {
				queue = append(queue, edge.To)
			}
		}
		depth--
	}

	return result, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func splitKeys(data []byte) [][]byte {
	var keys [][]byte
	current := []byte{}
	for _, b := range data {
		if b == '\n' {
			if len(current) > 0 {
				keys = append(keys, current)
				current = []byte{}
			}
		} else {
			current = append(current, b)
		}
	}
	if len(current) > 0 {
		keys = append(keys, current)
	}
	return keys
}
