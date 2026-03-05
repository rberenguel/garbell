# Garbell Reference

`garbell` is a local, daemonless CLI tool for indexing and searching codebases. It is designed to give LLM agents (and developers) structured access to code without reading whole files into context.

## Installation

```bash
task build
# or manually:
go build -o ~/.garbell/garbell ./cmd/garbell
```

Requires `rg` (ripgrep) in `PATH`.

---

## Commands

Each command answers a distinct question about a codebase:

| Question                             | Command                      |
| ------------------------------------ | ---------------------------- |
| _What exists and where?_             | `file-skeleton <path>`       |
| _What does this do?_                 | `read-chunk <file> <line>`   |
| _Where is X mentioned?_              | `search-lexical <query>`     |
| _Who calls this?_                    | `find-usages <symbol>`       |
| _What does this expose?_             | `extract-interface <file>`   |
| _What has this shape/signature?_     | `search-signature <pattern>` |
| _Where is the complexity?_           | `largest-chunks [n]`         |
| _What does this call?_               | `callees <file> <line>`      |
| _What imports this?_                 | `dependents <file>`          |
| _What is conceptually related to X?_ | `search-related <query>`     |
| _Interactive exploration?_           | `repl`                       |

`index` is a prerequisite for all of the above.

---

### `index`

Walks the project using `rg --files` (respects `.gitignore`), parses source files with language-specific heuristics, and writes a sharded JSON chunk map to `~/.garbell/indexes/<workspace-hash>/`. It also builds a **PPMI thesaurus** — a repository-specific vocabulary of related terms derived from token co-occurrences — and saves it as `ppmi.json` in the same directory. This file powers `search-related`.

```bash
garbell index
```

Expected output:

```
Parsed 23 files to index...
Successfully wrote 14 shards to /Users/you/.garbell/indexes/a3f9...
```

Re-run after modifying code. It is fast and safe to re-run at any time.

---

### `file-skeleton <path>`

Returns the structural outline of a file or directory: line ranges and function/class/type signatures extracted from the chunk map. Does not read file content at query time — it reads the index.

**Single file:**

```bash
garbell file-skeleton src/auth/login.go
```

```
12-34: func Login(ctx context.Context, creds Credentials) (Token, error) {
36-41: func validateCreds(c Credentials) bool {
43-78: func (s *Session) Refresh() error {
```

**Directory:**

```bash
garbell file-skeleton src/auth/
```

```
src/auth/login.go:
  12-34: func Login(ctx context.Context, creds Credentials) (Token, error) {
  36-41: func validateCreds(c Credentials) bool {

src/auth/token.go:
  5-22: func NewToken(claims Claims) (string, error) {
  24-39: func ParseToken(raw string) (Claims, error) {
```

When the output would exceed the line threshold, a directory summary is returned instead:

```
Output exceeds 500 lines (87 symbols across 12 files). Directory summary:

  src/auth/                            4 file(s)   15 symbol(s)
  src/models/                          2 file(s)   28 symbol(s)
  ...

Use `file-skeleton <subdir>` to drill down.
```

---

### `read-chunk <path> <line>`

Reads the complete source block (function, class, CSS rule, etc.) enclosing the given line number. Output is capped at 1000 lines.

```bash
garbell read-chunk src/auth/login.go 25
```

Use this after `file-skeleton` tells you the line range of the thing you want to read, or after a `search-lexical` result that was truncated.

**Known limitation:** line 1 and other module-scope lines (imports, top-level `const` blocks) are not inside any chunk. `read-chunk` will return an error for those lines. Use `extract-interface` or direct file reading instead.

---

### `search-lexical <query>`

Runs `rg -n -e <query>` across the workspace, maps every match to its enclosing chunk, deduplicates, and returns the full function/class body for each. Each chunk is capped at 100 lines.

The query is a **PCRE/Rust regex** — `|` alternation works directly, no escaping:

```bash
garbell search-lexical "handleLogin|handleLogout"
garbell search-lexical "SELECT.*FROM users"
garbell search-lexical "func.*Error\(\)"
```

When the total estimated output would exceed the line threshold, a directory-grouped overview is returned instead:

