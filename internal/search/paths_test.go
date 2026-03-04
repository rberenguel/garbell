package search_test

import (
	"reflect"
	"testing"

	"garbell/internal/search"
)

func TestIndexedPaths(t *testing.T) {
	requireWorkspace(t)
	paths, err := search.IndexedPaths(testWorkspace)
	if err != nil {
		t.Fatalf("IndexedPaths failed: %v", err)
	}

	expected := []string{
		"hello.go",
		"main.js",
		"utils.js",
	}

	if !reflect.DeepEqual(paths, expected) {
		t.Errorf("IndexedPaths() returned %v, want %v", paths, expected)
	}
}
