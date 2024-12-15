package vec

import (
	"strconv"
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

func String2Int(s string) (int, error) {
	return strconv.Atoi(s)
}

func FuzzString2Int(f *testing.F) {
	f.Add("123")
	f.Add("abc")
	f.Add("")

	f.Fuzz(func(t *testing.T, in string) {
		t.Logf("fuzzing test input is :%s", in)
		_, err := String2Int(in)
		if err != nil && in != "" {
			t.Errorf("unexpected error for input %q: %v", in, err)
		}
	})
}
