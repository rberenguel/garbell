package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"garbell/internal/indexer"
	"garbell/internal/search"
)

var rangeRegex = regexp.MustCompile(`(?m)^(\s*)(\d+-\d+)(:)`)

func colorizeRanges(s string) string {
	// \x1b[33m = yellow
	return rangeRegex.ReplaceAllString(s, "$1\x1b[33m$2\x1b[0m$3")
}

type REPL struct {
	workspace  string
	paths      []string
	history    []string
	historyPos int
}

func New() *REPL {
	return &REPL{
		history: make([]string, 0),
	}
}

func (r *REPL) Run() error {
	state, err := enableRawMode()
	if err != nil {
		fmt.Printf("Failed to enable raw mode: %v\n", err)
		return err
	}
	defer disableRawMode(state)

	var buf []rune
	cursor := 0
	r.historyPos = 0

	reader := bufio.NewReader(os.Stdin)

	drawPrompt := func() {
		ws := "no workspace"
		if r.workspace != "" {
			ws = filepath.Base(r.workspace)
		}
		// Clear line and draw prompt
		// \x1b[1;36m = bold cyan, \x1b[32m = green, \x1b[0m = reset
		fmt.Printf("\r\x1b[K\x1b[1;36mgarbell\x1b[0m (\x1b[32m%s\x1b[0m)> %s", ws, string(buf))
		// Move cursor to correct position
		if cursor < len(buf) {
			fmt.Printf("\x1b[%dD", len(buf)-cursor)
		}
	}

	drawPrompt()

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch b {
		case 3: // Ctrl+C
			if len(buf) == 0 {
				return nil
			}
			buf = nil
			cursor = 0
			fmt.Print("\r\n")
			drawPrompt()

		case 4: // Ctrl+D
			if len(buf) == 0 {
				fmt.Print("\r\n")
				return nil
			}

		case 127, 8: // DEL or Backspace
			if cursor > 0 {
				buf = append(buf[:cursor-1], buf[cursor:]...)
				cursor--
				drawPrompt()
			}

		case '\r', '\n':
			fmt.Print("\r\n")
			disableRawMode(state) // Disable raw mode before execution so normal output works

			cmdStr := strings.TrimSpace(string(buf))
			if cmdStr != "" {
				r.history = append(r.history, cmdStr)
				r.historyPos = len(r.history)
				r.execute(cmdStr)
			}

			enableRawMode() // Re-enable raw mode
			buf = nil
			cursor = 0
			drawPrompt()

		case '\t':
			completions, prefix := r.complete(string(buf))
			if len(completions) == 1 {
				// complete fully
				disableRawMode(state)
				rem := completions[0][len(prefix):]
				for _, char := range rem {
					buf = append(buf[:cursor], append([]rune{char}, buf[cursor:]...)...)
					cursor++
				}
				enableRawMode()
				drawPrompt()
			} else if len(completions) > 1 {
				// common prefix
				common := osCommonPrefix(completions)
				if len(common) > len(prefix) {
					rem := common[len(prefix):]
					for _, char := range rem {
						buf = append(buf[:cursor], append([]rune{char}, buf[cursor:]...)...)
						cursor++
					}
					drawPrompt()
				} else {
					// print candidates
					disableRawMode(state)
					fmt.Print("\r\n")
					fmt.Println(strings.Join(completions, "  "))
					enableRawMode()
					drawPrompt()
				}
			}

		case '\x1b':
			if b1, _ := reader.ReadByte(); b1 == '[' {
				if b2, _ := reader.ReadByte(); b2 != 0 {
					switch b2 {
					case 'A': // Up
						if r.historyPos > 0 {
							r.historyPos--
							buf = []rune(r.history[r.historyPos])
							cursor = len(buf)
							drawPrompt()
						}
					case 'B': // Down
						if r.historyPos < len(r.history)-1 {
							r.historyPos++
							buf = []rune(r.history[r.historyPos])
							cursor = len(buf)
							drawPrompt()
						} else if r.historyPos == len(r.history)-1 {
							r.historyPos++
							buf = nil
							cursor = 0
							drawPrompt()
						}
					case 'C': // Right
						if cursor < len(buf) {
							cursor++
							fmt.Print("\x1b[C")
						}
					case 'D': // Left
						if cursor > 0 {
							cursor--
							fmt.Print("\x1b[D")
						}
					}
				}
			}

		default:
			if b >= 32 { // printable
				buf = append(buf[:cursor], append([]rune{rune(b)}, buf[cursor:]...)...)
				cursor++
				drawPrompt()
			}
		}
	}
}

