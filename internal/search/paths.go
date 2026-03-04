package search

import (
	"sort"
)

// IndexedPaths returns a sorted list of all unique file paths currently in the index.
func IndexedPaths(workspacePath string) ([]string, error) {
	chunks, err := loadAllChunks(workspacePath)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var paths []string

	for _, chunk := range chunks {
		if !seen[chunk.File] {
			seen[chunk.File] = true
			paths = append(paths, chunk.File)
		}
	}

	sort.Strings(paths)
	return paths, nil
}
