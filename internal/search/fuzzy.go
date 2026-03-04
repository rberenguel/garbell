package search

import (
	"strings"

	"garbell/internal/models"
)

// SearchFuzzy searches across all shards to find the chunk whose signature
// has the smallest Levenshtein distance to the provided signature query.
func SearchFuzzy(workspacePath, signature string) (models.Chunk, string, error) {
	allChunks, err := loadAllChunks(workspacePath)
	if err != nil {
		return models.Chunk{}, "", err
	}

	if len(allChunks) == 0 {
		return models.Chunk{}, "", nil
	}

	var bestChunk models.Chunk
	bestDist := -1

	// very simple trigram/levenshtein
	lowerQuery := strings.ToLower(signature)

	for _, chunk := range allChunks {
		lowerSig := strings.ToLower(chunk.Sig)
		dist := levenshtein(lowerQuery, lowerSig)

		if bestDist == -1 || dist < bestDist {
			bestDist = dist
			bestChunk = chunk
		}
	}

	body, err := ReadChunkBody(workspacePath, bestChunk, 100)
	return bestChunk, body, err
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// levenshtein computes the distance between two strings
func levenshtein(s, t string) int {
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
	}
	for i := range d {
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				d[i][j] = min3(
					d[i-1][j]+1,   // deletion
					d[i][j-1]+1,   // insertion
					d[i-1][j-1]+1, // substitution
				)
			}
		}
	}
	return d[len(s)][len(t)]
}
