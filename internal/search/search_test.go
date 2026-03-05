package search_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"garbell/internal/indexer"
	"garbell/internal/search"
)

// testWorkspace is a shared indexed workspace set up once for the whole package.
var testWorkspace string

// testGoFile is the Go source file written into testWorkspace.
// Line numbers are significant — tests depend on them.
//
//	 1: package testpkg
//	 2: (blank)
//	 3: import "fmt"
//	 4: (blank)
//	 5: func Greet(name string) string {
//	 6: 	return fmt.Sprintf("Hello, %s!", name)
//	 7: }
//	 8: (blank)
//	 9: func Add(a, b int) int {
//	10: 	return a + b
//	11: }
//	12: (blank)
//	13: func UseGreet() string {
//	14: 	return Greet("world")
//	15: }
//	16: (blank)
//	17: func UseGreetAgain() string {
//	18: 	return Greet("Go")
//	19: }
const testUtilsJS = `export function add(a, b) {
	return a + b;
}

export function multiply(a, b) {
	return a * b;
}
`

const testMainJS = `import { add, multiply } from './utils';

function run() {
	return add(1, multiply(2, 3));
}
`

const testGoFile = `package testpkg

import "fmt"

func Greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

func Add(a, b int) int {
	return a + b
}

func UseGreet() string {
	return Greet("world")
}

func UseGreetAgain() string {
	return Greet("Go")
}
`

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("rg"); err != nil {
		// ripgrep unavailable — integration tests will be skipped individually.
		os.Exit(m.Run())
	}

	dir, err := os.MkdirTemp("", "cercle-search-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte(testGoFile), 0644); err != nil {
		panic(err)
	}
	// JS files for dependents test.
	if err := os.WriteFile(filepath.Join(dir, "utils.js"), []byte(testUtilsJS), 0644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.js"), []byte(testMainJS), 0644); err != nil {
		panic(err)
	}

	// Suppress GenerateIndex stdout during tests.
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	_, err = indexer.GenerateIndex(dir)
	os.Stdout = old
	if err != nil {
		panic(err)
	}

	testWorkspace = dir
	os.Exit(m.Run())
}

func requireRipgrep(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not in PATH")
	}
}

func requireWorkspace(t *testing.T) {
	t.Helper()
	requireRipgrep(t)
	if testWorkspace == "" {
		t.Skip("test workspace not initialized")
	}
}

// --- FileSkeleton ---

func TestFileSkeleton_File(t *testing.T) {
	requireWorkspace(t)

	out, err := search.FileSkeleton(testWorkspace, "hello.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range []string{"Greet", "Add", "UseGreet", "UseGreetAgain"} {
		if !strings.Contains(out, sig) {
			t.Errorf("expected skeleton to contain %q\ngot:\n%s", sig, out)
		}
	}
}

func TestFileSkeleton_File_LineFormat(t *testing.T) {
	requireWorkspace(t)

	out, err := search.FileSkeleton(testWorkspace, "hello.go")
	if err != nil {
		t.Fatal(err)
	}
	// Each line should match "start-end: sig" format.
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if !strings.Contains(line, ":") || !strings.Contains(line, "-") {
			t.Errorf("unexpected skeleton line format: %q", line)
		}
	}
}

func TestFileSkeleton_File_CorrectLineNumbers(t *testing.T) {
	requireWorkspace(t)

	out, err := search.FileSkeleton(testWorkspace, "hello.go")
	if err != nil {
		t.Fatal(err)
	}
	// Greet is on lines 5-7 per the test file layout above.
	if !strings.Contains(out, "5-7") {
		t.Errorf("expected Greet chunk at 5-7; skeleton:\n%s", out)
	}
	// Add is on lines 9-11.
	if !strings.Contains(out, "9-11") {
		t.Errorf("expected Add chunk at 9-11; skeleton:\n%s", out)
	}
}

func TestFileSkeleton_NonExistentFile(t *testing.T) {
	requireWorkspace(t)

	_, err := search.FileSkeleton(testWorkspace, "nonexistent.go")
	if err == nil {
		t.Error("expected an error for a nonexistent file, got nil")
	}
}

func TestFileSkeleton_Directory(t *testing.T) {
	requireWorkspace(t)

	out, err := search.FileSkeleton(testWorkspace, ".")
	if err != nil {
		t.Fatal(err)
	}
	// Directory output should include the filename as a header.
	if !strings.Contains(out, "hello.go") {
		t.Errorf("expected 'hello.go' in directory skeleton; got:\n%s", out)
	}
	if !strings.Contains(out, "Greet") {
		t.Errorf("expected 'Greet' in directory skeleton; got:\n%s", out)
	}
}

