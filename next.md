# Next Steps & Known Limitations (v2)

During real-world agent testing (Claude), several rough edges were identified in the `garbell` (v2) architecture. These are documented here for future tracking and resolution.

## ~~1. Directory Pathing in `file-skeleton`~~ ✓ DONE

`file-skeleton <dir>` now returns skeletons of all files inside the directory. If the output would exceed the threshold it returns a directory summary instead (see Progressive Disclosure below).

## 2. Module-Level Scope Missing in Chunk Maps

- **Issue**: Running `read-chunk` on line 1 of files often returned `"no chunk found covering line 1"`. The heuristic chunker aggressively chunks functions and classes but does not create a fallback chunk for top-level module scope (e.g., global `const` declarations, variable instantiations, or `import`/`require` blocks).
- **Next Step**: Agents currently must fall back to standard `cat` or `head` for those lines. The indexer (`parsers.go`) should be updated to emit "gap chunks" or a universal `module-scope` chunk covering the lines between explicit functions so that 100% of the file lines are technically inside a chunk.

## ~~3. Lexical Context Explosion on Alternation~~ ✓ DONE (via Progressive Disclosure)

`search-lexical`, `file-skeleton` (directory), and `find-usages` now check a line threshold (default 500, overridable via `GARBELL_MAX_LINES`) before returning full content. When the estimated output would exceed the threshold, they return a directory-grouped overview (chunk counts / symbol counts / caller counts) that the agent can use to zoom in.

## 4. Parser Language Boundaries

- **Issue**: The heuristic chunker currently only maps Go, Python, JS/TS, C++, CSS, and HTML.
- **Next Step**: While sufficient for the vast majority of agent tasks, adding lightweight regex heuristics for Rust, Java, and Ruby would effectively close the gap without needing to import large `tree-sitter` grammars.

## 5. `search-related` Tuning

### 5a. `--min-token N` flag on `index`

- **Issue**: Short abbreviated identifiers (e.g. `pnv`, `p2e`, `tgl`) are 3 characters, pass the current length floor, and pollute the PPMI thesaurus with noise — especially in JS codebases with terse variable names.
- **Next Step**: Add a `--min-token N` flag to the `index` command (default: 3, matching current behaviour). The `Builder` stores the value and applies it as an additional filter in `AddDocument`. Not persisted — the user can raise it for one indexing run and revert freely.

### 5b. `.garbellignore` file

- **Issue**: Third-party or vendored directories (e.g. `libs/3rdparty/`, `node_modules/`, `vendor/`, `dist/`) get indexed, adding thousands of chunks that overwhelm both lexical search results and the PPMI thesaurus.
- **Next Step**: Honour a `.garbellignore` file in the workspace root. Format: one path prefix per line, `#` comments, blank lines ignored. Affects indexing only (chunk shards + semantic builder) — `search-lexical` via `rg` continues to respect `.gitignore` as before. Implementation: after `rg --files` returns the file list in `discoverFiles`, load and apply the ignore prefixes before processing.
