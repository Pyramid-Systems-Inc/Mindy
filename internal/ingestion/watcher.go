package ingestion

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mindy/internal/indexer"
)

type Watcher struct {
	paths   []string
	indexer *indexer.Indexer
	events  chan string
	stop    chan struct{}
}

func NewWatcher(paths []string, indexer *indexer.Indexer) (*Watcher, error) {
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			if err := os.MkdirAll(p, 0755); err != nil {
				return nil, err
			}
		}
	}

	w := &Watcher{
		paths:   paths,
		indexer: indexer,
		events:  make(chan string, 100),
		stop:    make(chan struct{}),
	}

	return w, nil
}

func (w *Watcher) Start() {
	go w.scanInitial()
	go w.watchLoop()
}

func (w *Watcher) Stop() {
	close(w.stop)
}

func (w *Watcher) scanInitial() {
	for _, path := range w.paths {
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if isIndexable(p) {
				w.events <- p
			}
			return nil
		})
	}
}

func (w *Watcher) watchLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.checkChanges()
		case path := <-w.events:
			go w.processFile(path)
		case <-w.stop:
			return
		}
	}
}

func (w *Watcher) checkChanges() {
	for _, path := range w.paths {
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if isIndexable(p) && info.ModTime().After(time.Now().Add(-10*time.Second)) {
				w.events <- p
			}
			return nil
		})
	}
}

func (w *Watcher) processFile(path string) {
	log.Printf("Indexing: %s", path)
	if err := w.indexer.IndexFile(path); err != nil {
		log.Printf("Error indexing %s: %v", path, err)
	} else {
		log.Printf("Indexed: %s", path)
	}
}

func isIndexable(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	indexableExts := map[string]bool{
		".txt":     true,
		".md":      true,
		".markdown": true,
		".html":    true,
		".htm":     true,
		".json":    true,
		".xml":     true,
		".csv":     true,
		".log":     true,
	}
	return indexableExts[ext]
}
