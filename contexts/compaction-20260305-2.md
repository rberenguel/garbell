# Session Compaction Summary

## User Intent

- Fix `search-related` producing walls of text by violating progressive disclosure
- Align all documentation (README, REFERENCE, SKILL.md, inline help) with actual behaviour
- Add a VERSION file and badge

## Contextual Work Summary

### Progressive Disclosure Fix — `search-related`

- Root cause: `SearchRelated` built a broad OR-regex from original tokens + PPMI synonyms and delegated to `SearchLexical`, hitting extremely common terms and flooding output
- Fix: two-pass scoring (original query = score 2, synonym-only = score 1), results merged with originals first, hard cap at 50 chunks
- `maxRelatedChunks = 50`, configurable via the same pattern as other thresholds

### Progressive Disclosure Fix — `search-lexical`

- Root cause identified as architectural: returning full chunk bodies (up to 100 lines each) is wrong by default — even 5–20 chunks can be 500–2000 lines
- Fix: `SearchLexical` now returns a **compact grouped-by-file listing** (sig + line range, one line per chunk) instead of full bodies
- Overflow threshold changed from line-count (`GARBELL_MAX_LINES`) to chunk-count (`GARBELL_MAX_SUMMARY_CHUNKS`, default 80)
- `read-chunk` is now the explicit drill-down step for both search commands

### Refactoring

- Extracted `collectMatchingChunks` from `SearchLexical` into a shared helper (used by both search functions)
- Added `formatChunkSummary` shared helper producing the grouped-by-file compact format
- `SearchLexical` simplified from ~30 to ~15 lines; body-reading loop removed from both search functions
- `chunkBodyMaxLines` const retained (still used by `ReadChunkBody` / `read-chunk`)

### Markdown Exclusion Clarification

- Confirmed: `.md`/`.html` files are excluded from the **PPMI thesaurus builder** only; they remain fully indexed and searchable — intentional, no change made

### Documentation Alignment

- All four surfaces updated to reflect compact-listing output and the `read-chunk` drill-down pattern:
  - `README.md`: step 3 of progressive disclosure description
  - `REFERENCE.md`: search-lexical section (format, example, threshold); search-related step 3 (two-pass, not OR-regex); threshold table and env-var examples
  - `skills/SKILL.md`: search-lexical and search-related descriptions; Rule 3; read-chunk description
  - `cmd/garbell/main.go`: usage line for search-lexical
  - `internal/tui/tui.go`: inline help for `sl`

### Versioning

- Added `VERSION` file at repo root: `0.0.3`
- Added version badge to `README.md` alongside existing status badge

## Files Touched

### Core Logic

- **`internal/search/lexical.go`**: extracted `collectMatchingChunks`; added `formatChunkSummary` and `maxSummaryChunks()`; `SearchLexical` now returns compact listing
- **`internal/search/semantic.go`**: `SearchRelated` rewritten with two-pass scoring; compact output via `formatChunkSummary`; cap raised to 50

### Tests

- **`internal/search/search_test.go`**: updated `TestSearchLexical_ResultIncludesHeader` and `TestSearchLexical_OverflowSummary` for new format and `GARBELL_MAX_SUMMARY_CHUNKS` env var

### Documentation

- **`README.md`**: step 3 description + version badge
- **`REFERENCE.md`**: search-lexical section, search-related step 3, overflow table, threshold env-var examples
- **`skills/SKILL.md`**: search-lexical, search-related, read-chunk descriptions; Rule 3
- **`cmd/garbell/main.go`**: search-lexical usage line
- **`internal/tui/tui.go`**: `sl` inline help text

### New Files

- **`VERSION`**: `0.0.3`
- **`contexts/compaction-20260305-2.md`**: this file