func (r *REPL) execute(input string) {
	args := strings.Fields(input)
	if len(args) == 0 {
		return
	}

	cmd := args[0]
	switch cmd {
	case "use":
		if len(args) < 2 {
			fmt.Println("Usage: use <path>")
			return
		}
		path := args[1]
		if !filepath.IsAbs(path) {
			wd, err := os.Getwd()
			if err == nil {
				path = filepath.Join(wd, path)
			}
		}
		r.workspace = filepath.Clean(path)
		r.refreshPaths()
		fmt.Printf("Workspace set to %s\n", r.workspace)

	case "index":
		if r.workspace == "" {
			fmt.Println("No workspace set. Use 'use <path>' first.")
			return
		}
		duration, err := indexer.GenerateIndex(r.workspace)
		if err != nil {
			fmt.Printf("Error generating index: %v\n", err)
			return
		}
		r.refreshPaths()
		// Round to milliseconds for a cleaner display, or just print the native duration
		fmt.Printf("Index generated successfully in %s.\n", duration.Round(time.Millisecond))

	case "fs":
		if r.workspace == "" {
			fmt.Println("No workspace set.")
			return
		}
		if len(args) < 2 {
			fmt.Println("Usage: fs <filepath|dir>")
			return
		}
		skel, err := search.FileSkeleton(r.workspace, args[1])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Print(colorizeRanges(skel))

	case "rc":
		if r.workspace == "" {
			fmt.Println("No workspace set.")
			return
		}
		if len(args) < 3 {
			fmt.Println("Usage: rc <filepath> <line_number>")
			return
		}
		lineNum, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Println("Invalid line number")
			return
		}
		body, err := search.ReadChunkBlock(r.workspace, args[1], lineNum)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Print(body)

	case "sl":
		if r.workspace == "" {
			fmt.Println("No workspace set.")
			return
		}
		if len(args) < 2 {
			fmt.Println("Usage: sl <query>")
			return
		}
		query := strings.Join(args[1:], " ")
		bodies, err := search.SearchLexical(r.workspace, query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		for i, body := range bodies {
			fmt.Println(strings.TrimSpace(body))
			if i < len(bodies)-1 {
				fmt.Println("---")
			}
		}

	case "fu":
		if r.workspace == "" {
			fmt.Println("No workspace set.")
			return
		}
		if len(args) < 2 {
			fmt.Println("Usage: fu <symbol>")
			return
		}
		symbol := args[1]
		sigs, err := search.FindUsages(r.workspace, symbol)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		for _, sig := range sigs {
			fmt.Println(sig)
		}

	case "ei":
		if r.workspace == "" {
			fmt.Println("No workspace set.")
			return
		}
		if len(args) < 2 {
			fmt.Println("Usage: ei <filepath>")
			return
		}
		relPath := args[1]
		ext := filepath.Ext(relPath)
		iface, err := search.ExtractInterface(r.workspace, relPath, ext)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Print(iface)

	case "ss":
		if r.workspace == "" {
			fmt.Println("No workspace set.")
			return
		}
		if len(args) < 2 {
			fmt.Println("Usage: ss <pattern>")
			return
		}
		out, err := search.SearchSignature(r.workspace, args[1])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Print(colorizeRanges(out))

	case "lc":
		if r.workspace == "" {
			fmt.Println("No workspace set.")
			return
		}
		n := 10
		if len(args) >= 2 {
			if parsed, err := strconv.Atoi(args[1]); err == nil && parsed > 0 {
				n = parsed
			}
		}
		results, err := search.LargestChunks(r.workspace, n)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		for _, r := range results {
			fmt.Println(r)
		}

	case "ca":
		if r.workspace == "" {
			fmt.Println("No workspace set.")
			return
		}
		if len(args) < 3 {
			fmt.Println("Usage: ca <filepath> <line_number>")
			return
		}
		lineNum, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Println("Invalid line number")
			return
		}
		results, err := search.Callees(r.workspace, args[1], lineNum)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		for _, r := range results {
			fmt.Println(r)
		}

	case "dep":
		if r.workspace == "" {
			fmt.Println("No workspace set.")
			return
		}
		if len(args) < 2 {
			fmt.Println("Usage: dep <filepath>")
			return
		}
		results, err := search.Dependents(r.workspace, args[1])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		for _, r := range results {
			fmt.Println(r)
		}

	case "sf":
		if r.workspace == "" {
			fmt.Println("No workspace set.")
			return
		}
		if len(args) < 2 {
			fmt.Println("Usage: sf <sig>")
			return
		}
		sig := strings.Join(args[1:], " ")
		chunk, body, err := search.SearchFuzzy(r.workspace, sig)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if chunk.File != "" {
			fmt.Printf("Closest Match: %s (in %s)\n\n%s\n", chunk.Sig, chunk.File, strings.TrimSpace(body))
		} else {
			fmt.Println("No matches found.")
		}

	case "help", "?":
		if len(args) > 1 {
			printCommandHelp(args[1])
			return
		}
		fmt.Println("Commands:")
		fmt.Println("  use <path>         Set workspace and load index")
		fmt.Println("  index              Regenerate index for current workspace")
		fmt.Println("  fs <path>          file-skeleton")
		fmt.Println("  rc <file> <line>   read-chunk")
		fmt.Println("  sl <query>         search-lexical")
		fmt.Println("  fu <symbol>        find-usages")
		fmt.Println("  ei <file>          extract-interface")
		fmt.Println("  ss <pattern>       search-signature")
		fmt.Println("  lc [n]             largest-chunks")
		fmt.Println("  ca <file> <line>   callees")
		fmt.Println("  dep <file>         dependents")
		fmt.Println("  sf <sig>           search-fuzzy")
		fmt.Println("  exit, q            Quit REPL")
		fmt.Println("Type 'help <command>' for more info.")

	case "exit", "q":
		os.Exit(0)

	default:
		fmt.Printf("Unknown command '%s'. Type 'help' for commands.\n", cmd)
	}
	fmt.Println()
}

