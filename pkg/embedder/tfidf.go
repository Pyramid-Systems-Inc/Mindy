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
	"unicode"
)

var englishStopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true,
	"at": true, "be": true, "by": true, "for": true, "from": true,
	"has": true, "he": true, "in": true, "is": true, "it": true,
	"its": true, "of": true, "on": true, "that": true, "the": true,
	"to": true, "was": true, "will": true, "with": true, "this": true,
	"but": true, "they": true, "have": true, "had": true, "what": true,
	"when": true, "where": true, "who": true, "which": true, "why": true,
	"how": true, "all": true, "each": true, "every": true, "both": true,
	"few": true, "more": true, "most": true, "other": true, "some": true,
	"such": true, "no": true, "nor": true, "not": true, "only": true,
	"own": true, "same": true, "so": true, "than": true, "too": true,
	"very": true, "can": true, "just": true, "should": true, "now": true,
}

var commonSuffixes = []string{
	"ing", "ed", "ly", "ness", "ment", "tion", "sion", "ity",
	"ous", "ive", "able", "ible", "ful", "less", "er", "est",
}

type TFIDF struct {
	dim           int
	vocab         map[string]int
	idf           map[string]float32
	docCount      int
	docLengths    map[string]int
	avgDocLength  float32
	vectors       map[string][]float32
	mu            sync.RWMutex
	dataDir       string
	useBM25       bool
	k1            float32
	b             float32
}

type TFIDFConfig struct {
	Dimension int
	DataDir   string
	UseBM25   bool
	K1        float32
	B         float32
}

func NewTFIDF(dataDir string) (*TFIDF, error) {
	cfg := &TFIDFConfig{
		Dimension: 8192,
		DataDir:   dataDir,
		UseBM25:   false,
		K1:        1.5,
		B:         0.75,
	}
	return NewTFIDFWithConfig(cfg)
}

func NewTFIDFWithConfig(cfg *TFIDFConfig) (*TFIDF, error) {
	dim := cfg.Dimension
	if dim <= 0 {
		dim = 8192
	}

	t := &TFIDF{
		dim:        dim,
		vocab:      make(map[string]int),
		idf:        make(map[string]float32),
		docLengths: make(map[string]int),
		vectors:    make(map[string][]float32),
		dataDir:    cfg.DataDir,
		useBM25:    cfg.UseBM25,
		k1:         cfg.K1,
		b:          cfg.B,
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
		if idf == 0 {
			idf = float32(math.Log(float64(t.docCount+1))) + 1
		}
		queryVec[pos] = freq * idf
	}

	t.normalize(queryVec)
	return queryVec, nil
}

func (t *TFIDF) EmbedWithWeights(text string) ([]float32, map[string]float32, error) {
	terms := t.tokenize(text)
	tf := t.computeTF(terms)

	queryVec := make([]float32, t.dim)
	termWeights := make(map[string]float32)

	for term, freq := range tf {
		pos := t.hashTerm(term)
		idf := t.idf[term]
		if idf == 0 {
			idf = float32(math.Log(float64(t.docCount+1))) + 1
		}
		weight := freq * idf
		queryVec[pos] = weight
		termWeights[term] = weight
	}

	t.normalize(queryVec)
	return queryVec, termWeights, nil
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

	docLength := len(terms)
	t.docLengths[id] = docLength

	totalLength := 0
	for _, l := range t.docLengths {
		totalLength += l
	}
	if t.docCount > 0 {
		t.avgDocLength = float32(totalLength) / float32(t.docCount)
	}

	for term, df := range t.computeDF(terms) {
		oldIDF := t.idf[term]
		newIDF := float32(math.Log(float64(t.docCount+1)/float64(df+1)) + 1)
		if oldIDF == 0 {
			t.idf[term] = newIDF
		} else {
			t.idf[term] = (oldIDF*float32(oldDocCount) + newIDF) / float32(t.docCount)
		}
	}

	vec := make([]float32, t.dim)
	if t.useBM25 {
		for term, tf := range docTF {
			pos := t.hashTerm(term)
			idf := t.idf[term]
			if idf == 0 {
				idf = float32(math.Log(float64(t.docCount+1))) + 1
			}
			docLen := float32(docLength)
			tfNorm := (tf * (t.k1 + 1)) / (tf + t.k1*(1-t.b+t.b*docLen/t.avgDocLength))
			vec[pos] = tfNorm * idf
		}
	} else {
		for term, freq := range docTF {
			pos := t.hashTerm(term)
			idf := t.idf[term]
			if idf == 0 {
				idf = float32(math.Log(float64(t.docCount+1))) + 1
			}
			vec[pos] = freq * idf
		}
	}
	t.normalize(vec)

	t.vectors[id] = vec

	return t.save()
}

