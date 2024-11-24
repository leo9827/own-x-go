package vec

import "strings"

func BagOfWords(corpus []string) map[string]int {
	vocab := make(map[string]int)
	for _, doc := range corpus {
		words := strings.Fields(doc)
		for _, word := range words {
			vocab[word]++
		}
	}
	return vocab
}
