package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/nekomimist/neft/internal/search"
)

type rootFlags []string

func (r *rootFlags) String() string {
	return fmt.Sprint([]string(*r))
}

func (r *rootFlags) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "search" {
		fmt.Fprintln(os.Stderr, "usage: neft search --query QUERY --root DIR [--root DIR...] --format json")
		os.Exit(2)
	}
	if err := runSearch(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSearch(args []string) error {
	var roots rootFlags
	var query string
	var format string
	var recursive bool
	var manyThreshold int
	var snippetsWhenMany int
	var snippetsWhenFew int

	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&query, "query", "", "search query")
	fs.Var(&roots, "root", "root directory to search")
	fs.StringVar(&format, "format", "json", "output format")
	fs.BoolVar(&recursive, "recursive", true, "search child directories recursively")
	fs.IntVar(&manyThreshold, "many-threshold", 50, "file-count threshold for compact snippets")
	fs.IntVar(&snippetsWhenMany, "snippets-when-many", 1, "snippets per file for many results")
	fs.IntVar(&snippetsWhenFew, "snippets-when-few", 5, "snippets per file for few results")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if format != "json" {
		return fmt.Errorf("unsupported format: %s", format)
	}
	if len(roots) == 0 {
		return fmt.Errorf("at least one --root is required")
	}

	result, err := search.Run(search.Options{
		Query:            query,
		Roots:            []string(roots),
		Recursive:        recursive,
		ManyThreshold:    manyThreshold,
		SnippetsWhenMany: snippetsWhenMany,
		SnippetsWhenFew:  snippetsWhenFew,
	})
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(result)
}
