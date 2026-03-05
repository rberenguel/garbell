---
name: garbell
description: Code indexing and search skill using the purely local, daemonless Cercle v2 ("garbell") tool. Use when you need to navigate a codebase, map out files, locate usages, or extract code blocks directly from the source using ripgrep under the hood. 
allowed-tools: Bash(garbell *)
---

# Garbell Context Retrieval

`garbell` is a completely local, zero-dependency Go binary built around `ripgrep` (`rg`).
Output is pure, compact text—designed specifically for LLM contexts.
*Note on Reindexing: Reindexing is extremely fast. If the source code changes, you must re-run `index` instantly to avoid stale data.*

## Tools / Commands

Invoke these via `~/.garbell/garbell <command> [args...]` from the root of a project.

- `index` — Traverses the codebase (respecting `.gitignore`), parses Go, Python, JS/TS, C++, CSS, HTML, Markdown (`.md`/`.mdx`), and Protobuf (`.proto`), and generates JSON chunk map shards mapped cleanly in `~/.garbell/indexes/`. Re-run this whenever you modify the codebase.
- `search-lexical <query>` — Full-text search using `rg`. Evaluates the query implicitly as **PCRE/Rust regex** (so `generate|tint` works! **Do NOT escape pipes like `\|`**). Returns the **entire function/class body** surrounding each match, capped at 100 lines per chunk, deduplicated. **If results would exceed the line threshold, returns a directory-grouped overview instead** — use that to narrow your query or drill into a specific location.
- `search-fuzzy <signature>` — Fuzzy searches across the entire vocabulary of the chunk map using pure-Go Levenshtein distance. Capped at **100 lines** of output. Use this if you know a symbol name but aren't sure of the exact spelling or capitalization.
- `file-skeleton <filepath|dir>` — Returns a structural view of a file (line numbers + signatures) or, if given a directory, the skeletons of all files inside. **If the output would exceed the line threshold, returns a directory summary with symbol counts instead** — use that to pick a subdirectory to drill into.
- `read-chunk <filepath> <line_number>` — Reads exactly the code block enclosing a specific line number. Capped at **1000 lines**. Use this after `file-skeleton` or after a truncated `search-lexical` result to drill into a specific dense function.
- `find-usages <symbol>` — Uses `rg -w` to find usages of a symbol and returns only the calling function signatures. **If there are too many callers, returns a directory-grouped count instead.** Extreme token efficiency.
- `extract-interface <filepath>` — Extracts only the imports/includes and exported declarations (`func`, `export`, `#include`, `def`, `class`) for a file. Perfect for understanding file contracts.
- `search-signature <pattern>` — Searches chunk signatures by regex. No file I/O — purely against the index. Use to find functions by shape: `search-signature "func.*Handler"`, `search-signature "class.*Service"`.
- `largest-chunks [n]` — Returns the n largest chunks by line count (default 10). First thing to run on an unfamiliar codebase to identify where complexity lives.
- `callees <filepath> <line_number>` — Returns function names called within the chunk at this line. Callees defined in this codebase are annotated with their location (`→ file:start-end`). Heuristic — accurate for common call patterns.
- `dependents <filepath>` — Finds source files that import or reference this file. Run before refactoring or deleting a file to know what will break.

## Rules

1. **Always Index First**: If the chunk maps don't exist in `~/.garbell/indexes/` or if you've recently refactored heavily, boldly run `~/.garbell/garbell index`. It's incredibly fast and safe to re-run.
2. **Progressive disclosure**: When a tool returns a directory-grouped summary instead of full results, that is a signal to zoom in — narrow your query, or re-run the command on a specific subdirectory or file.
3. **Search-Lexical over Grep**: Prefer `search-lexical` over raw `grep` or `rg`. It gives you the full function boundaries, meaning you rarely have to follow up with `cat` to understand the context.
4. **File-Skeleton to Orient**: When you hit an unknown file or directory, run `file-skeleton` first. Then use `read-chunk` to jump directly into the function you care about.
5. **Find-Usages for Refactoring**: Use `find-usages` when renaming or modifying a core struct. It will tell you exactly which functions in which files call it, without spamming your context with the calls themselves.
6. **Extract-Interface to Grasp Modules**: If you want to know what a module provides without reading its implementation, `extract-interface` is your best friend.
7. **Override threshold**: Set `GARBELL_MAX_LINES=<n>` to raise or lower the output threshold for the current invocation (default: 500).

## Errors

- **Missing Index**: `open ~/.garbell/indexes/... no such file or directory`. Run `~/.garbell/garbell index`.
- **Empty Results**: Be cautious with exact queries in `search-lexical`. If it's empty, try a broader term, or use `search-fuzzy` if you suspect the symbol exists but is spelled differently.
- **Important**: debrief the user of any errors or usability improvements you find in the tooling after using it. This will make it better.
