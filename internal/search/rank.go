package search

import (
	"fmt"
	"sort"
)

// LargestChunks returns the n largest chunks by line count, descending.
// Answers: "where is the complexity in this codebase?"
func LargestChunks(workspacePath string, n int) ([]string, error) {
	allChunks, err := loadAllChunks(workspacePath)
	if err != nil {
		return nil, err
	}
	if len(allChunks) == 0 {
		return nil, nil
	}

	sort.Slice(allChunks, func(i, j int) bool {
		sizeI := allChunks[i].End - allChunks[i].Start
		sizeJ := allChunks[j].End - allChunks[j].Start
		return sizeI > sizeJ
	})

	if n <= 0 || n > len(allChunks) {
		n = len(allChunks)
	}

	results := make([]string, 0, n)
	for _, c := range allChunks[:n] {
		size := c.End - c.Start + 1
		results = append(results, fmt.Sprintf("%4d lines  %s  (%s:%d-%d)", size, c.Sig, c.File, c.Start, c.End))
	}
	return results, nil
}
