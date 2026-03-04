package search

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Dependents finds files that import or reference relFilePath.
// Answers: "what imports this?" — the reverse of extract-interface.
func Dependents(workspacePath, relFilePath string) ([]string, error) {
	if filepath.IsAbs(relFilePath) {
		if rel, err := filepath.Rel(workspacePath, relFilePath); err == nil {
			relFilePath = rel
		}
	}

	// Use the stem (name without extension) and the parent directory name as
	// search terms: either could appear in an import path.
	stem := strings.TrimSuffix(filepath.Base(relFilePath), filepath.Ext(relFilePath))
	dir := filepath.Dir(relFilePath)
	if dir == "." {
		dir = ""
	}

	// Build two complementary patterns:
	//   1. explicit: import/require/from/include keyword followed by the stem
	//   2. implicit: a bare quoted path containing the stem (Go multi-line imports,
	//      e.g. `"garbell/internal/models"` with no keyword on the same line)
	termRE := strings.Join(func() []string {
		seen := make(map[string]bool)
		var out []string
		for _, t := range []string{stem, filepath.Base(dir)} {
			if t != "" && !seen[t] {
				seen[t] = true
				out = append(out, regexp.QuoteMeta(t))
			}
		}
		return out
	}(), "|")

	patternKeyword := fmt.Sprintf(`(?i)(import|require|from|include)\b.*(%s)`, termRE)
	patternQuoted := fmt.Sprintf(`"[^"]*(%s)[^"]*"`, termRE)

	runRG := func(pattern string) ([]byte, error) {
		cmd := exec.Command("rg", "-n", "-e", pattern,
			"--glob=*.{go,py,js,ts,jsx,tsx,c,cpp,cc,h,hpp,css,html,htm}",
		)
		cmd.Dir = workspacePath
		var buf bytes.Buffer
		cmd.Stdout = &buf
		_ = cmd.Run() // non-zero exit = no matches, not an error
		return buf.Bytes(), nil
	}

	out1, _ := runRG(patternKeyword)
	out2, _ := runRG(patternQuoted)
	combined := string(out1) + string(out2)

	self := relFilePath
	seen := make(map[string]bool)
	var results []string

	for _, line := range strings.Split(combined, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		file, lineNo, content := parts[0], parts[1], strings.TrimSpace(parts[2])

		if file == self {
			continue
		}
		// Skip comment-only lines (Go //, Python #, C /* or block comment continuation *).
		if strings.HasPrefix(content, "//") || strings.HasPrefix(content, "#") ||
			strings.HasPrefix(content, "*") || strings.HasPrefix(content, "/*") {
			continue
		}
		key := file + ":" + lineNo
		if !seen[key] {
			seen[key] = true
			results = append(results, fmt.Sprintf("%s:%s: %s", file, lineNo, content))
		}
	}

	return results, nil
}
