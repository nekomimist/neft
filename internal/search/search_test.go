package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunSearchesRecursiveOrgFilesWithMigemoAndANDTerms(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note__tag.org"), "* Alpha\n検索 test\n")
	writeFile(t, filepath.Join(root, "child", "20260102T000000--beta-note.org"), "* Beta\n検索 only\n")
	writeFile(t, filepath.Join(root, "ignore.txt"), "検索 test\n")

	result, err := Run(Options{
		Query:            "kensaku test",
		Roots:            []string{root},
		Recursive:        true,
		ManyThreshold:    50,
		SnippetsWhenMany: 1,
		SnippetsWhenFew:  5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1: %#v", len(result.Files), result.Files)
	}
	if result.Files[0].Title != "alpha note__tag" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
	if len(result.Files[0].Snippets) != 1 {
		t.Fatalf("snippets = %d, want 1", len(result.Files[0].Snippets))
	}
	if result.Files[0].Snippets[0].Line != 2 {
		t.Fatalf("line = %d, want 2", result.Files[0].Snippets[0].Line)
	}
	if len(result.Files[0].Snippets[0].Matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(result.Files[0].Snippets[0].Matches))
	}
}

func TestRunLimitsSnippetsWhenManyFilesMatch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "one.org"), "needle\nneedle\nneedle\n")
	writeFile(t, filepath.Join(root, "two.org"), "needle\nneedle\nneedle\n")

	result, err := Run(Options{
		Query:            "needle",
		Roots:            []string{root},
		Recursive:        true,
		ManyThreshold:    2,
		SnippetsWhenMany: 1,
		SnippetsWhenFew:  3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("files = %d, want 2", len(result.Files))
	}
	for _, file := range result.Files {
		if len(file.Snippets) != 1 {
			t.Fatalf("%s snippets = %d, want 1", file.Path, len(file.Snippets))
		}
		if file.MatchCount != 3 {
			t.Fatalf("%s match count = %d, want 3", file.Path, file.MatchCount)
		}
	}
}

func TestRunEmptyQueryReturnsRecentOrgFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "one.org"), "one\n")
	writeFile(t, filepath.Join(root, "two.txt"), "two\n")

	result, err := Run(Options{Roots: []string{root}, Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if filepath.Base(result.Files[0].Path) != "one.org" {
		t.Fatalf("path = %q", result.Files[0].Path)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
