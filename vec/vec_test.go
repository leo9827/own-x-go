package vec

import (
	"testing"
)

func TestVectorizationSentence(t *testing.T) {
	wordVectors, err := LoadWordVectors("data/glove.6B.50d.txt")
	if err != nil {
		t.Fatalf("Error loading word vectors: %v", err)
	}

	vec1 := SentenceToVector("love", wordVectors)
	vec2 := SentenceToVector("like", wordVectors)

	similarly1 := CosineSimilarly(vec1, vec2)
	t.Logf("'I love you' and 'I like you', vector similarly is:%v", similarly1)

	vector1 := SentenceToVector("I love you", wordVectors)
	vector2 := SentenceToVector("I like you", wordVectors)

	similarly2 := CosineSimilarly(vector1, vector2)
	t.Logf("'I love you' and 'I like you', vector similarly is:%v", similarly2)
}
