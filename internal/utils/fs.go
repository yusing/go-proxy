package utils

import (
	"fmt"
	"os"
	"path"
)

// Recursively lists all files in a directory until `maxDepth` is reached
// Returns a slice of file paths relative to `dir`
func ListFiles(dir string, maxDepth int) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error listing directory %s: %w", dir, err)
	}
	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			if maxDepth <= 0 {
				continue
			}
			subEntries, err := ListFiles(path.Join(dir, entry.Name()), maxDepth-1)
			if err != nil {
				return nil, err
			}
			files = append(files, subEntries...)
		} else {
			files = append(files, path.Join(dir, entry.Name()))
		}
	}
	return files, nil
}
