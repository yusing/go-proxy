package utils

import (
	"fmt"
	"os"
	"path"
)

// Recursively lists all files in a directory until `maxDepth` is reached
// Returns a slice of file paths relative to `dir`.
func ListFiles(dir string, maxDepth int, hideHidden ...bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error listing directory %s: %w", dir, err)
	}
	hideHiddenFiles := len(hideHidden) > 0 && hideHidden[0]
	files := make([]string, 0)
	for _, entry := range entries {
		if hideHiddenFiles && entry.Name()[0] == '.' {
			continue
		}
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

// FileExists checks if a file exists.
//
// If the file does not exist, it returns false and nil,
// otherwise it returns true and any error that is not os.ErrNotExist.
func FileExists(file string) (bool, error) {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
