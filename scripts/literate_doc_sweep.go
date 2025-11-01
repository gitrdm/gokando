//go:build ignore

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// This tool scans a package and expands short godoc comments for exported
// declarations into a more literate, multi-paragraph form. It is conservative:
// - It only edits functions, types, and vars with exported names (starting with uppercase)
// - It only replaces comments that are empty or a single-line (no blank lines)
// - It preserves the original first line when present and uses it as the summary

func main() {
	pkgPath := flag.String("pkg", "pkg/minikanren", "package directory to sweep")
	dry := flag.Bool("dry", false, "dry run: print changes instead of writing files")
	flag.Parse()

	fset := token.NewFileSet()

	err := filepath.Walk(*pkgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
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

		modified := false

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				name := d.Name.Name
				if !isExported(name) {
					continue
				}
				// skip methods; only top-level funcs
				if d.Recv != nil {
					continue
				}
				if needsExpansion(d.Doc) {
					summary := firstLineOrDefault(d.Doc, name+" ...")
					d.Doc = makeLiterateComment(summary)
					modified = true
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						name := s.Name.Name
						if !isExported(name) {
							continue
						}
						if needsExpansion(d.Doc) {
							summary := firstLineOrDefault(d.Doc, name+" type")
							d.Doc = makeLiterateComment(summary)
							modified = true
						}
					case *ast.ValueSpec:
						for _, id := range s.Names {
							name := id.Name
							if !isExported(name) {
								continue
							}
							if needsExpansion(d.Doc) {
								summary := firstLineOrDefault(d.Doc, name+" variable")
								d.Doc = makeLiterateComment(summary)
								modified = true
							}
						}
					}
				}
			}
		}

		if modified {
			var sb strings.Builder
			if err := printer.Fprint(&sb, fset, file); err != nil {
				return err
			}
			newSrc := sb.String()
			if *dry {
				fmt.Printf("would modify: %s\n", path)
			} else {
				if err := ioutil.WriteFile(path, []byte(newSrc), 0644); err != nil {
					return err
				}
				fmt.Printf("modified: %s\n", path)
			}
		}

		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	r := rune(name[0])
	return r >= 'A' && r <= 'Z'
}

func needsExpansion(cg *ast.CommentGroup) bool {
	if cg == nil {
		return true
	}
	// if comment contains a blank line, assume it's already multi-paragraph
	text := cg.Text()
	if strings.Contains(text, "\n\n") {
		return false
	}
	// if it has more than one line, leave it alone
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) > 1 {
		return false
	}
	return true
}

func firstLineOrDefault(cg *ast.CommentGroup, def string) string {
	if cg == nil {
		return def
	}
	text := strings.TrimSpace(cg.Text())
	if text == "" {
		return def
	}
	lines := strings.Split(text, "\n")
	return strings.TrimSpace(lines[0])
}

func makeLiterateComment(summary string) *ast.CommentGroup {
	// Build a multi-paragraph comment using the summary as the first line
	paragraphs := []string{
		summary + "",
		"",
		"Description: \nThis function/type/variable is part of the public API. " +
			"The comment here was expanded automatically to provide a more readable, " +
			"literate-style documentation that appears in godoc. Review and edit as needed to add examples or more domain-specific detail.",
		"",
		"Example:\n\t// TODO: replace with a short, focused example demonstrating typical usage.",
		"",
		"See also: other related API in this package.",
	}

	var comments []*ast.Comment
	for _, p := range paragraphs {
		for _, line := range strings.Split(p, "\n") {
			comments = append(comments, &ast.Comment{Text: "// " + strings.TrimRight(line, "\r")})
		}
	}
	return &ast.CommentGroup{List: comments}
}