```
Results exceed 500 lines (~18 chunks across 9 files). Drill down by location:

  src/auth/                            6 chunk(s)  [login.go, session.go, token.go]
  src/models/                          5 chunk(s)  [user.go, session.go]
  ...

Refine your query, add a path, or use `file-skeleton <dir>` to explore.
```

To search within a specific directory, use a path-anchored regex: `garbell search-lexical "handleLogin" src/auth/` — actually, pass the path as part of the query or use `rg` flags. The current interface only takes a query string; for path filtering run `rg` directly or narrow the query.

---

### `search-related <query>`

An expanded search that uses the repository-specific **PPMI thesaurus** built during `index` to automatically widen the query before handing it off to `search-lexical`.

**How it works:**

1. The query is tokenised the same way the codebase was during indexing — split on non-alphanumeric boundaries and camelCase transitions, lowercased, stop-words removed.
2. Each token is looked up in the `ppmi.json` thesaurus. The thesaurus maps every token to the top 5 terms that co-occur with it most strongly across the whole codebase (measured by Positive Pointwise Mutual Information, a log-ratio of observed vs. expected co-occurrence frequency).
3. The original tokens and all their synonyms are combined into a single case-insensitive alternation regex — e.g. `(?i)(login|auth|session|jwt|token)` — and passed to `search-lexical`.

**When to use it:**

- When you know what a concept _does_ but not what it is _called_: `search-related "user login"` may surface `authenticate`, `session`, `jwt`, even if none of those words appear in your query.
- When `search-lexical` returns nothing or too few results for an abstract concept.
- When the codebase uses different vocabulary than your query (e.g., querying `"error handling"` but the code says `"failure recovery"`).

**When not to use it:**

- For exact symbol lookups — use `search-lexical` or `find-usages` instead; the expansion will add noise.
- On a freshly cloned project that has never been indexed — `ppmi.json` does not exist yet; run `index` first.

```bash
garbell search-related "authentication"
# Might expand to: (?i)(authentication|login|session|token|jwt|oauth)

garbell search-related "database connection"
# Might expand to: (?i)(database|connection|pool|driver|query|transaction)
```

The output format is identical to `search-lexical` — full chunk bodies, capped at 100 lines each, with the same progressive-disclosure threshold.

> **Requires `ppmi.json`** — produced by `garbell index`. If the file is missing, the command returns an error asking you to re-run `index`.

---

### `search-fuzzy <signature>`

Finds the closest matching function signature across the entire index using Levenshtein distance. Useful when you know roughly what something is called but not the exact name.

```bash
garbell search-fuzzy "parseJwtToken"
```

```
Closest Match: func ParseToken(raw string) (Claims, error) { (in src/auth/token.go)

func ParseToken(raw string) (Claims, error) {
    ...
}
```

---

### `find-usages <symbol>`

Searches for exact word-boundary matches of `symbol` using `rg -w`, maps each match to its enclosing chunk, and returns only the **calling function signatures** — not the call sites themselves. Efficient for impact analysis before refactoring.

```bash
garbell find-usages ParseToken
```

```
src/auth/middleware.go:func AuthMiddleware(next http.Handler) http.Handler {
src/auth/login.go:func Login(ctx context.Context, creds Credentials) (Token, error) {
```

When there are too many callers, a directory summary is returned instead.

---

### `extract-interface <path>`

Reads a source file directly (not the index) and returns only its public surface: imports and exported declarations. Useful for understanding what a module exposes without reading its implementation.

```bash
garbell extract-interface src/auth/token.go
```

Language support:

- **Go**: `import` blocks + `func`/`type` starting with uppercase
- **Python**: `import`/`from` + `def`/`class` not starting with `_`
- **JS/TS**: `import`/`export` statements + export declarations
- **C/C++**: `#include` directives only

---

### `search-signature <pattern>`

Searches chunk **signatures only** (not file bodies) for matches against a regex. No source file I/O — runs entirely against the in-memory index. Useful for finding functions by their structural shape rather than their content.

```bash
garbell search-signature "func.*Handler"
garbell search-signature "func.*\(.*Context"
garbell search-signature "class.*Repository"
```

