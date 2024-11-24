package vec

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type WordVector map[string][]float64

// LoadWordVectors Load loads a vector from a file.
func LoadWordVectors(filepath string) (*WordVector, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	wordVectors := make(WordVector)
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		word := fields[0]
		vector := make([]float64, len(fields)-1)
		for i, val := range fields[1:] {
			vector[i], _ = strconv.ParseFloat(val, 64)
		}
		wordVectors[word] = vector
	}
	return &wordVectors, nil
}
