package search

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const defaultMaxLines = 500

// maxLines returns the configured line threshold, overridable via GARBELL_MAX_LINES.
func maxLines() int {
	if v := os.Getenv("GARBELL_MAX_LINES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultMaxLines
}

// lexicalOverflow builds a directory-grouped summary when search-lexical would exceed the
// line threshold. chunksByFile maps relFilePath -> chunk count for that file.
func lexicalOverflow(chunksByFile map[string]int, totalChunks, estimatedLines int) string {
	type dirInfo struct {
		files  []string
		chunks int
	}
	dirs := make(map[string]*dirInfo)
	for file, n := range chunksByFile {
		dir := filepath.Dir(file)
		if dirs[dir] == nil {
			dirs[dir] = &dirInfo{}
		}
		dirs[dir].files = append(dirs[dir].files, filepath.Base(file))
		dirs[dir].chunks += n
	}

	type entry struct {
		dir  string
		info *dirInfo
	}
	entries := make([]entry, 0, len(dirs))
	for d, info := range dirs {
		entries = append(entries, entry{d, info})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].info.chunks != entries[j].info.chunks {
			return entries[i].info.chunks > entries[j].info.chunks
		}
		return entries[i].dir < entries[j].dir
	})

	var sb strings.Builder
	fmt.Fprintf(&sb, "Results exceed %d lines (~%d chunks across %d files). Drill down by location:\n\n",
		estimatedLines, totalChunks, len(chunksByFile))
	for _, e := range entries {
		sort.Strings(e.info.files)
		label := e.dir + "/"
		fmt.Fprintf(&sb, "  %-36s %d chunk(s)  [%s]\n",
			label, e.info.chunks, strings.Join(e.info.files, ", "))
	}
	sb.WriteString("\nRefine your query, add a path, or use `file-skeleton <dir>` to explore.")
	return sb.String()
}

// skeletonOverflow builds a directory-grouped summary when file-skeleton would exceed the
// line threshold. byFile maps relFilePath -> symbol count for that file.
func skeletonOverflow(byFile map[string]int, totalSymbols, totalFiles, estimatedLines int) string {
	type dirInfo struct {
		fileCount   int
		symbolCount int
	}
	dirs := make(map[string]*dirInfo)
	for file, n := range byFile {
		dir := filepath.Dir(file)
		if dirs[dir] == nil {
			dirs[dir] = &dirInfo{}
		}
		dirs[dir].fileCount++
		dirs[dir].symbolCount += n
	}

	type entry struct {
		dir  string
		info *dirInfo
	}
	entries := make([]entry, 0, len(dirs))
	for d, info := range dirs {
		entries = append(entries, entry{d, info})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].info.symbolCount != entries[j].info.symbolCount {
			return entries[i].info.symbolCount > entries[j].info.symbolCount
		}
		return entries[i].dir < entries[j].dir
	})

	var sb strings.Builder
	fmt.Fprintf(&sb, "Output exceeds %d lines (%d symbols across %d files). Directory summary:\n\n",
		estimatedLines, totalSymbols, totalFiles)
	for _, e := range entries {
		fmt.Fprintf(&sb, "  %-36s %d file(s)   %d symbol(s)\n",
			e.dir+"/", e.info.fileCount, e.info.symbolCount)
	}
	sb.WriteString("\nUse `file-skeleton <subdir>` to drill down.")
	return sb.String()
}

// usagesOverflow builds a directory-grouped summary when find-usages would exceed the
// line threshold. sigsByFile maps relFilePath -> caller signature count.
func usagesOverflow(sigsByFile map[string]int, totalSigs int) string {
	type dirInfo struct {
		files []string
		count int
	}
	dirs := make(map[string]*dirInfo)
	for file, n := range sigsByFile {
		dir := filepath.Dir(file)
		if dirs[dir] == nil {
			dirs[dir] = &dirInfo{}
		}
		dirs[dir].files = append(dirs[dir].files, filepath.Base(file))
		dirs[dir].count += n
	}

	type entry struct {
		dir  string
		info *dirInfo
	}
	entries := make([]entry, 0, len(dirs))
	for d, info := range dirs {
		entries = append(entries, entry{d, info})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].info.count != entries[j].info.count {
			return entries[i].info.count > entries[j].info.count
		}
		return entries[i].dir < entries[j].dir
	})

	var sb strings.Builder
	fmt.Fprintf(&sb, "Too many usages (%d callers across %d files). Summary by location:\n\n",
		totalSigs, len(sigsByFile))
	for _, e := range entries {
		sort.Strings(e.info.files)
		fmt.Fprintf(&sb, "  %-36s %d caller(s)  [%s]\n",
			e.dir+"/", e.info.count, strings.Join(e.info.files, ", "))
	}
	sb.WriteString("\nUse `find-usages` with a more specific symbol, or `search-lexical` to inspect a location.")
	return sb.String()
}