func (t *TFIDF) Search(query string, k int) ([]SearchResult, error) {
	queryVec, _, err := t.EmbedWithWeights(query)
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

func (t *TFIDF) SearchWithContext(query string, k int) ([]SearchResult, error) {
	results, err := t.Search(query, k)
	if err != nil {
		return nil, err
	}

	for i := range results {
		if chunk, ok := t.GetChunkFromID(results[i].ID); ok {
			results[i].Meta = chunk
		}
	}

	return results, nil
}

func (t *TFIDF) GetChunkFromID(id string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	docLen := t.docLengths[id]
	if docLen > 0 {
		return fmt.Sprintf("{\"doc_id\":\"%s\",\"length\":%d}", id, docLen), true
	}
	return "", false
}

func (t *TFIDF) GetVector(id string) ([]float32, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	vec, ok := t.vectors[id]
	return vec, ok
}

func (t *TFIDF) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	totalTerms := 0
	for _, v := range t.vectors {
		for _, f := range v {
			if f > 0 {
				totalTerms++
			}
		}
	}

	return map[string]interface{}{
		"doc_count":    t.docCount,
		"vocab_size":   len(t.vocab),
		"avg_doc_len":  t.avgDocLength,
		"total_terms":  totalTerms,
		"dimension":    t.dim,
		"algorithm":    "BM25",
		"k1":           t.k1,
		"b":            t.b,
	}
}

func (t *TFIDF) tokenize(text string) []string {
	text = strings.ToLower(text)

	var terms []string
	word := strings.Builder{}

	runes := []rune(text)
	for i, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word.WriteRune(r)
		} else if r == '\'' && word.Len() > 0 && i+1 < len(runes) && unicode.IsLetter(runes[i+1]) {
			word.WriteRune(r)
		} else {
			if word.Len() >= 2 {
				term := word.String()
				if !englishStopwords[term] {
					term = t.stem(term)
					if len(term) >= 2 {
						terms = append(terms, term)
					}
				}
			}
			word.Reset()
		}
	}

	if word.Len() >= 2 {
		term := word.String()
		if !englishStopwords[term] {
			term = t.stem(term)
			if len(term) >= 2 {
				terms = append(terms, term)
			}
		}
	}

	return terms
}

func (t *TFIDF) stem(word string) string {
	original := word

	for _, suffix := range commonSuffixes {
		if len(word) > len(suffix)+2 && strings.HasSuffix(word, suffix) {
			word = word[:len(word)-len(suffix)]
			break
		}
	}

	if len(word) < 3 {
		return original
	}

	return word
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
		DocCount     int              `json:"doc_count"`
		DocLengths   map[string]int   `json:"doc_lengths"`
		AvgDocLength float32          `json:"avg_doc_length"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	t.docCount = meta.DocCount
	t.docLengths = meta.DocLengths
	t.avgDocLength = meta.AvgDocLength

	if t.docLengths == nil {
		t.docLengths = make(map[string]int)
	}

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
		DocCount     int           `json:"doc_count"`
		DocLengths   map[string]int `json:"doc_lengths"`
		AvgDocLength float32       `json:"avg_doc_length"`
	}{
		DocCount:     t.docCount,
		DocLengths:   t.docLengths,
		AvgDocLength: t.avgDocLength,
	}
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
	fmt.Println("Enhanced TF-IDF/BM25 embedder loaded")
}
