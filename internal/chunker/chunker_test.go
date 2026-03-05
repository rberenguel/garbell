package chunker_test

import (
	"os"
	"path/filepath"
	"testing"

	"garbell/internal/chunker"
)

// writeTempFile creates a temp file with the given content and extension.
func writeTempFile(t *testing.T, content, ext string) string {
	t.Helper()
	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "test*"+ext)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

// --- Go parser ---

func TestGoParser_TwoFunctions(t *testing.T) {
	src := `package main

func Alpha() {
	return
}

func Beta(x int) int {
	return x * 2
}
`
	path := writeTempFile(t, src, ".go")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	alpha := chunks[0]
	if alpha.Sig != "func Alpha() {" {
		t.Errorf("unexpected sig: %q", alpha.Sig)
	}
	if alpha.Start != 3 || alpha.End != 5 {
		t.Errorf("Alpha: expected lines 3-5, got %d-%d", alpha.Start, alpha.End)
	}

	beta := chunks[1]
	if beta.Sig != "func Beta(x int) int {" {
		t.Errorf("unexpected sig: %q", beta.Sig)
	}
	if beta.Start != 7 || beta.End != 9 {
		t.Errorf("Beta: expected lines 7-9, got %d-%d", beta.Start, beta.End)
	}
}

func TestGoParser_TypeDecl(t *testing.T) {
	src := `package main

type Result struct {
	Value int
	Err   error
}
`
	path := writeTempFile(t, src, ".go")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Sig != "type Result struct {" {
		t.Errorf("unexpected sig: %q", chunks[0].Sig)
	}
}

func TestGoParser_NestedBraces(t *testing.T) {
	src := `package main

func WithMap() map[string]int {
	return map[string]int{
		"a": 1,
		"b": 2,
	}
}
`
	path := writeTempFile(t, src, ".go")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Nested braces inside the function body should not prematurely close the chunk.
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Start != 3 || chunks[0].End != 8 {
		t.Errorf("expected lines 3-8, got %d-%d", chunks[0].Start, chunks[0].End)
	}
}

func TestGoParser_FileRelPath(t *testing.T) {
	src := "package main\n\nfunc Foo() {}\n"
	path := writeTempFile(t, src, ".go")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}
	// Chunk.File should be the path passed to ParseFile.
	if chunks[0].File != path {
		t.Errorf("expected File=%q, got %q", path, chunks[0].File)
	}
}

// --- Python parser ---

func TestPythonParser_SingleFunction(t *testing.T) {
	src := `def compute(x):
    result = x * 2
    return result
`
	path := writeTempFile(t, src, ".py")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	c := chunks[0]
	if c.Sig != "def compute(x):" {
		t.Errorf("unexpected sig: %q", c.Sig)
	}
	if c.Start != 1 {
		t.Errorf("expected Start=1, got %d", c.Start)
	}
	// Closed by finalize at end of file.
	if c.End != 3 {
		t.Errorf("expected End=3, got %d", c.End)
	}
}

func TestPythonParser_AsyncFunction(t *testing.T) {
	src := `async def fetch(url):
    return await get(url)
`
	path := writeTempFile(t, src, ".py")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Sig != "async def fetch(url):" {
		t.Errorf("unexpected sig: %q", chunks[0].Sig)
	}
}

func TestPythonParser_FunctionClosedByNonDef(t *testing.T) {
	// A non-def line at indent 0 closes the previous function chunk.
	src := `def greet(name):
    return f"Hello, {name}!"

x = 1

def farewell(name):
    return f"Bye, {name}!"
`
	path := writeTempFile(t, src, ".py")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// greet closes at line 4 (x=1) → end=3; farewell closes via finalize → end=7
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Sig != "def greet(name):" {
		t.Errorf("unexpected sig[0]: %q", chunks[0].Sig)
	}
	if chunks[1].Sig != "def farewell(name):" {
		t.Errorf("unexpected sig[1]: %q", chunks[1].Sig)
	}
}

// --- JS/TS parser ---

func TestJSParser_FunctionDeclaration(t *testing.T) {
	src := `function greet(name) {
    return "Hello, " + name;
}
`
	path := writeTempFile(t, src, ".js")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Sig != "function greet(name)" {
		t.Errorf("unexpected sig: %q", chunks[0].Sig)
	}
	if chunks[0].Start != 1 || chunks[0].End != 3 {
		t.Errorf("expected lines 1-3, got %d-%d", chunks[0].Start, chunks[0].End)
	}
}

