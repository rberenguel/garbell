package search

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"garbell/internal/semantic"
)

// SearchRelated expands the query using the PPMI thesaurus built during indexing,
// then delegates to SearchLexical with the resulting expanded regex.
func SearchRelated(workspacePath, query string) ([]string, error) {
	thesaurus, err := loadThesaurus(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("ppmi.json not found — run 'index' first: %w", err)
	}

	tokens := semantic.Tokenize(query)
	if len(tokens) == 0 {
		// Fall back to plain lexical search if the query tokenises to nothing.
		return SearchLexical(workspacePath, query)
	}

	termSet := make(map[string]bool)
	for _, tok := range tokens {
		termSet[tok] = true
	}
	for _, tok := range tokens {
		for _, syn := range thesaurus[tok] {
			termSet[syn] = true
		}
	}

	terms := make([]string, 0, len(termSet))
	for t := range termSet {
		terms = append(terms, t)
	}

	expanded := "(?i)(" + strings.Join(terms, "|") + ")"
	return SearchLexical(workspacePath, expanded)
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
