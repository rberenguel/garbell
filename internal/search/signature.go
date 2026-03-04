package search

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// SearchSignature searches chunk signatures (not file bodies) for matches against a regex.
// Answers: "find all functions/types with this shape" — no source file I/O required.
func SearchSignature(workspacePath, pattern string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid pattern: %w", err)
	}

	allChunks, err := loadAllChunks(workspacePath)
	if err != nil {
		return "", err
	}

	type match struct {
		file  string
		start int
		end   int
		sig   string
	}
	var matches []match
	for _, c := range allChunks {
		if re.MatchString(c.Sig) {
			matches = append(matches, match{c.File, c.Start, c.End, c.Sig})
		}
	}
	if len(matches) == 0 {
		return "", nil
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].file != matches[j].file {
			return matches[i].file < matches[j].file
		}
		return matches[i].start < matches[j].start
	})

	// Overflow: too many matches — return directory-grouped summary.
	if len(matches) > maxLines() {
		chunksByFile := make(map[string]int)
		for _, m := range matches {
			chunksByFile[m.file]++
		}
		return lexicalOverflow(chunksByFile, len(matches), len(matches)), nil
	}

	// Group output by file.
	var sb strings.Builder
	prevFile := ""
	for _, m := range matches {
		if m.file != prevFile {
			if prevFile != "" {
				sb.WriteString("\n")
			}
			sb.WriteString(m.file + ":\n")
			prevFile = m.file
		}
		sb.WriteString(fmt.Sprintf("  %d-%d: %s\n", m.start, m.end, m.sig))
	}
	return sb.String(), nil
}
