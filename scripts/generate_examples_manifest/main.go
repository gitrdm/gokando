package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type ExampleEntry struct {
	File           string `json:"file"`
	Symbol         string `json:"symbol"`
	ExpectedOutput string `json:"expected_output"`
}

func main() {
	pkgPath := flag.String("pkg", ".", "package directory to scan (relative path)")
	outPath := flag.String("out", "examples_index.json", "output JSON file")
	flag.Parse()

	var entries []ExampleEntry
	fset := token.NewFileSet()

	err := filepath.WalkDir(*pkgPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		src, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return err
		}

		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			name := fd.Name.Name
			if !strings.HasPrefix(name, "Example") {
				continue
			}

			// Attempt to find an Output: comment block after the function
			endPos := fset.Position(fd.End()).Offset
			expected := extractOutputComment(src, endPos)

			relPath, _ := filepath.Rel(".", path)
			entries = append(entries, ExampleEntry{
				File:           relPath,
				Symbol:         name,
				ExpectedOutput: expected,
			})
		}

		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error scanning package: %v\n", err)
		os.Exit(2)
	}

	out, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding json: %v\n", err)
		os.Exit(2)
	}

	if err := ioutil.WriteFile(*outPath, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("wrote %d example entries to %s\n", len(entries), *outPath)
}

// extractOutputComment looks for a line containing "Output:" in the file
// after the given byte offset and returns the joined lines (without // prefixes).
func extractOutputComment(src []byte, startOffset int) string {
	if startOffset >= len(src) {
		return ""
	}
	r := bufio.NewReader(strings.NewReader(string(src[startOffset:])))
	scanner := bufio.NewScanner(r)
	var found bool
	var lines []string
	for i := 0; i < 200 && scanner.Scan(); i++ { // limit search to 200 lines
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !found {
			// look for a comment containing "Output:"
			if strings.HasPrefix(trimmed, "//") && strings.Contains(trimmed, "Output:") {
				found = true
				// capture any text after Output:
				idx := strings.Index(trimmed, "Output:")
				after := strings.TrimSpace(trimmed[idx+len("Output:"):])
				if after != "" {
					lines = append(lines, after)
				}
				continue
			}
		} else {
			// we are in the output block; accept consecutive // lines
			if strings.HasPrefix(trimmed, "//") {
				// strip // and possible leading spaces
				content := strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
				lines = append(lines, content)
				continue
			}
			// stop when a non-comment line appears
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return ""
	}
	return strings.Join(lines, "\n")
}
