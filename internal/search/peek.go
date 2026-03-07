package search

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Peek reads lines around targetLine within radius lines from the original file.
// The target line is marked with ">>" in the output. Works directly on the source
// file — no index required. Useful for inspecting context around a specific line
// without needing to know which chunk it belongs to.
func Peek(workspacePath, relFilePath string, targetLine, radius int) (string, error) {
	if filepath.IsAbs(relFilePath) {
		if rel, err := filepath.Rel(workspacePath, relFilePath); err == nil {
			relFilePath = rel
		}
	}

	absPath := filepath.Join(workspacePath, relFilePath)
	file, err := os.Open(absPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	start := targetLine - radius
	if start < 1 {
		start = 1
	}
	end := targetLine + radius

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s:%d (±%d):\n", relFilePath, targetLine, radius)

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum > end {
			break
		}
		if lineNum < start {
			continue
		}
		marker := "  "
		if lineNum == targetLine {
			marker = ">>"
		}
		fmt.Fprintf(&sb, "%s %4d: %s\n", marker, lineNum, scanner.Text())
	}
	return sb.String(), scanner.Err()
}
