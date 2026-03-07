package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"garbell/internal/indexer"
	"garbell/internal/search"
	"garbell/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Default to current directory
	workspacePath, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "index":
		if _, err := indexer.GenerateIndex(workspacePath); err != nil {
			fmt.Printf("Error generating index: %v\n", err)
			os.Exit(1)
		}

	case "search-lexical":
		args := os.Args[2:]
		fileFilter, args := extractFlag(args, "file")
		if len(args) == 0 {
			fmt.Println("Usage: garbell search-lexical <query> [--file <regex>]")
			os.Exit(1)
		}
		query := strings.Join(args, " ")
		bodies, err := search.SearchLexical(workspacePath, query, fileFilter)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for i, body := range bodies {
			fmt.Println(strings.TrimSpace(body))
			if i < len(bodies)-1 {
				fmt.Println("---")
			}
		}

	case "search-fuzzy":
		if len(os.Args) < 3 {
			fmt.Println("Usage: garbell search-fuzzy <signature>")
			os.Exit(1)
		}
		sig := os.Args[2]
		chunk, body, err := search.SearchFuzzy(workspacePath, sig)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if chunk.File != "" {
			fmt.Printf("Closest Match: %s (in %s)\n\n%s\n", chunk.Sig, chunk.File, strings.TrimSpace(body))
		} else {
			fmt.Println("No matches found.")
		}

	case "file-skeleton":
		args := os.Args[2:]
		fileFilter, args := extractFlag(args, "file")
		if len(args) == 0 {
			fmt.Println("Usage: garbell file-skeleton <filepath|dir> [--file <regex>]")
			os.Exit(1)
		}
		relPath := args[0]
		skel, err := search.FileSkeleton(workspacePath, relPath, fileFilter)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(skel)

	case "read-chunk":
		args := os.Args[2:]
		unsafe, args := extractBoolFlag(args, "unsafe")
		if len(args) < 2 {
			fmt.Println("Usage: garbell read-chunk <filepath> <line_number> [--unsafe]")
			os.Exit(1)
		}
		relPath := args[0]
		lineNum, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Println("Invalid line number")
			os.Exit(1)
		}
		if lineNum == -1 {
			body, err := search.ReadFullFile(workspacePath, relPath, unsafe)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Print(body)
		} else {
			body, err := search.ReadChunkBlock(workspacePath, relPath, lineNum)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Print(body)
		}

	case "find-usages":
		args := os.Args[2:]
		fileFilter, args := extractFlag(args, "file")
		if len(args) == 0 {
			fmt.Println("Usage: garbell find-usages <symbol> [--file <regex>]")
			os.Exit(1)
		}
		symbol := args[0]
		sigs, err := search.FindUsages(workspacePath, symbol, fileFilter)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for _, sig := range sigs {
			fmt.Println(sig)
		}

	case "extract-interface":
		if len(os.Args) < 3 {
			fmt.Println("Usage: garbell extract-interface <filepath>")
			os.Exit(1)
		}
		relPath := os.Args[2]
		ext := filepath.Ext(relPath)
		iface, err := search.ExtractInterface(workspacePath, relPath, ext)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(iface)

	case "search-signature":
		args := os.Args[2:]
		fileFilter, args := extractFlag(args, "file")
		if len(args) == 0 {
			fmt.Println("Usage: garbell search-signature <pattern> [--file <regex>]")
			os.Exit(1)
		}
		out, err := search.SearchSignature(workspacePath, args[0], fileFilter)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(out)

	case "largest-chunks":
		args := os.Args[2:]
		fileFilter, args := extractFlag(args, "file")
		n := 10
		if len(args) >= 1 {
			if parsed, err := strconv.Atoi(args[0]); err == nil && parsed > 0 {
				n = parsed
			}
		}
		results, err := search.LargestChunks(workspacePath, n, fileFilter)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for _, r := range results {
			fmt.Println(r)
		}

	case "callees":
		if len(os.Args) < 4 {
			fmt.Println("Usage: garbell callees <filepath> <line_number>")
			os.Exit(1)
		}
		lineNum, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Println("Invalid line number")
			os.Exit(1)
		}
		results, err := search.Callees(workspacePath, os.Args[2], lineNum)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for _, r := range results {
			fmt.Println(r)
		}

	case "dependents":
		if len(os.Args) < 3 {
			fmt.Println("Usage: garbell dependents <filepath>")
			os.Exit(1)
		}
		results, err := search.Dependents(workspacePath, os.Args[2])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for _, r := range results {
			fmt.Println(r)
		}

	case "search-related":
		args := os.Args[2:]
		fileFilter, args := extractFlag(args, "file")
		if len(args) == 0 {
			fmt.Println("Usage: garbell search-related <query> [--file <regex>]")
			os.Exit(1)
		}
		query := strings.Join(args, " ")
		bodies, err := search.SearchRelated(workspacePath, query, fileFilter)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for i, body := range bodies {
			fmt.Println(strings.TrimSpace(body))
			if i < len(bodies)-1 {
				fmt.Println("---")
			}
		}

	case "peek":
		if len(os.Args) < 4 {
			fmt.Println("Usage: garbell peek <filepath> <line_number> [radius]")
			os.Exit(1)
		}
		relPath := os.Args[2]
		lineNum, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Println("Invalid line number")
			os.Exit(1)
		}
		radius := 5
		if len(os.Args) >= 5 {
			if r, err := strconv.Atoi(os.Args[4]); err == nil && r > 0 {
				radius = r
			}
		}
		out, err := search.Peek(workspacePath, relPath, lineNum, radius)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(out)

	case "repl":
		r := tui.New()
		if err := r.Run(); err != nil {
			fmt.Printf("REPL error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// extractFlag removes a named flag and its value from args in a order-agnostic way.
// Supports both "--name value" and "--name=value" forms.
func extractFlag(args []string, name string) (value string, rest []string) {
	flag := "--" + name
	result := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == flag && i+1 < len(args) {
			value = args[i+1]
			i++ // skip the value token
		} else if strings.HasPrefix(args[i], flag+"=") {
			value = args[i][len(flag)+1:]
		} else {
			result = append(result, args[i])
		}
	}
	return value, result
}

// extractBoolFlag removes a boolean flag from args, returning whether it was present.
func extractBoolFlag(args []string, name string) (found bool, rest []string) {
	flag := "--" + name
	result := make([]string, 0, len(args))
	for _, a := range args {
		if a == flag {
			found = true
		} else {
			result = append(result, a)
		}
	}
	return found, result
}

func printUsage() {
	fmt.Println("Cercle v2 (garbell) - Daemonless Code Indexer & Search")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  index                                        Generate the interval map chunk index")
	fmt.Println("  search-lexical <query> [--file <regex>]      Full-text search; compact chunk list (use read-chunk for bodies)")
	fmt.Println("  search-fuzzy <signature>                     Fuzzy search for a symbol signature")
	fmt.Println("  file-skeleton <path> [--file <regex>]        View signatures in a file or directory")
	fmt.Println("  read-chunk <filepath> <line_number|'-1'>     Read chunk at line, or -1 for full file")
	fmt.Println("             [--unsafe]                        Force full-file read even if over the line limit")
	fmt.Println("  peek <filepath> <line_number> [radius]       Show lines around a target line (default radius: 5)")
	fmt.Println("  find-usages <symbol> [--file <regex>]        Find usages of a symbol (returns signatures)")
	fmt.Println("  extract-interface <filepath>                 Extract imports and exported declarations")
	fmt.Println("  search-signature <pattern> [--file <regex>]  Search chunk signatures by regex (no file I/O)")
	fmt.Println("  largest-chunks [n] [--file <regex>]          Show the n largest chunks by line count (default 10)")
	fmt.Println("  callees <filepath> <line_number>             Show functions called by the chunk at this line")
	fmt.Println("  dependents <filepath>                        Show files that import this file")
	fmt.Println("  search-related <query> [--file <regex>]      PPMI co-occurrence expansion + lexical search")
	fmt.Println()
	fmt.Println("  --file <regex>   Filter results to files whose path matches the regex (order-agnostic)")
}
