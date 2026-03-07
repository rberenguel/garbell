package search

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"garbell/internal/models"
	"garbell/internal/semantic"
)

// maxRelatedChunks caps SearchRelated output. Synonym expansion can match broadly;
// we show the most relevant hits first (original query > synonyms) and note the rest.
const maxRelatedChunks = 50

// SearchRelated expands the query using the PPMI thesaurus, scores matching chunks
// (original-query matches score higher than synonym-only matches), and returns the
// top maxRelatedChunks results with a note if more were found.
// fileFilter (if non-empty) restricts results to files whose path matches the regex.
func SearchRelated(workspacePath, query, fileFilter string) ([]string, error) {
	thesaurus, err := loadThesaurus(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("ppmi.json not found — run 'index' first: %w", err)
	}

	tokens := semantic.Tokenize(query)
	if len(tokens) == 0 {
		return SearchLexical(workspacePath, query, fileFilter)
	}

	// Build synonym set (terms NOT already in the original query).
	origSet := make(map[string]bool, len(tokens))
	for _, tok := range tokens {
		origSet[tok] = true
	}
	synSet := make(map[string]bool)
	for _, tok := range tokens {
		for _, syn := range thesaurus[tok] {
			if !origSet[syn] {
				synSet[syn] = true
			}
		}
	}

	// Pass 1: chunks matching the original query tokens (score 2).
	origRegex := "(?i)(" + strings.Join(tokens, "|") + ")"
	origChunks, err := collectMatchingChunks(workspacePath, origRegex, fileFilter)
	if err != nil {
		return nil, err
	}

	// Pass 2: chunks matching synonym-only terms (score 1).
	var synChunks []models.Chunk
	if len(synSet) > 0 {
		synTerms := make([]string, 0, len(synSet))
		for s := range synSet {
			synTerms = append(synTerms, s)
		}
		synRegex := "(?i)(" + strings.Join(synTerms, "|") + ")"
		synChunks, err = collectMatchingChunks(workspacePath, synRegex, fileFilter)
		if err != nil {
			return nil, err
		}
	}

	// Merge: original matches first (higher relevance), then synonym-only additions.
	seen := make(map[string]bool)
	var ranked []models.Chunk
	for _, c := range origChunks {
		key := fmt.Sprintf("%s:%d-%d", c.File, c.Start, c.End)
		if !seen[key] {
			seen[key] = true
			ranked = append(ranked, c)
		}
	}
	for _, c := range synChunks {
		key := fmt.Sprintf("%s:%d-%d", c.File, c.Start, c.End)
		if !seen[key] {
			seen[key] = true
			ranked = append(ranked, c)
		}
	}

	total := len(ranked)
	if total > maxRelatedChunks {
		ranked = ranked[:maxRelatedChunks]
	}

	result := formatChunkSummary(ranked)
	if total > maxRelatedChunks {
		result += fmt.Sprintf("\n... %d more chunks matched. Refine your query or use `search-lexical` for exact results.", total-maxRelatedChunks)
	}
	return []string{result}, nil
}

func loadThesaurus(workspacePath string) (map[string][]string, error) {
	absWorkspace, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dirHashBytes := md5.Sum([]byte(absWorkspace))
	dirHash := hex.EncodeToString(dirHashBytes[:])
	ppmiPath := filepath.Join(home, ".garbell", "indexes", dirHash, "ppmi.json")

	data, err := os.ReadFile(ppmiPath)
	if err != nil {
		return nil, err
	}

	var thesaurus map[string][]string
	if err := json.Unmarshal(data, &thesaurus); err != nil {
		return nil, err
	}
	return thesaurus, nil
}
