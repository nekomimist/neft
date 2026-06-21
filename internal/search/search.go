package search

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/koron/gomigemo/embedict"
	"github.com/koron/gomigemo/migemo"
)

type Options struct {
	Query            string
	Roots            []string
	Extensions       []string
	Recursive        bool
	ManyThreshold    int
	SnippetsWhenMany int
	SnippetsWhenFew  int
}

type Result struct {
	Query string      `json:"query"`
	Files []FileMatch `json:"files"`
}

type FileMatch struct {
	Path       string        `json:"path"`
	Title      string        `json:"title"`
	Modified   time.Time     `json:"modified"`
	MatchCount int           `json:"match_count"`
	Snippets   []LineSnippet `json:"snippets"`
}

type LineSnippet struct {
	Line    int     `json:"line"`
	Text    string  `json:"text"`
	Matches []Range `json:"matches"`
}

type Range struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type candidate struct {
	path     string
	title    string
	modified time.Time
}

type fileScan struct {
	candidate
	snippets []LineSnippet
	count    int
}

func Run(opts Options) (Result, error) {
	opts = normalizeOptions(opts)
	candidates, err := collectCandidates(opts.Roots, opts.Recursive, opts.Extensions)
	if err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(opts.Query) == "" {
		return Result{Query: opts.Query, Files: recentFiles(candidates)}, nil
	}

	matchers, err := compileMatchers(opts.Query)
	if err != nil {
		return Result{}, err
	}
	scans := make([]fileScan, 0)
	for _, c := range candidates {
		scan, err := scanFile(c, matchers, opts.SnippetsWhenFew)
		if err != nil {
			continue
		}
		if scan.count > 0 {
			scans = append(scans, scan)
		}
	}
	sort.Slice(scans, func(i, j int) bool {
		if scans[i].modified.Equal(scans[j].modified) {
			return scans[i].path < scans[j].path
		}
		return scans[i].modified.After(scans[j].modified)
	})

	limit := opts.SnippetsWhenFew
	if len(scans) >= opts.ManyThreshold {
		limit = opts.SnippetsWhenMany
	}
	files := make([]FileMatch, 0, len(scans))
	for _, scan := range scans {
		snippets := scan.snippets
		if len(snippets) > limit {
			snippets = snippets[:limit]
		}
		files = append(files, FileMatch{
			Path:       scan.path,
			Title:      scan.title,
			Modified:   scan.modified,
			MatchCount: scan.count,
			Snippets:   snippets,
		})
	}
	return Result{Query: opts.Query, Files: files}, nil
}

func normalizeOptions(opts Options) Options {
	opts.Extensions = normalizeExtensions(opts.Extensions)
	if opts.ManyThreshold <= 0 {
		opts.ManyThreshold = 50
	}
	if opts.SnippetsWhenMany <= 0 {
		opts.SnippetsWhenMany = 1
	}
	if opts.SnippetsWhenFew <= 0 {
		opts.SnippetsWhenFew = 5
	}
	return opts
}

func normalizeExtensions(extensions []string) []string {
	if len(extensions) == 0 {
		extensions = []string{"org"}
	}
	seen := map[string]bool{}
	normalized := make([]string, 0, len(extensions))
	for _, ext := range extensions {
		ext = strings.TrimSpace(ext)
		ext = strings.TrimPrefix(ext, ".")
		ext = strings.ToLower(ext)
		if ext == "" || seen[ext] {
			continue
		}
		seen[ext] = true
		normalized = append(normalized, ext)
	}
	if len(normalized) == 0 {
		return []string{"org"}
	}
	return normalized
}

