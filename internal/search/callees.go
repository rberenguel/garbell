package search

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// callPattern matches common call-site shapes across Go, Python, JS, C/C++.
// Captures the callee name from expressions like: foo(...), self.foo(...), obj.foo(...)
var callPattern = regexp.MustCompile(`\b([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)\s*\(`)

// Callees returns the distinct function/method names called within the chunk
// enclosing lineNum in relFilePath. Answers: "what does this function call?"
func Callees(workspacePath, relFilePath string, lineNum int) ([]string, error) {
	body, err := ReadChunkBlock(workspacePath, relFilePath, lineNum)
	if err != nil {
		return nil, err
	}

	// Collect candidate callee names from the body, excluding the first line
	// (which is the function signature itself).
	lines := strings.Split(body, "\n")
	if len(lines) > 1 {
		lines = lines[1:] // skip signature line
	}

	seen := make(map[string]bool)
	var callees []string
	for _, line := range lines {
		matches := callPattern.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			name := m[1]
			// Skip very common keywords/builtins that look like calls.
			if isBuiltin(name) {
				continue
			}
			if !seen[name] {
				seen[name] = true
				callees = append(callees, name)
			}
		}
	}

	sort.Strings(callees)

	// Cross-reference against indexed signatures to annotate which callees
	// are defined in this codebase (vs. stdlib/external).
	allChunks, err := loadAllChunks(workspacePath)
	if err != nil {
		// Non-fatal: return raw callee names without annotation.
		return callees, nil
	}

	// Build a map: callee name → defining file + line range.
	type def struct {
		file  string
		start int
		end   int
	}
	defined := make(map[string]def)
	for _, c := range allChunks {
		// Extract just the base name from the signature (first word after func/def/function/class).
		name := sigBaseName(c.Sig)
		if name != "" {
			defined[name] = def{c.File, c.Start, c.End}
		}
	}

	var results []string
	for _, callee := range callees {
		// Strip any receiver prefix for lookup (e.g. "obj.Method" → "Method").
		base := callee
		if idx := strings.LastIndex(callee, "."); idx >= 0 {
			base = callee[idx+1:]
		}
		if d, ok := defined[base]; ok {
			results = append(results, fmt.Sprintf("%s  →  %s:%d-%d", callee, d.file, d.start, d.end))
		} else {
			results = append(results, callee)
		}
	}
	return results, nil
}

// sigBaseName extracts the bare function/method/class name from a signature string.
func sigBaseName(sig string) string {
	// Go:   "func FuncName(" or "func (r Recv) MethodName("
	// Py:   "def func_name("  "class ClassName:"
	// JS:   "function funcName(" "class ClassName"
	// C++:  "RetType funcName("
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^func\s+\([^)]+\)\s+(\w+)\s*\(`), // Go method
		regexp.MustCompile(`^func\s+(\w+)\s*\(`),             // Go function
		regexp.MustCompile(`^(?:async\s+)?def\s+(\w+)\s*\(`), // Python
		regexp.MustCompile(`^(?:async\s+)?function\s+(\w+)`), // JS function
		regexp.MustCompile(`^class\s+(\w+)`),                 // JS/Go class/type
		regexp.MustCompile(`^type\s+(\w+)`),                  // Go type
		regexp.MustCompile(`^\w[\w\s*:<>]+\s+(\w+)\s*\(`),   // C++ function
	}
	for _, re := range patterns {
		if m := re.FindStringSubmatch(sig); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

// isBuiltin returns true for common language keywords/builtins that pattern-match as calls.
var builtins = map[string]bool{
	// Go
	"if": true, "for": true, "switch": true, "select": true,
	"make": true, "new": true, "append": true, "len": true, "cap": true,
	"delete": true, "copy": true, "close": true, "panic": true, "recover": true,
	"print": true, "println": true,
	// Python
	"range": true, "int": true, "str": true,
	"list": true, "dict": true, "set": true, "tuple": true, "type": true,
	"isinstance": true, "hasattr": true, "getattr": true, "setattr": true,
	"super": true, "staticmethod": true, "classmethod": true, "property": true,
	// JS
	"setTimeout": true, "setInterval": true,
	"parseInt": true, "parseFloat": true, "isNaN": true,
	// C
	"sizeof": true, "printf": true, "malloc": true, "free": true,
}

func isBuiltin(name string) bool {
	return builtins[name] || builtins[strings.ToLower(name)]
}