func TestFileSkeleton_DirectoryOverflow(t *testing.T) {
	requireWorkspace(t)
	t.Setenv("GARBELL_MAX_LINES", "1")

	out, err := search.FileSkeleton(testWorkspace, ".")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "exceeds") {
		t.Errorf("expected overflow summary, got:\n%s", out)
	}
	if !strings.Contains(out, "file-skeleton") {
		t.Errorf("expected drill-down hint, got:\n%s", out)
	}
}

// --- ReadChunkBlock ---

func TestReadChunkBlock_ByLine(t *testing.T) {
	requireWorkspace(t)

	// Line 10 is inside the Add function (lines 9-11).
	out, err := search.ReadChunkBlock(testWorkspace, "hello.go", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Add") {
		t.Errorf("expected chunk to contain 'Add'; got:\n%s", out)
	}
	if !strings.Contains(out, "return a + b") {
		t.Errorf("expected chunk body; got:\n%s", out)
	}
}

func TestReadChunkBlock_FirstLineOfChunk(t *testing.T) {
	requireWorkspace(t)

	// Line 5 is the first line of the Greet function.
	out, err := search.ReadChunkBlock(testWorkspace, "hello.go", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Greet") {
		t.Errorf("expected chunk to contain 'Greet'; got:\n%s", out)
	}
}

func TestReadChunkBlock_ModuleScopeLine(t *testing.T) {
	requireWorkspace(t)

	// Line 1 (package declaration) is not inside any chunk.
	_, err := search.ReadChunkBlock(testWorkspace, "hello.go", 1)
	if err == nil {
		t.Error("expected error for module-scope line, got nil")
	}
	if !strings.Contains(err.Error(), "no chunk found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- SearchLexical ---

func TestSearchLexical_ReturnsMatchingChunk(t *testing.T) {
	requireWorkspace(t)

	results, err := search.SearchLexical(testWorkspace, "fmt.Sprintf")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result for 'fmt.Sprintf'")
	}
	if !strings.Contains(results[0], "Greet") {
		t.Errorf("expected result to contain 'Greet'; got:\n%s", results[0])
	}
}

func TestSearchLexical_NoMatches(t *testing.T) {
	requireWorkspace(t)

	results, err := search.SearchLexical(testWorkspace, "xyznomatchstring")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestSearchLexical_ResultIncludesHeader(t *testing.T) {
	requireWorkspace(t)

	results, err := search.SearchLexical(testWorkspace, "return a \\+ b")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected result for 'return a + b'")
	}
	// New compact format: "file.go:\n  start-end: sig\n..."
	if !strings.Contains(results[0], ":\n") {
		t.Errorf("expected compact file:chunk listing; got:\n%s", results[0])
	}
	if !strings.Contains(results[0], "read-chunk") {
		t.Errorf("expected read-chunk hint in result; got:\n%s", results[0])
	}
}

func TestSearchLexical_Deduplication(t *testing.T) {
	requireWorkspace(t)

	// Both UseGreet and UseGreetAgain call Greet — but they are different chunks.
	// Searching for "Greet" should match many things; verify no duplicate chunk keys.
	results, err := search.SearchLexical(testWorkspace, "Greet")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, r := range results {
		// Extract the file+line header as dedup key.
		firstLine := strings.SplitN(r, "\n", 2)[0]
		if seen[firstLine] {
			t.Errorf("duplicate result: %s", firstLine)
		}
		seen[firstLine] = true
	}
}

func TestSearchLexical_OverflowSummary(t *testing.T) {
	requireWorkspace(t)
	t.Setenv("GARBELL_MAX_SUMMARY_CHUNKS", "1")

	// With cap=1 chunk, any query returning 2+ chunks triggers the directory overview.
	results, err := search.SearchLexical(testWorkspace, "func")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 overflow summary result, got %d", len(results))
	}
	if !strings.Contains(results[0], "exceed") {
		t.Errorf("expected overflow message, got:\n%s", results[0])
	}
	if !strings.Contains(results[0], "file-skeleton") {
		t.Errorf("expected drill-down hint in overflow, got:\n%s", results[0])
	}
}

