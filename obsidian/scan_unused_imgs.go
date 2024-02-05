package obsidian

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var ignoreDirs = map[string]int8{"Illustrations": 1, ".git": 1, ".obsidian": 1, ".trash": 1}

func ScanUnusedImages(root, imgDir string) {
	existsImages := make(map[string]int8)
	entries, err := os.ReadDir(imgDir)
	if err != nil {
		fmt.Println(err)
	}
	for _, entry := range entries {
		existsImages[entry.Name()] = 0
	}

	rootDir := root
	scan(rootDir, existsImages)

	fmt.Println("unused images:")
	for k, v := range existsImages {
		if v == 0 {
			fmt.Println(k)
		}
	}
}

func scan(rootDir string, existsImages map[string]int8) {
	fmt.Printf("start scan dir: %s\n", rootDir)
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		fmt.Printf("os.ReadDir %s, error: %v", rootDir, err)
	}
	for _, entry := range entries {
		if _, ignore := ignoreDirs[entry.Name()]; ignore {
			continue
		}
		if strings.LastIndex(entry.Name(), ".md") > 0 {
			// read md file in line
			filePath := rootDir + entry.Name()
			fmt.Printf("start scan file: %s\n", filePath)
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Println(err)
				continue
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				// Process each line of the Markdown file here
				if strings.Contains(line, "![[") {
					img := line[strings.Index(line, "![[")+3 : strings.LastIndex(line, "]]")]
					fmt.Printf("scan file, find img: %s\n", img)
					if _, ok := existsImages[img]; ok {
						existsImages[img] = 1
					}
				}
				if err := scanner.Err(); err != nil {
					fmt.Println(err)
				}
			}

			file.Close()
		}
		if entry.IsDir() {
			subDir := rootDir + entry.Name() + "/"
			scan(subDir, existsImages)
		}
	}
}
