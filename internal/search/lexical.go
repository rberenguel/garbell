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

// SearchLexical uses `rg` to find matches, determines the enclosing chunk,
// and returns the deduplicated chunk bodies. When the estimated output would
// exceed the line threshold it returns a directory-grouped summary instead.
func SearchLexical(workspacePath, query string) ([]string, error) {
	cmd := exec.Command("rg", "-n", "-e", query)
	cmd.Dir = workspacePath

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, nil // no matches
	}

	// Phase 1: collect unique matching chunks without reading file bodies.
	seenChunks := make(map[string]bool)
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
				if !seenChunks[key] {
					seenChunks[key] = true
					matched = append(matched, chunk)
				}
				break
			}
		}
	}

	// Phase 2: estimate total output lines and check threshold.
	estimated := 0
	for _, c := range matched {
		estimated += 1 + min(c.End-c.Start+1, chunkBodyMaxLines) // header + body
	}
	if estimated > maxLines() {
		chunksByFile := make(map[string]int)
		for _, c := range matched {
			chunksByFile[c.File]++
		}
		return []string{lexicalOverflow(chunksByFile, len(matched), estimated)}, nil
	}

	// Phase 3: read bodies.
	var results []string
	for _, c := range matched {
		body, err := ReadChunkBody(workspacePath, c, chunkBodyMaxLines)
		if err == nil {
			header := fmt.Sprintf("// File: %s (L%d-L%d)\n", c.File, c.Start, c.End)
			results = append(results, header+body)
		}
	}
	return results, nil
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