func TestJSParser_ArrowFunction(t *testing.T) {
	src := `const square = (x) => {
    return x * x;
};
`
	path := writeTempFile(t, src, ".js")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Sig != "const square = (x) =>" {
		t.Errorf("unexpected sig: %q", chunks[0].Sig)
	}
}

func TestJSParser_Class(t *testing.T) {
	src := `class Calculator {
    add(a, b) {
        return a + b;
    }
}
`
	path := writeTempFile(t, src, ".js")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}
	// The outer class chunk should be present.
	found := false
	for _, c := range chunks {
		if c.Sig == "class Calculator" {
			found = true
			if c.Start != 1 || c.End != 5 {
				t.Errorf("Calculator: expected lines 1-5, got %d-%d", c.Start, c.End)
			}
		}
	}
	if !found {
		t.Errorf("no chunk with sig 'class Calculator' found; got: %+v", chunks)
	}
}

func TestJSParser_InlineArrowFunction(t *testing.T) {
	// Single-line arrow: chunk is captured and sig does not include the body.
	src := `const double = (x) => { return x * 2; };
`
	path := writeTempFile(t, src, ".js")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Sig != "const double = (x) =>" {
		t.Errorf("unexpected sig: %q", chunks[0].Sig)
	}
	if chunks[0].Start != 1 || chunks[0].End != 1 {
		t.Errorf("expected lines 1-1, got %d-%d", chunks[0].Start, chunks[0].End)
	}
}

func TestJSParser_ExpressionBodyArrow(t *testing.T) {
	// Expression-body arrow (no braces) must not pollute the stack.
	// It produces no chunk (no block body to index), and must not cause
	// the next closing brace in the file to be mis-attributed.
	src := `const double = x => x * 2;

function greet(name) {
    return "Hello, " + name;
}
`
	path := writeTempFile(t, src, ".js")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Only the function declaration should produce a chunk.
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Sig != "function greet(name)" {
		t.Errorf("unexpected sig: %q", chunks[0].Sig)
	}
}

func TestJSParser_TypedArrowFunction(t *testing.T) {
	// TypeScript arrow with a return type annotation: const f = (a: string): string => {
	src := `const greet = (name: string): string => {
    return "Hello, " + name;
};
`
	path := writeTempFile(t, src, ".ts")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Sig != "const greet = (name: string): string =>" {
		t.Errorf("unexpected sig: %q", chunks[0].Sig)
	}
}

func TestJSParser_TSExtension(t *testing.T) {
	src := `function hello(): string {
    return "world";
}
`
	// .ts extension should use the same JS parser
	path := writeTempFile(t, src, ".ts")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
}

// --- CSS parser ---

func TestCSSParser_SimpleRule(t *testing.T) {
	src := `.container {
    display: flex;
    gap: 1rem;
}
`
	path := writeTempFile(t, src, ".css")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Sig != ".container" {
		t.Errorf("unexpected sig: %q", chunks[0].Sig)
	}
	if chunks[0].Start != 1 || chunks[0].End != 4 {
		t.Errorf("expected lines 1-4, got %d-%d", chunks[0].Start, chunks[0].End)
	}
}

func TestCSSParser_MultipleRules(t *testing.T) {
	src := `.header {
    color: red;
}

.footer {
    color: blue;
}
`
	path := writeTempFile(t, src, ".css")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
}

// --- HTML parser ---

func TestHTMLParser_ScriptBlock(t *testing.T) {
	src := `<html>
<head></head>
<body>
<script>
  console.log("hello");
</script>
</body>
</html>
`
	path := writeTempFile(t, src, ".html")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk for <script> block")
	}
	found := false
	for _, c := range chunks {
		if c.Start == 4 && c.End == 6 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a chunk covering lines 4-6; got: %+v", chunks)
	}
}

// --- Markdown parser ---

func TestMarkdownParser_HeadingChunks(t *testing.T) {
	src := `# Title

Some intro text.

## Section One

Content of section one.

## Section Two

Content of section two.
`
	path := writeTempFile(t, src, ".md")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Sig != "# Title" {
		t.Errorf("unexpected sig[0]: %q", chunks[0].Sig)
	}
	if chunks[0].Start != 1 || chunks[0].End != 4 {
		t.Errorf("Title: expected lines 1-4, got %d-%d", chunks[0].Start, chunks[0].End)
	}
	if chunks[1].Sig != "## Section One" {
		t.Errorf("unexpected sig[1]: %q", chunks[1].Sig)
	}
	if chunks[2].Sig != "## Section Two" {
		t.Errorf("unexpected sig[2]: %q", chunks[2].Sig)
	}
}

