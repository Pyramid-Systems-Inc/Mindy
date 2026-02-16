package vector

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	DefaultDim          = 8192
	DefaultNlists       = 100
	DefaultNprobes     = 10
	DefaultKmeansIters = 10
)

type Vector struct {
	ID     string
	Vector []float32
	Meta   string
}

type Index struct {
	dim     int
	nlists  int
	nprobes int

	centroids [][]float32
	vectors   map[int][]Vector

	mu      sync.RWMutex
	dataDir string
	file    *os.File
	index   *os.File
}

func NewIndex(dataDir string) (*Index, error) {
	baseDir := filepath.Join(dataDir, "vector")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	idx := &Index{
		dim:     DefaultDim,
		nlists:  DefaultNlists,
		nprobes: DefaultNprobes,
		vectors: make(map[int][]Vector),
		dataDir: baseDir,
	}

	centroidsFile := filepath.Join(baseDir, "centroids.bin")
	if _, err := os.Stat(centroidsFile); err == nil {
		if err := idx.loadCentroids(centroidsFile); err != nil {
			return nil, err
		}
	} else {
		idx.centroids = idx.initCentroids()
	}

	return idx, nil
}

func (i *Index) Add(id string, vec []float32, meta string) error {
	if len(vec) != i.dim {
		return fmt.Errorf("dimension mismatch: got %d, want %d", len(vec), i.dim)
	}

	listID := i.assignCluster(vec)

	i.mu.Lock()
	defer i.mu.Unlock()

	i.vectors[listID] = append(i.vectors[listID], Vector{
		ID:     id,
		Vector: vec,
		Meta:   meta,
	})

	return nil
}

func (i *Index) Search(query []float32, k int) ([]SearchResult, error) {
	if len(query) != i.dim {
		return nil, fmt.Errorf("dimension mismatch: got %d, want %d", len(query), i.dim)
	}

	candidates := i.searchClusters(query, i.nprobes)

	type result struct {
		id    string
		score float32
		meta  string
	}

	var results []result
	for _, listID := range candidates {
		i.mu.RLock()
		for _, v := range i.vectors[listID] {
			score := cosineSimilarity(query, v.Vector)
			results = append(results, result{id: v.ID, score: score, meta: v.Meta})
		}
		i.mu.RUnlock()
	}

	sort.Slice(results, func(a, b int) bool {
		return results[a].score > results[b].score
	})

	if k > len(results) {
		k = len(results)
	}

	out := make([]SearchResult, k)
	for j := 0; j < k; j++ {
		out[j] = SearchResult{
			ID:    results[j].id,
			Score: results[j].score,
			Meta:  results[j].meta,
		}
	}

	return out, nil
}

func (i *Index) assignCluster(vec []float32) int {
	minDist := float32(math.MaxFloat32)
	best := 0
	for j, c := range i.centroids {
		dist := euclideanDist(vec, c)
		if dist < minDist {
			minDist = dist
			best = j
		}
	}
	return best
}

func (i *Index) searchClusters(query []float32, nprobes int) []int {
	type pair struct {
		dist float32
		id   int
	}

	var distances []pair
	for j, c := range i.centroids {
		dist := euclideanDist(query, c)
		distances = append(distances, pair{dist: dist, id: j})
	}

	sort.Slice(distances, func(a, b int) bool {
		return distances[a].dist < distances[b].dist
	})

	if nprobes > len(distances) {
		nprobes = len(distances)
	}

	result := make([]int, nprobes)
	for j := 0; j < nprobes; j++ {
		result[j] = distances[j].id
	}
	return result
}

func (i *Index) initCentroids() [][]float32 {
	centroids := make([][]float32, i.nlists)
	r := rand.New(rand.NewSource(42))
	for j := 0; j < i.nlists; j++ {
		centroids[j] = make([]float32, i.dim)
		for d := 0; d < i.dim; d++ {
			centroids[j][d] = r.Float32()*2 - 1
		}
	}
	return centroids
}

func (i *Index) loadCentroids(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	n := len(data) / (i.dim * 4)
	i.centroids = make([][]float32, n)

	for j := 0; j < n; j++ {
		i.centroids[j] = make([]float32, i.dim)
		for d := 0; d < i.dim; d++ {
			offset := (j*i.dim + d) * 4
			i.centroids[j][d] = math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4]))
		}
	}

	return nil
}

func (i *Index) Save() error {
	centroidsFile := filepath.Join(i.dataDir, "centroids.bin")
	f, err := os.Create(centroidsFile)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, c := range i.centroids {
		for _, v := range c {
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, math.Float32bits(v))
			if _, err := f.Write(buf); err != nil {
				return err
			}
		}
	}

	return nil
}

func (i *Index) Close() error {
	return i.Save()
}

func euclideanDist(a, b []float32) float32 {
	var sum float32
	for j := 0; j < len(a); j++ {
		d := a[j] - b[j]
		sum += d * d
	}
	return sum
}

func cosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float32
	for j := 0; j < len(a); j++ {
		dot += a[j] * b[j]
		normA += a[j] * a[j]
		normB += b[j] * b[j]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

type SearchResult struct {
	ID    string  `json:"id"`
	Score float32 `json:"score"`
	Meta  string  `json:"meta"`
}

type IndexOption func(*Index)

func WithDimension(dim int) IndexOption {
	return func(i *Index) {
		i.dim = dim
	}
}

func WithNLists(n int) IndexOption {
	return func(i *Index) {
		i.nlists = n
	}
}
