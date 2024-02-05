package obsidian

import "testing"

func TestScanUnusedImages(t *testing.T) {

	rootDir := "/Users/leo9827/Documents/on-the-road/"
	imgDir := "/Users/leo9827/Documents/on-the-road/Illustrations"
	ScanUnusedImages(rootDir, imgDir)
}
