package indexer

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"garbell/internal/chunker"
	"garbell/internal/models"
)

// SupportedExtensions defines the file extensions the chunker currently understands.
var SupportedExtensions = map[string]bool{
	".go":   true,
	".py":   true,
	".js":   true,
	".ts":   true,
	".jsx":  true,
	".tsx":  true,
	".c":    true,
	".cpp":  true,
	".cc":   true,
	".cxx":  true,
	".h":    true,
	".hpp":  true,
	".css":   true,
	".html":  true,
	".htm":   true,
	".md":    true,
	".mdx":   true,
	".proto": true,
}

// GenerateIndex traverses the given directory using rg, parses the supported
// source files, and writes the chunk maps into the global ~/.garbell/indexes/ location.
func GenerateIndex(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	files, err := discoverFiles(absDir)
	if err != nil {
		return fmt.Errorf("failed to discover files: %w", err)
	}

	fmt.Printf("Parsed %d files to index...\n", len(files))

	// Map of Shard ID (00-ff) to list of chunks
	shardMap := make(map[string][]models.Chunk)
	var mapMutex sync.Mutex
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrency
	sem := make(chan struct{}, 10)

	for _, file := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(f string) {
			defer wg.Done()
			defer func() { <-sem }()

			chunks, err := chunker.ParseFile(f)
			if err != nil || len(chunks) == 0 {
				return
			}

			// Calculate shard ID
			// Shard ID is first 2 chars of md5(relative_path)
			rel, err := filepath.Rel(absDir, f)
			if err != nil {
				rel = f
			}

			// Update chunks to use relative path
			for i := range chunks {
				chunks[i].File = rel
			}

			shardID := GetShardID(rel)

			mapMutex.Lock()
			shardMap[shardID] = append(shardMap[shardID], chunks...)
			mapMutex.Unlock()
		}(file)
	}

	wg.Wait()

	// Write shards to disk
	return writeShards(absDir, shardMap)
}

// GetShardID computes the first two characters of the md5 hash of a relative filepath
func GetShardID(relativePath string) string {
	hash := md5.Sum([]byte(relativePath))
	hexString := hex.EncodeToString(hash[:])
	return hexString[:2]
}

func discoverFiles(dir string) ([]string, error) {
	cmd := exec.Command("rg", "--files")
	cmd.Dir = dir

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If rg fails (e.g. no files found), return empty
		return nil, nil
	}

	lines := strings.Split(out.String(), "\n")
	var validFiles []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(line))
		if SupportedExtensions[ext] {
			validFiles = append(validFiles, filepath.Join(dir, line))
		}
	}

	return validFiles, nil
}

func writeShards(absDir string, shardMap map[string][]models.Chunk) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dirHashBytes := md5.Sum([]byte(absDir))
	dirHash := hex.EncodeToString(dirHashBytes[:])

	indexPath := filepath.Join(home, ".garbell", "indexes", dirHash)
	if err := os.MkdirAll(indexPath, 0755); err != nil {
		return err
	}

	// Write metadata.json
	metadataPath := filepath.Join(indexPath, "metadata.json")
	metadata := map[string]string{
		"path":       absDir,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	mData, _ := json.MarshalIndent(metadata, "", "  ")
	os.WriteFile(metadataPath, mData, 0644)

	// Write each active shard
	for shardID, chunks := range shardMap {
		shardPath := filepath.Join(indexPath, shardID+".json")
		data, err := json.Marshal(chunks)
		if err != nil {
			continue
		}
		os.WriteFile(shardPath, data, 0644)
	}

	fmt.Printf("Successfully wrote %d shards to %s\n", len(shardMap), indexPath)
	return nil
}
