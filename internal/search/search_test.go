package search

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
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
	if result.Files[0].Title != "alpha note" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
	if !reflect.DeepEqual(result.Files[0].Tags, []string{"tag"}) {
		t.Fatalf("tags = %#v", result.Files[0].Tags)
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

func TestRunPrefersOrgTitleMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note__tag.org"), "#+title: Mixed CASE: A+B!\nneedle\n")

	result, err := Run(Options{
		Query:     "needle",
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if result.Files[0].Title != "Mixed CASE: A+B!" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
	if !reflect.DeepEqual(result.Files[0].Tags, []string{"tag"}) {
		t.Fatalf("tags = %#v", result.Files[0].Tags)
	}
}

func TestRunPrefersOrgFileTagsOverFilenameTags(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note__filename.org"), "#+filetags: :Org:Tag:\nneedle\n")

	result, err := Run(Options{
		Query:     "needle",
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if !reflect.DeepEqual(result.Files[0].Tags, []string{"Org", "Tag"}) {
		t.Fatalf("tags = %#v", result.Files[0].Tags)
	}
}

func TestRunFallsBackToFilenameTagsWhenOrgFileTagsMissing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note__one_two.org"), "needle\n")

	result, err := Run(Options{
		Query:     "needle",
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if !reflect.DeepEqual(result.Files[0].Tags, []string{"one", "two"}) {
		t.Fatalf("tags = %#v", result.Files[0].Tags)
	}
}

func TestRunTagQueryFiltersByFileTags(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note__one_two.org"), "needle\n")
	writeFile(t, filepath.Join(root, "20260102T000000--beta-note__one.org"), "needle\n")
	writeFile(t, filepath.Join(root, "20260103T000000--gamma-note__two.org"), "needle\n")

	result, err := Run(Options{
		Query:     ":one:two:",
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1: %#v", len(result.Files), result.Files)
	}
	if result.Files[0].Title != "alpha note" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
	if len(result.Files[0].Snippets) != 0 || result.Files[0].MatchCount != 0 {
		t.Fatalf("tag-only result should have no snippets or matches: %#v", result.Files[0])
	}
}

func TestRunMixedTextAndTagQueryRequiresBoth(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note__tag.org"), "needle\n")
	writeFile(t, filepath.Join(root, "20260102T000000--beta-note__tag.org"), "other\n")
	writeFile(t, filepath.Join(root, "20260103T000000--gamma-note__other.org"), "needle\n")

	result, err := Run(Options{
		Query:     "needle :tag:",
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1: %#v", len(result.Files), result.Files)
	}
	if result.Files[0].Title != "alpha note" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
	if result.Files[0].MatchCount != 1 {
		t.Fatalf("match count = %d, want 1", result.Files[0].MatchCount)
	}
}

func TestRunTagQueryMatchesCaseInsensitively(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note__Work.org"), "needle\n")

	result, err := Run(Options{
		Query:     ":work:",
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
}

func TestRunEscapedTagTokenSearchesLiteralText(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note.org"), "literal :tag:\n")
	writeFile(t, filepath.Join(root, "20260102T000000--beta-note__tag.org"), "other\n")

	result, err := Run(Options{
		Query:     `\:tag:`,
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1: %#v", len(result.Files), result.Files)
	}
	if result.Files[0].Title != "alpha note" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
}

func TestRunEscapedBackslashSearchesLiteralBackslashTag(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note.org"), `literal \:tag:`+"\n")
	writeFile(t, filepath.Join(root, "20260102T000000--beta-note.org"), "literal :tag:\n")

	result, err := Run(Options{
		Query:     `\\:tag:`,
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1: %#v", len(result.Files), result.Files)
	}
	if result.Files[0].Title != "alpha note" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
}

func TestRunRecognizesOrgTitleCaseInsensitivelyWithLeadingWhitespace(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note.org"), "  #+TITLE: Upper Title\nneedle\n")

	result, err := Run(Options{
		Query:     "needle",
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if result.Files[0].Title != "Upper Title" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
}

func TestRunFallsBackToFilenameWhenOrgTitleIsMissingOrBlank(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note.org"), "#+title:   \nneedle\n")

	result, err := Run(Options{
		Query:     "needle",
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if result.Files[0].Title != "alpha note" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
}

func TestRunCanUseFilenameTitleMode(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note.org"), "#+title: Org Title\nneedle\n")

	result, err := Run(Options{
		Query:       "needle",
		Roots:       []string{root},
		Recursive:   true,
		TitleSource: TitleSourceFilename,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if result.Files[0].Title != "alpha note" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
}

func TestRunEmptyQueryUsesOrgTitleMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "20260101T000000--alpha-note.org"), "#+title: Recent Title\n")

	result, err := Run(Options{Roots: []string{root}, Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if result.Files[0].Title != "Recent Title" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
}

func TestRunUsesOrgTitleForConfiguredTextExtensions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "plain.txt"), "#+title: Plain Text Title\nneedle\n")

	result, err := Run(Options{
		Query:      "needle",
		Roots:      []string{root},
		Extensions: []string{"txt"},
		Recursive:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if result.Files[0].Title != "Plain Text Title" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
}

func TestRunPreservesSnippetLeadingWhitespace(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "indented.org"), "  needle\n")

	result, err := Run(Options{
		Query:            "needle",
		Roots:            []string{root},
		Recursive:        true,
		SnippetsWhenFew:  5,
		SnippetsWhenMany: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || len(result.Files[0].Snippets) != 1 {
		t.Fatalf("result = %#v", result)
	}
	snippet := result.Files[0].Snippets[0]
	if snippet.Text != "  needle" {
		t.Fatalf("text = %q, want leading spaces preserved", snippet.Text)
	}
	if len(snippet.Matches) != 1 || snippet.Matches[0].Start != 2 || snippet.Matches[0].End != 8 {
		t.Fatalf("matches = %#v, want range 2..8", snippet.Matches)
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

func TestRunSortsMatchesByModifiedTimeDescending(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "old.org")
	newPath := filepath.Join(root, "new.org")
	writeFileWithModTime(t, oldPath, "needle\n", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	writeFileWithModTime(t, newPath, "needle\n", time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))

	result, err := Run(Options{
		Query:     "needle",
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("files = %d, want 2", len(result.Files))
	}
	if filepath.Base(result.Files[0].Path) != "new.org" {
		t.Fatalf("first path = %q, want new.org", result.Files[0].Path)
	}
	if filepath.Base(result.Files[1].Path) != "old.org" {
		t.Fatalf("second path = %q, want old.org", result.Files[1].Path)
	}
}

func TestRunMatchesCaseInsensitivelyByDefault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "mixed.org"), "Needle\n")

	result, err := Run(Options{
		Query:     "needle",
		Roots:     []string{root},
		Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	matches := result.Files[0].Snippets[0].Matches
	if len(matches) != 1 || matches[0].Start != 0 || matches[0].End != 6 {
		t.Fatalf("matches = %#v, want range 0..6", matches)
	}
}

func TestRunCanMatchCaseSensitively(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "mixed.org"), "Needle\n")

	result, err := Run(Options{
		Query:         "needle",
		Roots:         []string{root},
		Recursive:     true,
		CaseSensitive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("files = %d, want 0: %#v", len(result.Files), result.Files)
	}
}

func TestRunSearchesConfiguredTextExtensions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "one.org"), "needle org\n")
	writeFile(t, filepath.Join(root, "two.txt"), "needle txt\n")
	writeFile(t, filepath.Join(root, "three.md"), "needle md\n")

	result, err := Run(Options{
		Query:      "needle",
		Roots:      []string{root},
		Extensions: []string{"org", "txt"},
		Recursive:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("files = %d, want 2: %#v", len(result.Files), result.Files)
	}
	paths := map[string]bool{}
	for _, file := range result.Files {
		paths[filepath.Base(file.Path)] = true
	}
	if !paths["one.org"] || !paths["two.txt"] || paths["three.md"] {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestRunSearchesConfiguredDirectFileRoot(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "plain.txt")
	writeFile(t, file, "needle\n")

	result, err := Run(Options{
		Query:      "needle",
		Roots:      []string{file},
		Extensions: []string{".txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if result.Files[0].Title != "plain" {
		t.Fatalf("title = %q", result.Files[0].Title)
	}
}

func TestRunMatchesConfiguredExtensionsCaseInsensitively(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "upper.TXT"), "needle\n")

	result, err := Run(Options{
		Query:      "needle",
		Roots:      []string{root},
		Extensions: []string{"txt"},
		Recursive:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(result.Files))
	}
	if filepath.Base(result.Files[0].Path) != "upper.TXT" {
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

func writeFileWithModTime(t *testing.T, path string, content string, modTime time.Time) {
	t.Helper()
	writeFile(t, path, content)
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}
