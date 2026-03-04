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
		if len(os.Args) < 3 {
			fmt.Println("Usage: garbell search-lexical <query>")
			os.Exit(1)
		}
		query := os.Args[2]
		bodies, err := search.SearchLexical(workspacePath, query)
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
		if len(os.Args) < 3 {
			fmt.Println("Usage: garbell file-skeleton <filepath>")
			os.Exit(1)
		}
		relPath := os.Args[2]
		skel, err := search.FileSkeleton(workspacePath, relPath)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(skel)

	case "read-chunk":
		if len(os.Args) < 4 {
			fmt.Println("Usage: garbell read-chunk <filepath> <line_number>")
			os.Exit(1)
		}
		relPath := os.Args[2]
		lineNum, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Println("Invalid line number")
			os.Exit(1)
		}
		body, err := search.ReadChunkBlock(workspacePath, relPath, lineNum)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(body)

	case "find-usages":
		if len(os.Args) < 3 {
			fmt.Println("Usage: garbell find-usages <symbol>")
			os.Exit(1)
		}
		symbol := os.Args[2]
		sigs, err := search.FindUsages(workspacePath, symbol)
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
		if len(os.Args) < 3 {
			fmt.Println("Usage: garbell search-signature <pattern>")
			os.Exit(1)
		}
		out, err := search.SearchSignature(workspacePath, os.Args[2])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(out)

	case "largest-chunks":
		n := 10
		if len(os.Args) >= 3 {
			if parsed, err := strconv.Atoi(os.Args[2]); err == nil && parsed > 0 {
				n = parsed
			}
		}
		results, err := search.LargestChunks(workspacePath, n)
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

func printUsage() {
	fmt.Println("Cercle v2 (garbell) - Daemonless Code Indexer & Search")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  index                                  Generate the interval map chunk index")
	fmt.Println("  search-lexical <query>                 Full-text search returning chunk bodies")
	fmt.Println("  search-fuzzy <signature>               Fuzzy search for a symbol signature")
	fmt.Println("  file-skeleton <filepath|dir>           View signatures and lines in a file or directory")
	fmt.Println("  read-chunk <filepath> <line_number>    Read the chunk block surrounding a line")
	fmt.Println("  find-usages <symbol>                   Find usages of a symbol (returns signatures)")
	fmt.Println("  extract-interface <filepath>           Extract imports and exported declarations")
	fmt.Println("  search-signature <pattern>             Search chunk signatures by regex (no file I/O)")
	fmt.Println("  largest-chunks [n]                     Show the n largest chunks by line count (default 10)")
	fmt.Println("  callees <filepath> <line_number>       Show functions called by the chunk at this line")
	fmt.Println("  dependents <filepath>                  Show files that import this file")
}
