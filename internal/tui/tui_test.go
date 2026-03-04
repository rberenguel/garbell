package tui

import (
	"reflect"
	"testing"
)

func TestOSCommonPrefix(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{"apple", "app", "application"}, "app"},
		{[]string{"banana", "bandana", "band"}, "ban"},
		{[]string{"car", "dog"}, ""},
		{[]string{"test"}, "test"},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		result := osCommonPrefix(tt.input)
		if result != tt.expected {
			t.Errorf("osCommonPrefix(%v) = %v; want %v", tt.input, result, tt.expected)
		}
	}
}

func TestComplete(t *testing.T) {
	r := &REPL{
		paths: []string{
			"/Users/ruben/code/garbell/internal/tui/tui.go",
			"/Users/ruben/code/garbell/internal/search/paths.go",
			"/Users/ruben/code/garbell/cmd/garbell/main.go",
		},
	}

	// Test command completion
	matches, prefix := r.complete("u")
	if len(matches) != 1 || matches[0] != "use" || prefix != "u" {
		t.Errorf("expected ['use'], 'u', got %v, %v", matches, prefix)
	}

	matches, prefix = r.complete("s")
	expected := []string{"sf", "sl", "ss"}
	if !reflect.DeepEqual(matches, expected) || prefix != "s" {
		t.Errorf("expected %v, 's', got %v, %v", expected, matches, prefix)
	}

	// Test path completion
	matches, prefix = r.complete("fs /Users/ruben/code/garbell/internal/t")
	if len(matches) != 1 || matches[0] != "/Users/ruben/code/garbell/internal/tui/tui.go" || prefix != "/Users/ruben/code/garbell/internal/t" {
		t.Errorf("expected tui.go path, got %v, %v", matches, prefix)
	}

	// Test invalid path command
	matches, prefix = r.complete("use /Users")
	if len(matches) != 0 || prefix != "" {
		t.Errorf("expected no matches for 'use', got %v, %v", matches, prefix)
	}
}
