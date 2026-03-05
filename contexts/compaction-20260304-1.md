# Session Compaction Summary

## User Intent

- Focus exclusively on v2 (cercle-lite) development — v1 is archived, do not touch
- Improve tooling quality: fix known bugs, add progressive disclosure, add new "slicing" commands
- Create human-readable documentation and automated tests for all functionality

## Contextual Work Summary

### Bug Fix: file-skeleton on directories

- `FileSkeleton` now accepts a directory path and returns skeletons of all files inside
- When output exceeds line threshold, returns a directory-level summary instead

### Progressive Disclosure (threshold system)

- New `internal/search/threshold.go`: `maxLines()` reads `CERCLE_MAX_LINES` env var (default 500)
- Three overflow summary builders: `lexicalOverflow`, `skeletonOverflow`, `usagesOverflow`
- `SearchLexical` refactored to two-phase (collect chunks first, estimate lines, then read bodies)
- `FileSkeleton` directory case and `FindUsages` also check threshold before returning full output
- Overflow messages are actionable: tell agent exactly which subdir to drill into

### New Commands (4)

- `search-signature <pattern>`: regex over chunk signatures only, zero file I/O, grouped by file
- `largest-chunks [n]`: all chunks sorted by line count desc, top N (default 10)
- `callees <file> <line>`: heuristic extraction of called functions from chunk body, annotates local ones with `→ file:start-end`
- `dependents <file>`: finds files importing this file via two-pass rg (keyword imports + bare quoted paths for Go multi-line imports), source files only, comments filtered

### Documentation

- `v2/REFERENCE.md` (new): human developer reference — every command with examples, workflows, progressive disclosure table, supported languages, index storage, known limitations
- README and REFERENCE both open with the "question → command" framing table
- SKILL.md updated with all new commands and progressive disclosure rule

### Tests

- `internal/chunker/chunker_test.go` (new): 19 unit tests covering all parsers (Go, Python, JS/TS, CSS, HTML, C++), sliding window fallback, unsupported extensions
- `internal/search/search_test.go` (new): 22 + 20 = 42 integration tests; `TestMain` indexes a shared Go+JS workspace; all rg-dependent tests skip gracefully if ripgrep absent
- Tests for all four new commands included in the integration suite

## Files Touched

### Core Search Logic

- **`internal/search/threshold.go`**: new — `maxLines()`, three overflow summary builders
- **`internal/search/lexical.go`**: two-phase search, threshold check, `chunkBodyMaxLines` const
- **`internal/search/search.go`**: `FileSkeleton` dir+threshold, `FindUsages` threshold, `Neighbors` (not merged — replaced by new commands)
- **`internal/search/signature.go`**: new — `SearchSignature`
- **`internal/search/rank.go`**: new — `LargestChunks`
- **`internal/search/callees.go`**: new — `Callees`, `sigBaseName`, `isBuiltin`
- **`internal/search/dependents.go`**: new — `Dependents`, two-pass rg, comment-line filter

### CLI

- **`main.go`**: four new command cases + updated `printUsage`

### Tests

- **`internal/chunker/chunker_test.go`**: new — 19 unit tests
- **`internal/search/search_test.go`**: new — 42 integration tests, `TestMain` with shared workspace (hello.go + utils.js + main.js)

### Documentation

- **`v2/REFERENCE.md`**: new — full human developer reference
- **`v2/README.md`**: usage section replaced with question→command table
- **`v2/skills/SKILL.md`**: new commands added, progressive disclosure rule added
- **`v2/next.md`**: issues 1 and 3 marked resolved

### Memory

- **`~/.claude/projects/-Users-ruben-code-cercle/memory/MEMORY.md`**: created — project overview, v2 architecture, pending work