func TestMarkdownParser_NoHeadings_SlidingWindow(t *testing.T) {
	// A markdown file with no headings but >50 lines falls back to sliding windows.
	var lines string
	for i := 0; i < 55; i++ {
		lines += "Just some prose without any headings.\n"
	}
	path := writeTempFile(t, lines, ".md")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected sliding window chunks for headingless markdown")
	}
	for _, c := range chunks {
		if c.Sig != "Sliding Window" {
			t.Errorf("expected 'Sliding Window' sig, got %q", c.Sig)
		}
	}
}

func TestMarkdownParser_MDXExtension(t *testing.T) {
	src := "# MDX Component\n\nSome MDX content.\n\n## Sub-section\n\nMore content.\n"
	path := writeTempFile(t, src, ".mdx")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
}

// --- Proto parser ---

func TestProtoParser_MessageAndService(t *testing.T) {
	src := `syntax = "proto3";

message SearchRequest {
  string query = 1;
  int32 page = 2;
}

service SearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
}
`
	path := writeTempFile(t, src, ".proto")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Sig != "message SearchRequest {" {
		t.Errorf("unexpected sig[0]: %q", chunks[0].Sig)
	}
	if chunks[0].Start != 3 || chunks[0].End != 6 {
		t.Errorf("SearchRequest: expected lines 3-6, got %d-%d", chunks[0].Start, chunks[0].End)
	}
	if chunks[1].Sig != "service SearchService {" {
		t.Errorf("unexpected sig[1]: %q", chunks[1].Sig)
	}
}

func TestProtoParser_Enum(t *testing.T) {
	src := `syntax = "proto3";

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}
`
	path := writeTempFile(t, src, ".proto")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Sig != "enum Status {" {
		t.Errorf("unexpected sig: %q", chunks[0].Sig)
	}
	if chunks[0].Start != 3 || chunks[0].End != 7 {
		t.Errorf("Status: expected lines 3-7, got %d-%d", chunks[0].Start, chunks[0].End)
	}
}

// --- Unsupported extensions ---

func TestUnsupportedExtension_ReturnsNil(t *testing.T) {
	src := "this is a plain text file\nwith no recognized structure\n"
	path := writeTempFile(t, src, ".txt")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if chunks != nil {
		t.Errorf("expected nil for unsupported extension, got %d chunks", len(chunks))
	}
}

// --- Sliding window fallback ---

func TestSlidingWindowFallback_LargeUnstructuredHTML(t *testing.T) {
	// An HTML file with no recognized tags but >50 lines falls back to sliding windows.
	var lines string
	for i := 0; i < 60; i++ {
		lines += "<p>Some paragraph content here.</p>\n"
	}
	path := writeTempFile(t, lines, ".html")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected sliding window chunks for large unstructured HTML file")
	}
	for _, c := range chunks {
		if c.Sig != "Sliding Window" {
			t.Errorf("expected Sig='Sliding Window', got %q", c.Sig)
		}
		if c.End-c.Start+1 > 50 {
			t.Errorf("window too large: lines %d-%d", c.Start, c.End)
		}
	}
	// Verify total coverage: last chunk must reach line 60.
	last := chunks[len(chunks)-1]
	if last.End != 60 {
		t.Errorf("last chunk End should be 60, got %d", last.End)
	}
}

func TestSlidingWindowFallback_ShortFileNoSliding(t *testing.T) {
	// A short unsupported-structure file (<= 50 lines) should return no chunks.
	src := "<p>Short file.</p>\n"
	path := writeTempFile(t, src, ".html")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Only 1 line — below the sliding window threshold of 50 lines.
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for short file, got %d", len(chunks))
	}
}

// --- C++ parser ---

func TestCPPParser_FunctionDefinition(t *testing.T) {
	src := `int add(int a, int b) {
    return a + b;
}
`
	path := writeTempFile(t, src, ".cpp")
	chunks, err := chunker.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Start != 1 || chunks[0].End != 3 {
		t.Errorf("expected lines 1-3, got %d-%d", chunks[0].Start, chunks[0].End)
	}
}

func TestCPPParser_AbsPathPreserved(t *testing.T) {
	src := "int foo(int x) {\n    return x;\n}\n"
	absPath := filepath.Join(t.TempDir(), "test.cpp")
	if err := os.WriteFile(absPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	chunks, err := chunker.ParseFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected 1 chunk")
	}
	if chunks[0].File != absPath {
		t.Errorf("expected File=%q, got %q", absPath, chunks[0].File)
	}
}
