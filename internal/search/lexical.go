package search

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"garbell/internal/models"
)

const chunkBodyMaxLines = 100

// maxSummaryChunks returns the cap for SearchLexical compact output before falling back
// to a directory-grouped overview. Overridable via GARBELL_MAX_SUMMARY_CHUNKS.
func maxSummaryChunks() int {
	if v := os.Getenv("GARBELL_MAX_SUMMARY_CHUNKS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 80
}

// formatChunkSummary formats a slice of chunks as a compact, grouped-by-file listing.
// One line per chunk: "  <start>-<end>: <sig>". Used by SearchLexical and SearchRelated.
func formatChunkSummary(chunks []models.Chunk) string {
	var sb strings.Builder
	prevFile := ""
	for _, c := range chunks {
		if c.File != prevFile {
			if prevFile != "" {
				sb.WriteString("\n")
			}
			sb.WriteString(c.File + ":\n")
			prevFile = c.File
		}
		sb.WriteString(fmt.Sprintf("  %d-%d: %s\n", c.Start, c.End, c.Sig))
	}
	sb.WriteString("\nUse `read-chunk <file> <line>` to read any chunk's full body.")
	return sb.String()
}

// collectMatchingChunks runs rg for query and returns the deduplicated set of
// enclosing chunks, preserving rg match order. Used by SearchLexical and SearchRelated.
func collectMatchingChunks(workspacePath, query string) ([]models.Chunk, error) {
	cmd := exec.Command("rg", "-n", "-e", query)
	cmd.Dir = workspacePath

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, nil // no matches
	}

	seen := make(map[string]bool)
	var matched []models.Chunk

	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		relPath := parts[0]
		lineNum, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		fileChunks, err := loadChunksForFile(workspacePath, relPath)
		if err != nil || len(fileChunks) == 0 {
			continue
		}
		for _, chunk := range fileChunks {
			if lineNum >= chunk.Start && lineNum <= chunk.End {
				key := fmt.Sprintf("%s:%d-%d", chunk.File, chunk.Start, chunk.End)
				if !seen[key] {
					seen[key] = true
					matched = append(matched, chunk)
				}
				break
			}
		}
	}
	return matched, nil
}

// SearchLexical uses `rg` to find matches, determines the enclosing chunk,
// and returns a compact grouped-by-file summary (sig + line range per chunk).
// When results exceed maxSummaryChunks it falls back to a directory-grouped overview.
// Use `read-chunk <file> <line>` to fetch a specific chunk's full body.
func SearchLexical(workspacePath, query string) ([]string, error) {
	matched, err := collectMatchingChunks(workspacePath, query)
	if err != nil {
		return nil, err
	}
	if len(matched) == 0 {
		return nil, nil
	}

	if len(matched) > maxSummaryChunks() {
		chunksByFile := make(map[string]int)
		for _, c := range matched {
			chunksByFile[c.File]++
		}
		return []string{lexicalOverflow(chunksByFile, len(matched), len(matched))}, nil
	}

	return []string{formatChunkSummary(matched)}, nil
}

// ReadChunkBody reads the physical lines of code for a given chunk
func ReadChunkBody(workspacePath string, chunk models.Chunk, maxLines int) (string, error) {
	absPath := filepath.Join(workspacePath, chunk.File)
	file, err := os.Open(absPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var sb strings.Builder
	scanner := bufio.NewScanner(file)
	currentLine := 1
	linesRead := 0

	for scanner.Scan() {
		if currentLine > chunk.End {
			break
		}
		if currentLine >= chunk.Start {
			if linesRead >= maxLines {
				sb.WriteString(fmt.Sprintf("\n... (truncated after %d lines) ...\n", maxLines))
				break
			}
			sb.WriteString(scanner.Text() + "\n")
			linesRead++
		}
		currentLine++
	}

	return sb.String(), scanner.Err()
}
