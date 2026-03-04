package chunker

import (
	"regexp"
	"strings"

	"garbell/internal/models"
)

type goParser struct {
	inChunk    bool
	startLine  int
	braceCount int
	sig        string
}

var (
	goFuncRegex = regexp.MustCompile(`^func\s+`)
	goTypeRegex = regexp.MustCompile(`^type\s+`)
)

func (p *goParser) processLine(filePath string, lineNum int, lineText string) []models.Chunk {
	if !p.inChunk {
		if goFuncRegex.MatchString(lineText) || goTypeRegex.MatchString(lineText) {
			p.inChunk = true
			p.startLine = lineNum
			p.braceCount = 0
			p.sig = strings.TrimSpace(lineText) // very simple signature
		}
	}

	if p.inChunk {
		p.braceCount += strings.Count(lineText, "{")
		p.braceCount -= strings.Count(lineText, "}")

		if p.braceCount == 0 && strings.Contains(lineText, "}") {
			p.inChunk = false
			return []models.Chunk{{
				File:  filePath,
				Start: p.startLine,
				End:   lineNum,
				Sig:   p.sig,
			}}
		}
	}

	return nil
}

func (p *goParser) finalize(filePath string, totalLines int) []models.Chunk {
	return nil // typically missing a closing brace means syntax error
}

type pythonParser struct {
	inChunk     bool
	startLine   int
	startIndent int
	sig         string
}

var pyDefClassRegex = regexp.MustCompile(`^(?:async\s+)?(?:def|class)\s+`)

func (p *pythonParser) processLine(filePath string, lineNum int, lineText string) []models.Chunk {
	trimmed := strings.TrimSpace(lineText)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}

	indent := len(lineText) - len(strings.TrimLeft(lineText, " \t"))

	if !p.inChunk {
		if pyDefClassRegex.MatchString(trimmed) {
			p.inChunk = true
			p.startLine = lineNum
			p.startIndent = indent
			p.sig = trimmed
		}
		return nil
	}

	if p.inChunk {
		if indent <= p.startIndent && !pyDefClassRegex.MatchString(trimmed) {
			// Found the end of the Python block (different block starting at same/lower indent)
			res := []models.Chunk{{
				File:  filePath,
				Start: p.startLine,
				End:   lineNum - 1,
				Sig:   p.sig,
			}}

			// We returned to same/lower indent, check if this new line starts a NEW block
			if pyDefClassRegex.MatchString(trimmed) {
				p.inChunk = true
				p.startLine = lineNum
				p.startIndent = indent
				p.sig = trimmed
			} else {
				p.inChunk = false
			}

			return res
		}
	}
	return nil
}

func (p *pythonParser) finalize(filePath string, totalLines int) []models.Chunk {
	if p.inChunk {
		p.inChunk = false
		return []models.Chunk{{
			File:  filePath,
			Start: p.startLine,
			End:   totalLines,
			Sig:   p.sig,
		}}
	}
	return nil
}

type jsChunkInfo struct {
	startLine  int
	braceCount int
	sig        string
}

type jsParser struct {
	stack      []jsChunkInfo
	braceCount int
}

var (
	jsFuncRegex   = regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+\w+`)
	jsConstRegex  = regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+\w+\s*=\s*(?:async\s+)?(?:\(.*?\)|\w+)\s*=>`)
	jsClassRegex  = regexp.MustCompile(`^\s*(?:export\s+)?class\s+\w+`)
	jsMethodRegex = regexp.MustCompile(`^\s*(?:async\s+)?\w+\s*\([^)]*\)\s*\{`)
)

