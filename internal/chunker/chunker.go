package chunker

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	"garbell/internal/models"
)

// ParseFile parses a source file and returns a list of chunks representing
// the boundaries of functions, classes, and other meaningful structural blocks.
func ParseFile(filePath string) ([]models.Chunk, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	var p parser

	switch ext {
	case ".go":
		p = &goParser{}
	case ".py":
		p = &pythonParser{}
	case ".js", ".ts", ".jsx", ".tsx":
		p = &jsParser{}
	case ".c", ".cpp", ".cc", ".cxx", ".h", ".hpp":
		p = &cppParser{}
	case ".css":
		p = &cssParser{}
	case ".html", ".htm":
		p = &htmlParser{}
	case ".md", ".mdx":
		p = &markdownParser{}
	case ".proto":
		p = &protoParser{}
	default:
		// Unsupported extension
		return nil, nil
	}

	return parseLines(filePath, file, p)
}

// parser defines the interface for language-specific heuristic parsing.
type parser interface {
	// processLine takes the 1-indexed line number, the raw line text, and returns a slice of completed chunks.
	processLine(filePath string, lineNum int, lineText string) []models.Chunk

	// finalize is called cleanly at the end of the file in case a chunk was left open
	finalize(filePath string, totalLines int) []models.Chunk
}

func parseLines(filePath string, r io.Reader, p parser) ([]models.Chunk, error) {
	scanner := bufio.NewScanner(r)
	var chunks []models.Chunk
	lineNum := 1

	for scanner.Scan() {
		lineText := scanner.Text()
		if newChunks := p.processLine(filePath, lineNum, lineText); len(newChunks) > 0 {
			chunks = append(chunks, newChunks...)
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return chunks, err
	}

	if finalChunks := p.finalize(filePath, lineNum-1); len(finalChunks) > 0 {
		chunks = append(chunks, finalChunks...)
	}

	// Fallback for HTML or files with very few chunks but many lines
	if len(chunks) == 0 && lineNum > 50 {
		return slidingWindowChunks(filePath, lineNum-1), nil
	}

	return chunks, nil
}

func slidingWindowChunks(filePath string, totalLines int) []models.Chunk {
	var chunks []models.Chunk
	windowSize := 50
	overlap := 10

	for start := 1; start <= totalLines; start += (windowSize - overlap) {
		end := start + windowSize - 1
		if end > totalLines {
			end = totalLines
		}
		chunks = append(chunks, models.Chunk{
			File:  filePath,
			Start: start,
			End:   end,
			Sig:   "Sliding Window",
		})
		if end == totalLines {
			break
		}
	}
	return chunks
}
