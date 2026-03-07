# Session Compaction Summary

## User Intent

- Add three new features: `--file <regex>` filter for search commands, `read-chunk -1 [--unsafe]` for full-file reading, and `peek` for line-context inspection
- Keep all new flags order-agnostic and document them thoroughly in skill, README, REFERENCE, and REPL help
- Bump version to 0.0.4

## Contextual Work Summary

### `--file <regex>` Filter (Feature 1)

- Adds a file path regex filter to: `search-lexical`, `search-related`, `find-usages`, `file-skeleton` (directory mode), `search-signature`, `largest-chunks`
- New `internal/search/filter.go`: `compileFileFilter` / `matchesFileFilter` helpers
- All affected search functions gained a `fileFilter string` parameter
- CLI and REPL use a manual order-agnostic flag extraction (`extractFlag` / `tuiExtractFlag`) instead of `flag.FlagSet` — the latter stops at the first non-flag positional arg, which breaks the common `command query --file pattern` usage
- `collectMatchingChunks` filters rg output by file path before chunk lookup; index-only commands (signature, rank) filter after loading chunks

### `read-chunk -1 [--unsafe]` (Feature 2)

- New `ReadFullFile(workspacePath, relFilePath string, unsafe bool)` in `search.go`
- Counts lines first; returns a warning message if over `GARBELL_MAX_LINES` (default 500) and `unsafe` is false
- `-1` as line number is a natural sentinel (`strconv.Atoi` handles it); dispatched in `main.go` and `tui.go` before calling `ReadChunkBlock`
- `--unsafe` extracted via `extractBoolFlag` / `tuiExtractBoolFlag`

### `peek` (Feature 3)

- New command and REPL alias `pk`; new `internal/search/peek.go` with `Peek()`
- Reads directly from source file, no index required; outputs `radius` lines around target with `>>` marker
- Default radius 5; accepts optional 3rd positional arg
- Added to REPL tab-completion as a path command

### Tests

- Added tests for all three features across `search_test.go`: file filter (matching, no-match, invalid regex) for lexical/find-usages/signature/rank/skeleton; `ReadFullFile` safe/unsafe/over-limit; peek header, marker, radius, clamping; `SearchRelated` invalid-filter error

### Documentation

- `README.md`: `peek` added to commands table; version badge bumped to 0.0.4
- `REFERENCE.md`: commands table updated; `file-skeleton`, `read-chunk`, `search-lexical`, `find-usages`, `search-signature`, `largest-chunks`, `search-related` sections updated with `--file`; new `peek` section; REPL shorthand table now includes a flags column
- `SKILL.md`: all three features documented; dedicated `--file` section with examples; Rules 8 and 9 added
- `internal/tui/tui.go`: `help` listing and `printCommandHelp` updated for all affected commands and `pk`

## Files Touched

### New Files

- **`internal/search/filter.go`**: `compileFileFilter`, `matchesFileFilter`
- **`internal/search/peek.go`**: `Peek` function
- **`contexts/compaction-20260307-1.md`**: this file

### Core Logic

- **`internal/search/lexical.go`**: `collectMatchingChunks` and `SearchLexical` gain `fileFilter string`
- **`internal/search/semantic.go`**: `SearchRelated` gains `fileFilter string`; passes to both collect calls and fallback `SearchLexical`
- **`internal/search/signature.go`**: `SearchSignature` gains `fileFilter string`
- **`internal/search/rank.go`**: `LargestChunks` gains `fileFilter string`
- **`internal/search/search.go`**: `FileSkeleton` and `FindUsages` gain `fileFilter string`; new `ReadFullFile`

### CLI / TUI

- **`cmd/garbell/main.go`**: all affected commands use `extractFlag`/`extractBoolFlag`; `peek` and `read-chunk -1` cases added; `printUsage` updated
- **`internal/tui/tui.go`**: all affected cases use `tuiExtractFlag`/`tuiExtractBoolFlag`; `pk` case added; `help` and `printCommandHelp` updated; `pk` in tab-completion

### Tests

- **`internal/search/search_test.go`**: all existing calls updated with empty `fileFilter`; new tests for file filter, `ReadFullFile`, and `Peek`

### Documentation

- **`README.md`**: `peek` in commands table; version badge 0.0.4
- **`REFERENCE.md`**: commands table, all affected command sections, REPL shorthand table
- **`SKILL.md`**: `--file` section, `peek`, `read-chunk -1`, Rules 8 & 9
- **`VERSION`**: `0.0.3` → `0.0.4`