func (p *jsParser) processLine(filePath string, lineNum int, lineText string) []models.Chunk {
	var completed []models.Chunk
	trimmed := strings.TrimSpace(lineText)

	if jsFuncRegex.MatchString(trimmed) || jsConstRegex.MatchString(trimmed) || jsClassRegex.MatchString(trimmed) || jsMethodRegex.MatchString(trimmed) {
		sig := strings.TrimSuffix(trimmed, "{")
		sig = strings.TrimSpace(sig)

		p.stack = append(p.stack, jsChunkInfo{
			startLine:  lineNum,
			braceCount: p.braceCount,
			sig:        sig,
		})
	}

	p.braceCount += strings.Count(lineText, "{")
	p.braceCount -= strings.Count(lineText, "}")

	if p.braceCount >= 0 && len(p.stack) > 0 {
		var active []jsChunkInfo
		for _, info := range p.stack {
			if p.braceCount <= info.braceCount && strings.Contains(lineText, "}") {
				completed = append(completed, models.Chunk{
					File:  filePath,
					Start: info.startLine,
					End:   lineNum,
					Sig:   info.sig,
				})
			} else {
				active = append(active, info)
			}
		}
		p.stack = active
	}

	return completed
}

func (p *jsParser) finalize(filePath string, totalLines int) []models.Chunk {
	return nil
}

type cppParser struct {
	inChunk    bool
	startLine  int
	braceCount int
	sig        string
}

var (
	cppFuncRegex  = regexp.MustCompile(`^[\w:<>]+\s+[\w:~]+\s*\(.*?\)(?:\s*const)?\s*\{`)
	cppClassRegex = regexp.MustCompile(`^(?:class|struct)\s+\w+`)
)

func (p *cppParser) processLine(filePath string, lineNum int, lineText string) []models.Chunk {
	trimmed := strings.TrimSpace(lineText)

	if !p.inChunk {
		if cppFuncRegex.MatchString(trimmed) || cppClassRegex.MatchString(trimmed) {
			p.inChunk = true
			p.startLine = lineNum
			p.braceCount = 0
			// Remove the opening brace from the signature if it exists
			if cppFuncRegex.MatchString(trimmed) {
				p.sig = strings.TrimSuffix(trimmed, "{")
			} else {
				p.sig = trimmed
			}
			p.sig = strings.TrimSpace(p.sig)
		}
	}

	if p.inChunk {
		p.braceCount += strings.Count(lineText, "{")
		p.braceCount -= strings.Count(lineText, "}")

		if p.braceCount == 0 && strings.Contains(lineText, "}") {
			p.inChunk = false
			return []models.Chunk{{
				File:  filePath,
				Start: p.startLine,
				End:   lineNum,
				Sig:   p.sig,
			}}
		}
	}
	return nil
}

func (p *cppParser) finalize(filePath string, totalLines int) []models.Chunk {
	return nil
}

type cssParser struct {
	inChunk    bool
	startLine  int
	braceCount int
	sig        string
}

var cssSelectorRegex = regexp.MustCompile(`^[\.#\w][^{]+{\s*$`)

func (p *cssParser) processLine(filePath string, lineNum int, lineText string) []models.Chunk {
	trimmed := strings.TrimSpace(lineText)
	if !p.inChunk {
		if cssSelectorRegex.MatchString(trimmed) {
			p.inChunk = true
			p.startLine = lineNum
			p.braceCount = 0
			p.sig = strings.TrimSuffix(trimmed, "{")
			p.sig = strings.TrimSpace(p.sig)
		}
	}

	if p.inChunk {
		p.braceCount += strings.Count(lineText, "{")
		p.braceCount -= strings.Count(lineText, "}")

		if p.braceCount <= 0 && strings.Contains(lineText, "}") {
			p.inChunk = false
			return []models.Chunk{{
				File:  filePath,
				Start: p.startLine,
				End:   lineNum,
				Sig:   p.sig,
			}}
		}
	}

	return nil
}

func (p *cssParser) finalize(filePath string, totalLines int) []models.Chunk {
	return nil
}

type htmlParser struct {
	inChunk   bool
	startLine int
	tagName   string
	sig       string
}

