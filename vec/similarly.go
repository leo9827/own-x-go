package vec

import "math"

func CosineSimilarly(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		panic("Vectors must have the same len")
	}

	dotProduct, normA, normB := 0.0, 0.0, 0.0

	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		normA += vec1[i] * vec2[i]
		normB += vec1[i] * vec2[i]
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
