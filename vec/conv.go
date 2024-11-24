package vec

import (
	"strings"
)

// SentenceToVector By statistically the average vector of all words in this sentence
func SentenceToVector(sentence string, wordVectors *WordVector) []float64 {
	words := strings.Fields(sentence)

	vectorDim := len((*wordVectors)[words[0]])
	if vectorDim == 0 {
		vectorDim = 50 // default vector dimension: 50
	}
	avgVector := make([]float64, vectorDim)
	wordsCount := 0
	for _, word := range words {
		if vec, exists := (*wordVectors)[word]; exists {
			for i, val := range vec {
				avgVector[i] += val
			}
			wordsCount++
		}
	}
	// got average vector
	if wordsCount > 0 {
		for i := range avgVector {
			avgVector[i] = avgVector[i] / float64(wordsCount)
		}
	}
	return avgVector
}