func collectCandidates(roots []string, recursive bool, extensions []string) ([]candidate, error) {
	allowedExtensions := extensionSet(extensions)
	seen := map[string]bool{}
	var candidates []candidate
	for _, root := range roots {
		root = filepath.Clean(root)
		info, err := os.Stat(root)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if matchesExtension(root, allowedExtensions) {
				candidates = appendCandidate(candidates, seen, root, info)
			}
			continue
		}
		if recursive {
			err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if entry.IsDir() {
					return nil
				}
				if !matchesExtension(path, allowedExtensions) {
					return nil
				}
				info, err := entry.Info()
				if err != nil {
					return nil
				}
				candidates = appendCandidate(candidates, seen, path, info)
				return nil
			})
		} else {
			entries, readErr := os.ReadDir(root)
			if readErr != nil {
				return nil, readErr
			}
			for _, entry := range entries {
				if entry.IsDir() || !matchesExtension(entry.Name(), allowedExtensions) {
					continue
				}
				path := filepath.Join(root, entry.Name())
				info, err := entry.Info()
				if err != nil {
					continue
				}
				candidates = appendCandidate(candidates, seen, path, info)
			}
			err = nil
		}
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].path < candidates[j].path
	})
	return candidates, nil
}

func extensionSet(extensions []string) map[string]bool {
	allowed := map[string]bool{}
	for _, ext := range normalizeExtensions(extensions) {
		allowed[ext] = true
	}
	return allowed
}

func matchesExtension(path string, allowed map[string]bool) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	ext = strings.ToLower(ext)
	return allowed[ext]
}

func appendCandidate(candidates []candidate, seen map[string]bool, path string, info os.FileInfo) []candidate {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if seen[abs] {
		return candidates
	}
	seen[abs] = true
	return append(candidates, candidate{
		path:     abs,
		title:    displayTitle(abs),
		modified: info.ModTime(),
	})
}

func recentFiles(candidates []candidate) []FileMatch {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].modified.Equal(candidates[j].modified) {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].modified.After(candidates[j].modified)
	})
	files := make([]FileMatch, 0, len(candidates))
	for _, c := range candidates {
		files = append(files, FileMatch{
			Path:     c.path,
			Title:    c.title,
			Modified: c.modified,
		})
	}
	return files
}

func compileMatchers(query string) ([]*regexp.Regexp, error) {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return nil, errors.New("query has no terms")
	}
	dict, err := embedict.Load()
	if err != nil {
		return nil, err
	}
	matchers := make([]*regexp.Regexp, 0, len(terms))
	for _, term := range terms {
		re, err := migemo.Compile(dict, term)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, re)
	}
	return matchers, nil
}

func scanFile(c candidate, matchers []*regexp.Regexp, maxSnippets int) (fileScan, error) {
	file, err := os.Open(c.path)
	if err != nil {
		return fileScan{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	lineNumber := 0
	count := 0
	var snippets []LineSnippet
	for scanner.Scan() {
		lineNumber++
		text := scanner.Text()
		ranges, ok := matchLine(text, matchers)
		if !ok {
			continue
		}
		count++
		if len(snippets) < maxSnippets {
			snippets = append(snippets, LineSnippet{
				Line:    lineNumber,
				Text:    strings.TrimSpace(text),
				Matches: ranges,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return fileScan{}, err
	}
	return fileScan{candidate: c, snippets: snippets, count: count}, nil
}

func matchLine(text string, matchers []*regexp.Regexp) ([]Range, bool) {
	var ranges []Range
	for _, re := range matchers {
		loc := re.FindStringIndex(text)
		if loc == nil {
			return nil, false
		}
		ranges = append(ranges, Range{
			Start: utf8.RuneCountInString(text[:loc[0]]),
			End:   utf8.RuneCountInString(text[:loc[1]]),
		})
	}
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].Start == ranges[j].Start {
			return ranges[i].End < ranges[j].End
		}
		return ranges[i].Start < ranges[j].Start
	})
	return ranges, true
}

func displayTitle(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	parts := strings.Split(base, "--")
	if len(parts) > 1 && parts[1] != "" {
		return strings.ReplaceAll(parts[1], "-", " ")
	}
	return base
}