```
src/http/routes.go:
  12-45: func LoginHandler(w http.ResponseWriter, r *http.Request) {
  48-82: func LogoutHandler(w http.ResponseWriter, r *http.Request) {
```

When there are too many matches, a directory-grouped count is returned instead.

---

### `largest-chunks [n]`

Returns the `n` largest chunks by line count, descending (default: 10). Instant answer to "where is the complexity in this codebase?" — useful as the first step when inheriting unfamiliar code.

```bash
garbell largest-chunks
garbell largest-chunks 5
```

```
 312 lines  func processEvents(ctx context.Context) error {  (src/engine/loop.go:88-399)
 201 lines  class SceneManager  (src/scene.js:14-214)
  98 lines  func renderFrame(...)  (src/renderer.go:22-119)
```

---

### `callees <path> <line>`

Returns the function/method names called **within** the chunk enclosing the given line. Answers "what does this function call?" — the inverse of `find-usages`. Callees that are defined in this codebase are annotated with their location (`→ file:start-end`); external/stdlib calls are listed without annotation.

```bash
garbell callees src/auth/login.go 15
```

```
validateCreds  →  src/auth/login.go:36-41
generateToken  →  src/auth/token.go:5-22
bcrypt.CompareHashAndPassword
context.WithTimeout
fmt.Errorf
```

Uses heuristic regex pattern matching — accurate for straightforward call expressions, may miss indirect calls or method chains in complex expressions.

---

### `dependents <path>`

Finds files that **import or reference** this file. Answers "what imports this?" — the reverse of `extract-interface`. Searches source files for import/require/from/include lines (and bare quoted paths for Go multi-line imports) containing the file's name or parent directory.

```bash
garbell dependents src/auth/token.go
```

```
src/auth/login.go:3: "myproject/src/auth"
src/middleware/auth.go:5: "myproject/src/auth"
src/api/routes.go:8: "myproject/src/auth"
```

---

### `repl`

Opens an interactive Read-Eval-Print Loop (REPL) for exploring the workspace. The REPL includes history (up/down arrow keys), cursor navigation, and tab completion for both commands and file paths using the codebase index.

Within the REPL, commands have shorthand aliases to save typing:

| Shorthand          | Full Command        |
| ------------------ | ------------------- |
| `fs <path>`        | `file-skeleton`     |
| `rc <file> <line>` | `read-chunk`        |
| `sl <query>`       | `search-lexical`    |
| `fu <symbol>`      | `find-usages`       |
| `ei <file>`        | `extract-interface` |
| `ss <pattern>`     | `search-signature`  |
| `lc [n]`           | `largest-chunks`    |
| `ca <file> <line>` | `callees`           |
| `dep <file>`       | `dependents`        |
| `sf <sig>`         | `search-fuzzy`      |
| `sr <query>`       | `search-related`    |

Other REPL-specific commands:

- `use <path>`: Sets the workspace directory and loads its index for tab completion.
- `index`: Regenerates the index for the current workspace.
- `help` / `?`: Prints the command list.
- `exit` / `q` / `Ctrl+D`: Quits the REPL.

_Example workflow:_

```bash
garbell repl
garbell (no workspace)> use /path/to/project
garbell (project)> fs src/auth/            # Press Tab to auto-complete paths
garbell (project)> rl src/auth/login.go 25
garbell (project)> exit
```

---

## Workflows

### Cold start on an unfamiliar codebase

```bash
# 1. Build the index
garbell index

# 2. Get a high-level structural overview
garbell file-skeleton .
# → if too many results, it shows a directory summary; drill into the dirs that look relevant

# 3. Explore a specific area
garbell file-skeleton src/auth/

# 4. Read the function you care about
garbell read-chunk src/auth/login.go 12

# 5. Search for related code
garbell search-lexical "refreshToken|renewSession"
```

### Active editing session

After modifying files, re-index to keep results accurate:

```bash
garbell index
```

The index is written to `~/.garbell/indexes/` (not inside the repo), so re-indexing does not create any tracked files.

### Refactoring: find all callers before renaming a function

```bash
# Find everything that calls OldFunctionName
garbell find-usages OldFunctionName

# For each caller file, get the skeleton so you know the structure
garbell file-skeleton src/payments/

# Read the specific caller
garbell read-chunk src/payments/charge.go 88
```