func printCommandHelp(cmd string) {
	switch cmd {
	case "use":
		fmt.Println("use <path>\n  Set the workspace directory and load its index.\n  Essential for auto-completion and index-aware commands.")
	case "index":
		fmt.Println("index\n  Regenerate the index for the current workspace.\n  Run this after making changes to the codebase.")
	case "fs", "file-skeleton":
		fmt.Println("fs <filepath|dir>\n  Returns the structural outline of a file or directory.\n  Shows line ranges and signatures for functions, classes, etc.")
	case "rc", "read-chunk":
		fmt.Println("rc <filepath> <line_number>\n  Reads the complete source block (e.g. function body) enclosing the given line number.")
	case "sl", "search-lexical":
		fmt.Println("sl <query>\n  Full-text regex search. Returns the full enclosing chunk (function body) for every match.")
	case "fu", "find-usages":
		fmt.Println("fu <symbol>\n  Finds exact usages of a symbol and returns only the calling function signatures.")
	case "ei", "extract-interface":
		fmt.Println("ei <filepath>\n  Reads a source file and returns its imports/includes and exported public declarations.")
	case "ss", "search-signature":
		fmt.Println("ss <pattern>\n  Regex search strictly against function/class signatures in the index, not file bodies.")
	case "lc", "largest-chunks":
		fmt.Println("lc [n]\n  Returns the top 'n' largest chunks by line count in the workspace. Default 10.")
	case "ca", "callees":
		fmt.Println("ca <filepath> <line_number>\n  Returns a list of function names called *within* the chunk enclosing the given line.")
	case "dep", "dependents":
		fmt.Println("dep <filepath>\n  Finds all files in the workspace that import or reference the given file.")
	case "sf", "search-fuzzy":
		fmt.Println("sf <signature>\n  Finds the closest matching function signature in the index using Levenshtein distance.")
	case "help", "?":
		fmt.Println("help [command]\n  Shows this help message. Pass a command to see details.")
	case "exit", "q":
		fmt.Println("exit\n  Quit the REPL.")
	default:
		fmt.Printf("No help available for '%s'\n", cmd)
	}
}

func (r *REPL) refreshPaths() {
	if r.workspace == "" {
		return
	}
	paths, _ := search.IndexedPaths(r.workspace)
	r.paths = paths
}

var commands = []string{"use", "index", "fs", "rc", "sl", "fu", "ei", "ss", "lc", "ca", "dep", "sf", "help", "exit", "q", "?"}

func (r *REPL) complete(input string) ([]string, string) {
	// First check if workspace is loaded at all
	parts := strings.Split(input, " ")

	if len(parts) == 1 {
		// Completing command
		var matches []string
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, parts[0]) {
				matches = append(matches, cmd)
			}
		}
		sort.Strings(matches)
		return matches, parts[0]
	}

	// Completing a file argument
	cmd := parts[0]
	isPathCommand := cmd == "fs" || cmd == "rc" || cmd == "ei" || cmd == "ca" || cmd == "dep"

	if !isPathCommand {
		return nil, ""
	}

	prefix := parts[len(parts)-1]
	var matches []string
	for _, p := range r.paths {
		if strings.HasPrefix(p, prefix) {
			matches = append(matches, p)
		}
	}

	return matches, prefix
}

func osCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		for len(prefix) > 0 {
			if strings.HasPrefix(s, prefix) {
				break
			}
			prefix = prefix[:len(prefix)-1]
		}
		if prefix == "" {
			break
		}
	}

	return prefix
}
