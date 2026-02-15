package blob

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	baseDir string
	mu      sync.RWMutex
}

func NewStore(dataDir string) (*Store, error) {
	baseDir := filepath.Join(dataDir, "blobs")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &Store{baseDir: baseDir}, nil
}

func (s *Store) Put(content []byte) (string, error) {
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	dir := filepath.Join(s.baseDir, hashStr[:2])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, hashStr[2:])
	exists, err := exists(path)
	if err != nil {
		return "", err
	}

	if !exists {
		if err := os.WriteFile(path, content, 0644); err != nil {
			return "", err
		}
	}

	return hashStr, nil
}

func (s *Store) Get(hash string) ([]byte, error) {
	if len(hash) < 2 {
		return nil, fmt.Errorf("invalid hash length")
	}

	path := filepath.Join(s.baseDir, hash[:2], hash[2:])
	return os.ReadFile(path)
}

func (s *Store) Path(hash string) string {
	return filepath.Join(s.baseDir, hash[:2], hash[2:])
}

func (s *Store) Has(hash string) (bool, error) {
	if len(hash) < 2 {
		return false, nil
	}
	path := filepath.Join(s.baseDir, hash[:2], hash[2:])
	return exists(path)
}

func (s *Store) Close() error {
	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func HashReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
