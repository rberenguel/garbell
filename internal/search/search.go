package search

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"garbell/internal/models"
)

// FileSkeleton returns line numbers and signatures for a file or, if given a directory,
// all files under that directory from the chunk map.
// fileFilter (if non-empty) restricts directory results to files whose path matches the regex.
// fileFilter is ignored when relFilePath points directly to a file.
func FileSkeleton(workspacePath, relFilePath, fileFilter string) (string, error) {
	if filepath.IsAbs(relFilePath) {
		if rel, err := filepath.Rel(workspacePath, relFilePath); err == nil {
			relFilePath = rel
		}
	}

	absPath := filepath.Join(workspacePath, relFilePath)
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		return fileSkeletonSingle(workspacePath, relFilePath)
	}

	// Directory: load all chunks and filter by prefix.
	allChunks, err := loadAllChunks(workspacePath)
	if err != nil {
		return "", err
	}

	fileRe, err := compileFileFilter(fileFilter)
	if err != nil {
		return "", fmt.Errorf("invalid --file pattern: %w", err)
	}

	prefix := ""
	if relFilePath != "." {
		prefix = strings.TrimSuffix(relFilePath, "/") + "/"
	}

	byFile := make(map[string][]models.Chunk)
	for _, chunk := range allChunks {
		if prefix == "" || strings.HasPrefix(chunk.File, prefix) {
			if matchesFileFilter(fileRe, chunk.File) {
				byFile[chunk.File] = append(byFile[chunk.File], chunk)
			}
		}
	}

	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	// Estimate output lines: 1 header + len(chunks) per file, plus blank lines between files.
	totalSymbols := 0
	for _, chunks := range byFile {
		totalSymbols += len(chunks)
	}
	estimated := totalSymbols + len(files) + max(len(files)-1, 0)
	if estimated > maxLines() {
		symbolsByFile := make(map[string]int, len(byFile))
		for f, chunks := range byFile {
			symbolsByFile[f] = len(chunks)
		}
		return skeletonOverflow(symbolsByFile, totalSymbols, len(files), estimated), nil
	}

	var sb strings.Builder
	for i, f := range files {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(f + ":\n")
		for _, chunk := range byFile[f] {
			sb.WriteString(fmt.Sprintf("  %d-%d: %s\n", chunk.Start, chunk.End, chunk.Sig))
		}
	}
	return sb.String(), nil
}

func fileSkeletonSingle(workspacePath, relFilePath string) (string, error) {
	fileChunks, err := loadChunksForFile(workspacePath, relFilePath)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, chunk := range fileChunks {
		sb.WriteString(fmt.Sprintf("%d-%d: %s\n", chunk.Start, chunk.End, chunk.Sig))
	}
	return sb.String(), nil
}

// ReadChunkBlock looks up the line number and returns the specific code block.
func ReadChunkBlock(workspacePath, relFilePath string, lineNum int) (string, error) {
	if filepath.IsAbs(relFilePath) {
		if rel, err := filepath.Rel(workspacePath, relFilePath); err == nil {
			relFilePath = rel
		}
	}
	fileChunks, err := loadChunksForFile(workspacePath, relFilePath)
	if err != nil {
		return "", err
	}

	var matchedChunk *models.Chunk
	for _, chunk := range fileChunks {
		if lineNum >= chunk.Start && lineNum <= chunk.End {
			matchedChunk = &chunk
			break
		}
	}

	if matchedChunk == nil {
		return "", fmt.Errorf("no chunk found covering line %d", lineNum)
	}

	return ReadChunkBody(workspacePath, *matchedChunk, 1000)
}

// ReadFullFile reads the entire content of a file.
// If the file exceeds the safe line limit and unsafe is false, it returns a warning
// message instead of the content. Pass unsafe=true to force reading regardless.
func ReadFullFile(workspacePath, relFilePath string, unsafe bool) (string, error) {
	if filepath.IsAbs(relFilePath) {
		if rel, err := filepath.Rel(workspacePath, relFilePath); err == nil {
			relFilePath = rel
		}
	}

	absPath := filepath.Join(workspacePath, relFilePath)

	// Count lines first.
	{
		f, err := os.Open(absPath)
		if err != nil {
			return "", err
		}
		scanner := bufio.NewScanner(f)
		lineCount := 0
		for scanner.Scan() {
			lineCount++
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			return "", err
		}
		limit := maxLines()
		if lineCount > limit && !unsafe {
			return fmt.Sprintf(
				"File has %d lines (safe limit: %d).\n"+
					"Reading the whole file at this size is likely not what you want — "+
					"consider using your native file-reading tool instead.\n"+
					"If you still want the full content, pass --unsafe.",
				lineCount, limit,
			), nil
		}
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FindUsages uses `rg -w` to find usages, maps to the Chunk Map, and returns only caller signatures.
// fileFilter (if non-empty) restricts results to files whose path matches the regex.
func FindUsages(workspacePath, symbol, fileFilter string) ([]string, error) {
	fileRe, err := compileFileFilter(fileFilter)
	if err != nil {
		return nil, fmt.Errorf("invalid --file pattern: %w", err)
	}

	cmd := exec.Command("rg", "-w", "-n", symbol)
	cmd.Dir = workspacePath

	var out bytes.Buffer
	cmd.Stdout = &out
	// Ignore stderr

	if err := cmd.Run(); err != nil {
		return nil, nil // no matches
	}

	lines := strings.Split(out.String(), "\n")
	seenSigs := make(map[string]bool)
	var signatures []string

	sigsByFile := make(map[string]int)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		relPath := parts[0]
		if !matchesFileFilter(fileRe, relPath) {
			continue
		}
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
				sigKey := fmt.Sprintf("%s:%s", relPath, chunk.Sig)
				if !seenSigs[sigKey] {
					seenSigs[sigKey] = true
					signatures = append(signatures, sigKey)
					sigsByFile[relPath]++
				}
				break
			}
		}
	}

	if len(signatures) > maxLines() {
		return []string{usagesOverflow(sigsByFile, len(signatures))}, nil
	}

	return signatures, nil
}

// ExtractInterface returns imports/includes and exported declarations for a file.
func ExtractInterface(workspacePath, relFilePath string, language string) (string, error) {
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

	var sb strings.Builder
	scanner := bufio.NewScanner(file)

	// Create common regexes
	importGo := regexp.MustCompile(`^(import\s*\(?|"[^"]+")`)
	exportGo := regexp.MustCompile(`^(func|type)\s+[A-Z]`)

	importPy := regexp.MustCompile(`^(import|from)\s+`)
	exportPy := regexp.MustCompile(`^(def|class)\s+[^\_]`)

	importJs := regexp.MustCompile(`^(import\s+|export\s+)`)
	exportJs := regexp.MustCompile(`^export\s+`)

	importCpp := regexp.MustCompile(`^#include\s+`)

	markdownHeader := regexp.MustCompile(`^#{1,6}\s+`)

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		match := false
		switch language {
		case ".go":
			match = importGo.MatchString(text) || exportGo.MatchString(text)
		case ".py":
			match = importPy.MatchString(text) || exportPy.MatchString(text)
		case ".js", ".ts", ".jsx", ".tsx":
			match = importJs.MatchString(text) || exportJs.MatchString(text)
		case ".cpp", ".hpp", ".h", ".c":
			match = importCpp.MatchString(text) // simplifying C++ to just headers
		case ".md", ".mdx":
			match = markdownHeader.MatchString(text)
		default:
			match = true // print all if unsupported
		}

		if match {
			sb.WriteString(text + "\n")
		}
	}

	return sb.String(), nil
}
