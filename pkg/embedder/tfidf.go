package embedder

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type TFIDF struct {
	dim         int
	vocab       map[string]int
	idf         map[string]float32
	docCount    int
	vectors     map[string][]float32
	mu          sync.RWMutex
	dataDir     string
}

type TFIDFConfig struct {
	Dimension int
	DataDir   string
}

func NewTFIDF(dataDir string) (*TFIDF, error) {
	dim := 4096

	t := &TFIDF{
		dim:     dim,
		vocab:   make(map[string]int),
		idf:     make(map[string]float32),
		vectors: make(map[string][]float32),
		dataDir: dataDir,
	}

	if err := t.load(); err == nil {
		return t, nil
	}

	return t, nil
}

func (t *TFIDF) Embed(text string) ([]float32, error) {
	terms := t.tokenize(text)
	tf := t.computeTF(terms)

	queryVec := make([]float32, t.dim)
	for term, freq := range tf {
		pos := t.hashTerm(term)
		idf := t.idf[term]
		queryVec[pos] = freq * idf
	}

	t.normalize(queryVec)
	return queryVec, nil
}

func (t *TFIDF) Dimension() int {
	return t.dim
}

func (t *TFIDF) AddDocument(id string, text string) error {
	terms := t.tokenize(text)
	if len(terms) == 0 {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	docTF := t.computeTF(terms)
	for term := range docTF {
		if _, exists := t.vocab[term]; !exists {
			t.vocab[term] = len(t.vocab)
		}
	}

	oldDocCount := t.docCount
	t.docCount++

	for term, df := range t.computeDF(terms) {
		oldIDF := t.idf[term]
		newIDF := float32(math.Log(float64(t.docCount+1) / float64(df+1)))
		if oldIDF == 0 {
			t.idf[term] = newIDF
		} else {
			t.idf[term] = (oldIDF*float32(oldDocCount) + newIDF) / float32(t.docCount)
		}
	}

	vec := make([]float32, t.dim)
	for term, freq := range docTF {
		pos := t.hashTerm(term)
		idf := t.idf[term]
		vec[pos] = freq * idf
	}
	t.normalize(vec)

	t.vectors[id] = vec

	return t.save()
}

func (t *TFIDF) Search(query string, k int) ([]SearchResult, error) {
	queryVec, err := t.Embed(query)
	if err != nil {
		return nil, err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	type result struct {
		id    string
		score float32
	}

	var results []result
	for id, vec := range t.vectors {
		score := cosineSimilarity(queryVec, vec)
		results = append(results, result{id: id, score: score})
	}

	sort.Slice(results, func(a, b int) bool {
		return results[a].score > results[b].score
	})

	if k > len(results) {
		k = len(results)
	}

	out := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		out[i] = SearchResult{
			ID:    results[i].id,
			Score: results[i].score,
		}
	}

	return out, nil
}

func (t *TFIDF) GetVector(id string) ([]float32, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	vec, ok := t.vectors[id]
	return vec, ok
}

func (t *TFIDF) tokenize(text string) []string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")

	var terms []string
	word := strings.Builder{}
	for _, c := range text {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			word.WriteRune(c)
		} else {
			if word.Len() >= 2 {
				terms = append(terms, word.String())
			}
			word.Reset()
		}
	}
	if word.Len() >= 2 {
		terms = append(terms, word.String())
	}

	return terms
}

func (t *TFIDF) computeTF(terms []string) map[string]float32 {
	tf := make(map[string]float32)
	for _, term := range terms {
		tf[term]++
	}
	for term := range tf {
		tf[term] = float32(math.Log1p(float64(tf[term])))
	}
	return tf
}

func (t *TFIDF) computeDF(terms []string) map[string]int {
	df := make(map[string]int)
	seen := make(map[string]bool)
	for _, term := range terms {
		if !seen[term] {
			df[term]++
			seen[term] = true
		}
	}
	return df
}

func (t *TFIDF) hashTerm(term string) int {
	h := fnv.New32a()
	h.Write([]byte(term))
	return int(h.Sum32()) % t.dim
}

func (t *TFIDF) normalize(vec []float32) {
	var norm float32
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		norm = float32(math.Sqrt(float64(norm)))
		for i := range vec {
			vec[i] /= norm
		}
	}
}

func (t *TFIDF) load() error {
	vocabFile := filepath.Join(t.dataDir, "tfidf", "vocab.json")
	data, err := os.ReadFile(vocabFile)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &t.vocab); err != nil {
		return err
	}

	idfFile := filepath.Join(t.dataDir, "tfidf", "idf.json")
	data, err = os.ReadFile(idfFile)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &t.idf); err != nil {
		return err
	}

	vectorsFile := filepath.Join(t.dataDir, "tfidf", "vectors.json")
	data, err = os.ReadFile(vectorsFile)
	if err != nil {
		return err
	}
	var vectors map[string][]float32
	if err := json.Unmarshal(data, &vectors); err != nil {
		return err
	}
	t.vectors = vectors

	metaFile := filepath.Join(t.dataDir, "tfidf", "meta.json")
	data, err = os.ReadFile(metaFile)
	if err != nil {
		return err
	}
	var meta struct {
		DocCount int `json:"doc_count"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	t.docCount = meta.DocCount

	return nil
}

func (t *TFIDF) save() error {
	dir := filepath.Join(t.dataDir, "tfidf")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	vocabData, err := json.Marshal(t.vocab)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "vocab.json"), vocabData, 0644); err != nil {
		return err
	}

	idfData, err := json.Marshal(t.idf)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "idf.json"), idfData, 0644); err != nil {
		return err
	}

	vectorsData, err := json.Marshal(t.vectors)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "vectors.json"), vectorsData, 0644); err != nil {
		return err
	}

	meta := struct {
		DocCount int `json:"doc_count"`
	}{DocCount: t.docCount}
	metaData, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), metaData, 0644); err != nil {
		return err
	}

	return nil
}

func (t *TFIDF) Close() error {
	return t.save()
}

type SearchResult struct {
	ID    string  `json:"id"`
	Score float32 `json:"score"`
	Meta  string  `json:"meta"`
}

func cosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

func init() {
	fmt.Println("TF-IDF embedder loaded")
}
