package search

import "regexp"

// compileFileFilter compiles a file path regex pattern used to restrict results to files
// whose relative path matches. Returns nil if pattern is empty (match all files).
func compileFileFilter(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	return regexp.Compile(pattern)
}

// matchesFileFilter reports whether the relative file path matches the compiled filter.
// A nil filter matches all files.
func matchesFileFilter(re *regexp.Regexp, file string) bool {
	if re == nil {
		return true
	}
	return re.MatchString(file)
}
