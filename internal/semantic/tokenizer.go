package semantic

import (
	"strings"
	"unicode"
)

var stopWords = map[string]bool{
	// Common English
	"the": true, "and": true, "for": true, "not": true, "are": true,
	"with": true, "that": true, "this": true, "from": true, "have": true,
	"but": true, "was": true, "his": true, "her": true, "they": true,
	"you": true, "all": true, "can": true, "had": true, "one": true,
	"its": true, "out": true, "has": true, "use": true, "our": true,
	// Go/programming keywords
	"return": true, "func": true, "class": true, "import": true, "export": true,
	"var":      true, "let": true, "const": true, "type": true, "new": true,
	"nil":      true, "null": true, "true": true, "false": true, "else": true,
	"case":     true, "break": true, "continue": true, "switch": true, "goto": true,
	"def":      true, "pass": true, "self": true, "super": true, "void": true,
	"int":      true, "str": true, "bool": true, "uint": true, "byte": true,
	"interface": true, "struct": true, "package": true, "range": true, "make": true,
	"append": true, "error": true, "string": true, "map": true, "chan": true,
	"defer": true, "select": true, "fallthrough": true, "default": true,
}

// Tokenize splits text into lowercase, code-aware tokens.
// It splits on non-alphanumeric characters and camelCase boundaries,
// filters tokens shorter than 3 or longer than 30 characters, and
// removes common stop words.
func Tokenize(text string) []string {
	rawWords := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var tokens []string
	for _, word := range rawWords {
		for _, part := range splitCamel(word) {
			p := strings.ToLower(part)
			if len(p) < 3 || len(p) > 30 {
				continue
			}
			if stopWords[p] {
				continue
			}
			tokens = append(tokens, p)
		}
	}
	return tokens
}

// splitCamel splits a camelCase or PascalCase word into its constituent parts.
func splitCamel(word string) []string {
	if len(word) == 0 {
		return nil
	}
	runes := []rune(word)
	var parts []string
	start := 0

	for i := 1; i < len(runes); i++ {
		prev := runes[i-1]
		curr := runes[i]

		// Transition from lower/digit to upper: splitCamel → split, Camel
		if unicode.IsUpper(curr) && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
			parts = append(parts, string(runes[start:i]))
			start = i
			continue
		}
		// Transition from a run of uppers to upper+lower: HTMLParser → HTML, Parser
		if unicode.IsUpper(curr) && unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
			parts = append(parts, string(runes[start:i]))
			start = i
		}
	}
	parts = append(parts, string(runes[start:]))
	return parts
}