func TestSearchLexical_RegexAlternation(t *testing.T) {
	requireWorkspace(t)

	// Pipe alternation should work without escaping.
	results, err := search.SearchLexical(testWorkspace, "Greet|Add")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("expected results for 'Greet|Add'")
	}
}

// --- FindUsages ---

func TestFindUsages_ReturnsCallers(t *testing.T) {
	requireWorkspace(t)

	sigs, err := search.FindUsages(testWorkspace, "Greet")
	if err != nil {
		t.Fatal(err)
	}
	if len(sigs) == 0 {
		t.Fatal("expected callers of Greet")
	}
	// UseGreet and UseGreetAgain both call Greet.
	found := make(map[string]bool)
	for _, s := range sigs {
		if strings.Contains(s, "UseGreet") {
			found["UseGreet"] = true
		}
		if strings.Contains(s, "UseGreetAgain") {
			found["UseGreetAgain"] = true
		}
	}
	if !found["UseGreet"] {
		t.Errorf("expected UseGreet in callers; got: %v", sigs)
	}
	if !found["UseGreetAgain"] {
		t.Errorf("expected UseGreetAgain in callers; got: %v", sigs)
	}
}

func TestFindUsages_NoMatches(t *testing.T) {
	requireWorkspace(t)

	sigs, err := search.FindUsages(testWorkspace, "XyzNoSuchSymbol")
	if err != nil {
		t.Fatal(err)
	}
	if len(sigs) != 0 {
		t.Errorf("expected no usages, got: %v", sigs)
	}
}

func TestFindUsages_OverflowSummary(t *testing.T) {
	requireWorkspace(t)
	t.Setenv("GARBELL_MAX_LINES", "1")

	// Greet is called in 2 functions; with threshold=1 it should overflow.
	sigs, err := search.FindUsages(testWorkspace, "Greet")
	if err != nil {
		t.Fatal(err)
	}
	if len(sigs) != 1 {
		t.Fatalf("expected exactly 1 overflow summary, got %d", len(sigs))
	}
	if !strings.Contains(sigs[0], "caller") {
		t.Errorf("expected overflow message with 'caller'; got:\n%s", sigs[0])
	}
}

func TestFindUsages_Deduplication(t *testing.T) {
	requireWorkspace(t)

	// Each caller function should appear at most once.
	sigs, err := search.FindUsages(testWorkspace, "Greet")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, s := range sigs {
		if seen[s] {
			t.Errorf("duplicate usage: %s", s)
		}
		seen[s] = true
	}
}

func TestFindUsages_FormatIsFilePlusSig(t *testing.T) {
	requireWorkspace(t)

	sigs, err := search.FindUsages(testWorkspace, "Greet")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sigs {
		// Format is "relpath:signature"
		if !strings.Contains(s, ":") {
			t.Errorf("unexpected format (missing ':'): %q", s)
		}
		parts := strings.SplitN(s, ":", 2)
		if !strings.HasSuffix(parts[0], ".go") {
			t.Errorf("expected file path part to end in .go: %q", parts[0])
		}
	}
}

// --- SearchSignature ---

func TestSearchSignature_MatchesBySig(t *testing.T) {
	requireWorkspace(t)

	out, err := search.SearchSignature(testWorkspace, `func.*int`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Add") {
		t.Errorf("expected 'Add' in signature results; got:\n%s", out)
	}
}

func TestSearchSignature_GroupedByFile(t *testing.T) {
	requireWorkspace(t)

	out, err := search.SearchSignature(testWorkspace, `func`)
	if err != nil {
		t.Fatal(err)
	}
	// Output should include the filename as a group header.
	if !strings.Contains(out, "hello.go:") {
		t.Errorf("expected file header in grouped output; got:\n%s", out)
	}
}

func TestSearchSignature_NoMatches(t *testing.T) {
	requireWorkspace(t)

	out, err := search.SearchSignature(testWorkspace, `XyzNoSuchPattern`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("expected empty output, got: %q", out)
	}
}

