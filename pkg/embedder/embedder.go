package embedder

import (
	"math/rand"
)

type Embedder interface {
	Embed(text string) ([]float32, error)
	Dimension() int
}

type RandomEmbedder struct {
	dim int
}

func NewRandom(dim int) *RandomEmbedder {
	return &RandomEmbedder{dim: dim}
}

func (e *RandomEmbedder) Embed(text string) ([]float32, error) {
	vec := make([]float32, e.dim)
	r := rand.New(rand.NewSource(hashString(text)))
	for i := 0; i < e.dim; i++ {
		vec[i] = r.Float32()*2 - 1
	}
	return vec, nil
}

func (e *RandomEmbedder) Dimension() int {
	return e.dim
}

func hashString(s string) int64 {
	var h int64
	for i, c := range s {
		h = h*31 + int64(c)*int64(i+1)
	}
	return h
}