---

## Progressive Disclosure

Several commands have a line threshold (default **500 lines**, configurable via `GARBELL_MAX_LINES`). When the output would exceed this threshold, the command returns a **directory-grouped summary** instead of the full content.

This is not pagination. The summary is a genuinely different, denser representation designed to help you decide where to zoom in next:

| Full output exceeded  | What you get instead                 | Next step                                           |
| --------------------- | ------------------------------------ | --------------------------------------------------- |
| `search-lexical`      | Dirs with chunk counts + file names  | Narrow the query or drill into a specific dir       |
| `file-skeleton <dir>` | Dirs with file/symbol counts         | Run `file-skeleton` on a specific subdir            |
| `find-usages`         | Dirs with caller counts + file names | Use `read-chunk` on files in the dir you care about |

The summary messages always include a hint about what to do next.

### Adjusting the threshold

```bash
# Temporarily raise the limit (e.g., bigger context window)
GARBELL_MAX_LINES=2000 garbell search-lexical "authenticate"

# Lower it for a smaller context budget
GARBELL_MAX_LINES=200 garbell file-skeleton src/
```

---

## Supported Languages

| Extension                            | Parser                     | Extracts                                                               |
| ------------------------------------ | -------------------------- | ---------------------------------------------------------------------- |
| `.go`                                | Heuristic (brace counting) | `func`, `type`                                                         |
| `.py`                                | Heuristic (indentation)    | `def`, `class`, `async def`                                            |
| `.js` `.ts` `.jsx` `.tsx`            | Heuristic (brace stack)    | `function`, `class`, arrow functions (`const f = () => {}`)            |
| `.c` `.cpp` `.cc` `.cxx` `.h` `.hpp` | Heuristic (brace counting) | function definitions, `class`, `struct`                                |
| `.css`                               | Heuristic (brace counting) | CSS rules/selectors                                                    |
| `.html` `.htm`                       | Tag matching               | `<script>`, `<style>`, `<main>`, `<div id=...>`                        |
| `.md` `.mdx`                         | Heading-based              | ATX headings (`#` through `######`); each heading section is one chunk |
| `.proto`                             | Heuristic (brace counting) | `message`, `service`, `enum` blocks                                    |

Files with unsupported extensions are skipped entirely by the indexer.

For files of supported types that produce no recognized chunks (e.g., a Go file that only contains variable declarations), and that are longer than 50 lines, the indexer falls back to overlapping 50-line sliding windows.

---

## Index Storage

Indexes are stored globally in `~/.garbell/indexes/<workspace-hash>/` where the hash is the MD5 of the absolute workspace path. Each workspace gets its own directory; multiple projects do not interfere.

Inside each workspace index:

- `metadata.json` — absolute path + last-updated timestamp
- `00.json` … `ff.json` — chunk shards (up to 256), keyed by the first 2 hex chars of the MD5 of each file's relative path
- `ppmi.json` — PPMI thesaurus: a map from every indexed token to its top 5 co-occurring terms, used by `search-related`

### Removing an index

```bash
# Remove the index for the current project
rm -rf ~/.garbell/indexes/$(echo -n $(pwd) | md5 | cut -c1-32)

# Remove all indexes
rm -rf ~/.garbell/indexes/
```

---

## Known Limitations

1. **Module-scope lines** — `read-chunk` returns an error for lines not inside a function/class (imports, top-level constants, etc.). Use `extract-interface` for the public surface or read the file directly for top-level declarations.

2. **Python consecutive top-level defs** — The Python parser closes a function/class chunk only when it sees a non-def/class line at the same indentation level. Two consecutive top-level `def` blocks without intervening non-def code may be grouped into one chunk.

3. **Brace-in-string literals** — Go, JS, C++ parsers count raw `{`/`}` characters. A function containing string literals with braces may have its chunk boundary miscounted.

4. **Missing languages** — Rust, Java, Ruby, Swift are not currently supported. Files in those languages are skipped by the indexer.

5. **Search requires ripgrep** — `search-lexical`, `find-usages`, and `index` all shell out to `rg`. If ripgrep is not installed, these commands fail silently or return no results.