func TestSearchSignature_InvalidPattern(t *testing.T) {
	requireWorkspace(t)

	_, err := search.SearchSignature(testWorkspace, `[invalid`)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestSearchSignature_Overflow(t *testing.T) {
	requireWorkspace(t)
	t.Setenv("GARBELL_MAX_LINES", "1")

	out, err := search.SearchSignature(testWorkspace, `func`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "exceed") {
		t.Errorf("expected overflow summary; got:\n%s", out)
	}
}

// --- LargestChunks ---

func TestLargestChunks_ReturnsN(t *testing.T) {
	requireWorkspace(t)

	results, err := search.LargestChunks(testWorkspace, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestLargestChunks_SortedDescending(t *testing.T) {
	requireWorkspace(t)

	results, err := search.LargestChunks(testWorkspace, 10)
	if err != nil {
		t.Fatal(err)
	}
	// Parse the line count from each result ("NNN lines ...").
	prevSize := int(^uint(0) >> 1) // max int
	for _, r := range results {
		var size int
		if _, err := fmt.Sscanf(strings.TrimSpace(r), "%d", &size); err != nil {
			t.Fatalf("could not parse size from %q: %v", r, err)
		}
		if size > prevSize {
			t.Errorf("results not sorted descending: %d > %d", size, prevSize)
		}
		prevSize = size
	}
}

func TestLargestChunks_FormatIncludesFile(t *testing.T) {
	requireWorkspace(t)

	results, err := search.LargestChunks(testWorkspace, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	// Format: "NNN lines  <sig>  (<file>:start-end)"
	r := results[0]
	if !strings.Contains(r, "lines") {
		t.Errorf("expected 'lines' in result; got: %q", r)
	}
	if !strings.Contains(r, ".go") && !strings.Contains(r, ".js") {
		t.Errorf("expected a filename in result; got: %q", r)
	}
}

func TestLargestChunks_DefaultsToAll(t *testing.T) {
	requireWorkspace(t)

	all, err := search.LargestChunks(testWorkspace, 0)
	if err != nil {
		t.Fatal(err)
	}
	top10, err := search.LargestChunks(testWorkspace, 10)
	if err != nil {
		t.Fatal(err)
	}
	// With n=0 we should get all chunks; they should be >= 10 count (or equal if fewer exist).
	if len(all) < len(top10) {
		t.Errorf("n=0 returned fewer results (%d) than n=10 (%d)", len(all), len(top10))
	}
}

// --- Callees ---

func TestCallees_ReturnsCalledFunctions(t *testing.T) {
	requireWorkspace(t)

	// UseGreet (line 13-15) calls Greet.
	results, err := search.Callees(testWorkspace, "hello.go", 13)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range results {
		if strings.Contains(r, "Greet") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Greet' in callees; got: %v", results)
	}
}

func TestCallees_AnnotatesLocalFunctions(t *testing.T) {
	requireWorkspace(t)

	// Greet (line 5-7) calls fmt.Sprintf. Sprintf is external; Greet itself is local.
	// UseGreet calls Greet which IS in the index → should be annotated with location.
	results, err := search.Callees(testWorkspace, "hello.go", 13)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if strings.Contains(r, "Greet") && strings.Contains(r, "→") {
			return // found an annotated local callee
		}
	}
	t.Errorf("expected at least one callee annotated with '→ file:line'; got: %v", results)
}

func TestCallees_ModuleScopeError(t *testing.T) {
	requireWorkspace(t)

	_, err := search.Callees(testWorkspace, "hello.go", 1)
	if err == nil {
		t.Error("expected error for module-scope line")
	}
}

// --- Dependents ---

func TestDependents_FindsImporter(t *testing.T) {
	requireRipgrep(t)
	if testWorkspace == "" {
		t.Skip("test workspace not initialized")
	}

	// main.js imports from './utils', so dependents of utils.js should include main.js.
	results, err := search.Dependents(testWorkspace, "utils.js")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range results {
		if strings.Contains(r, "main.js") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected main.js in dependents of utils.js; got: %v", results)
	}
}

func TestDependents_ExcludesSelf(t *testing.T) {
	requireRipgrep(t)
	if testWorkspace == "" {
		t.Skip("test workspace not initialized")
	}

	results, err := search.Dependents(testWorkspace, "utils.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if strings.HasPrefix(r, "utils.js:") {
			t.Errorf("dependents should not include the file itself: %s", r)
		}
	}
}

func TestDependents_NoImporters(t *testing.T) {
	requireRipgrep(t)
	if testWorkspace == "" {
		t.Skip("test workspace not initialized")
	}

	// main.js is not imported by anyone in the workspace.
	results, err := search.Dependents(testWorkspace, "main.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		// No file should list main.js as an import.
		if strings.Contains(r, "main.js") && !strings.HasPrefix(r, "main.js:") {
			t.Errorf("unexpected dependent of main.js: %s", r)
		}
	}
}
