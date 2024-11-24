// Word2Vec 的 Skip-gram 模型优化目标是：预测上下文单词的概率。

package vec

// pseudocode

//func TrainWord2Vec(sentences [][]string, embeddingDim int, epochs int) map[string][]float64 {
//	vocab := buildVocabulary(sentences)
//	vectors := initializeVectors(vocab, embeddingDim)
//
//	for epoch :=0; epoch < epochs; epoch++ {
//		for _, sentence := range sentences {
//			for targetIndex, target := range sentence {
//				context := getContext(sentence, targetIndex)
//				updateVectors(target, context, vectors)
//			}
//		}
//	}
//	return vectors;
//}
