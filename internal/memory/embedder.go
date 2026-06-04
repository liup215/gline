package memory

import (
	"context"
	"fmt"
	"math"
)

// Embedder produces dense vector embeddings for text.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
	ModelName() string
	MaxBatchSize() int
}

// EmbedAndNormalize computes embeddings and L2-normalises each vector
// so that cosine similarity becomes a simple dot product.
func EmbedAndNormalize(ctx context.Context, e Embedder, texts []string) ([][]float32, error) {
	vecs, err := e.Embed(ctx, texts)
	if err != nil {
		return nil, err
	}
	for i := range vecs {
		vecs[i] = normalizeL2(vecs[i])
	}
	return vecs, nil
}

func normalizeL2(v []float32) []float32 {
	var s float64
	for _, x := range v {
		s += float64(x) * float64(x)
	}
	if s == 0 {
		return v
	}
	n := float32(math.Sqrt(s))
	for i := range v {
		v[i] /= n
	}
	return v
}

// CosineSimilarity assumes pre-normalised vectors and returns dot product.
func CosineSimilarity(a, b []float32) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("dimension mismatch: %d vs %d", len(a), len(b))
	}
	var s float64
	for i := range a {
		s += float64(a[i]) * float64(b[i])
	}
	return s, nil
}

// TopK cosine-similarity search with fixed dimension.
func TopK(query []float32, vectors [][]float32, k int) ([]int, []float64) {
	type score struct{ idx int; sim float64 }
	scores := make([]score, 0, len(vectors))
	for i, v := range vectors {
		s, _ := CosineSimilarity(query, v)
		scores = append(scores, score{i, s})
	}
	
	// heap-less sort for small N
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].sim > scores[i].sim {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	if k > len(scores) {
		k = len(scores)
	}
	idx := make([]int, k)
	sim := make([]float64, k)
	for i := 0; i < k; i++ {
		idx[i] = scores[i].idx
		sim[i] = scores[i].sim
	}
	return idx, sim
}