var htmlOuterTagRegex = regexp.MustCompile(`(?i)<(script|style|main|div\s+id=.*?)>`)

func (p *htmlParser) processLine(filePath string, lineNum int, lineText string) []models.Chunk {
	// A simple tag parser that looks for closing tags
	if !p.inChunk {
		matches := htmlOuterTagRegex.FindStringSubmatch(lineText)
		if len(matches) > 1 {
			p.inChunk = true
			p.startLine = lineNum
			p.sig = matches[0]

			// Determine the closing tag we are looking for
			switch strings.ToLower(matches[1]) {
			case "script":
				p.tagName = "</script>"
			case "style":
				p.tagName = "</style>"
			case "main":
				p.tagName = "</main>"
			default:
				if strings.HasPrefix(strings.ToLower(matches[1]), "div") {
					p.tagName = "</div>"
				}
			}
		}
	}

	if p.inChunk {
		if p.tagName != "" && strings.Contains(strings.ToLower(lineText), p.tagName) {
			p.inChunk = false
			return []models.Chunk{{
				File:  filePath,
				Start: p.startLine,
				End:   lineNum,
				Sig:   p.sig,
			}}
		}
	}

	// Wait, sliding window fallback is required if we fail to parse chunks.
	// For simplicity, returning structural chunks here. Sliding window can be evaluated in standard indexer.
	return nil
}

func (p *htmlParser) finalize(filePath string, totalLines int) []models.Chunk {
	// Close any open tags if reach EOF
	if p.inChunk {
		return []models.Chunk{{
			File:  filePath,
			Start: p.startLine,
			End:   totalLines,
			Sig:   p.sig,
		}}
	}
	return nil
}

// markdownParser chunks by ATX headings (# through ######).
// Each heading starts a new chunk spanning until the next heading at any level.
type markdownParser struct {
	inChunk   bool
	startLine int
	sig       string
}

var mdHeadingRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)`)

func (p *markdownParser) processLine(filePath string, lineNum int, lineText string) []models.Chunk {
	if !mdHeadingRegex.MatchString(lineText) {
		return nil
	}
	var result []models.Chunk
	if p.inChunk {
		result = append(result, models.Chunk{
			File:  filePath,
			Start: p.startLine,
			End:   lineNum - 1,
			Sig:   p.sig,
		})
	}
	p.inChunk = true
	p.startLine = lineNum
	p.sig = strings.TrimSpace(lineText)
	return result
}

func (p *markdownParser) finalize(filePath string, totalLines int) []models.Chunk {
	if p.inChunk {
		return []models.Chunk{{
			File:  filePath,
			Start: p.startLine,
			End:   totalLines,
			Sig:   p.sig,
		}}
	}
	return nil
}

// protoParser chunks message, service, and enum blocks in Protocol Buffer files.
type protoParser struct {
	inChunk    bool
	startLine  int
	braceCount int
	sig        string
}

var protoBlockRegex = regexp.MustCompile(`^(?:message|service|enum)\s+\w+`)

func (p *protoParser) processLine(filePath string, lineNum int, lineText string) []models.Chunk {
	trimmed := strings.TrimSpace(lineText)

	if !p.inChunk {
		if protoBlockRegex.MatchString(trimmed) {
			p.inChunk = true
			p.startLine = lineNum
			p.braceCount = 0
			p.sig = trimmed
		}
	}

	if p.inChunk {
		p.braceCount += strings.Count(lineText, "{")
		p.braceCount -= strings.Count(lineText, "}")

		if p.braceCount == 0 && strings.Contains(lineText, "}") {
			p.inChunk = false
			return []models.Chunk{{
				File:  filePath,
				Start: p.startLine,
				End:   lineNum,
				Sig:   p.sig,
			}}
		}
	}
	return nil
}

func (p *protoParser) finalize(filePath string, totalLines int) []models.Chunk {
	return nil
}
