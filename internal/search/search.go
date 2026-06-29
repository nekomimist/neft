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
	CaseSensitive    bool
	TitleSource      TitleSource
	ManyThreshold    int
	SnippetsWhenMany int
	SnippetsWhenFew  int
}

type TitleSource int

const (
	TitleSourceOrgTitle TitleSource = iota
	TitleSourceFilename
)

type Result struct {
	Query string      `json:"query"`
	Files []FileMatch `json:"files"`
}

type FileMatch struct {
	Path       string        `json:"path"`
	Title      string        `json:"title"`
	Tags       []string      `json:"tags"`
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
	tags     []string
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
	query := parseQuery(opts.Query)
	if query.empty() {
		candidates = readCandidateMetadata(candidates, opts.TitleSource)
		return Result{Query: opts.Query, Files: recentFiles(candidates)}, nil
	}

	var matchers []*regexp.Regexp
	if len(query.textTerms) > 0 {
		matchers, err = compileMatchers(query.textTerms, opts.CaseSensitive)
		if err != nil {
			return Result{}, err
		}
	}
	scans := make([]fileScan, 0)
	for _, c := range candidates {
		if len(query.tagTerms) > 0 {
			c = readCandidateMetadataForCandidate(c, opts.TitleSource)
			if !matchesTags(c.tags, query.tagTerms) {
				continue
			}
		}
		if len(matchers) == 0 {
			scans = append(scans, fileScan{candidate: c})
			continue
		}
		scan, err := scanFile(c, matchers, opts.SnippetsWhenFew, opts.TitleSource)
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
			Tags:       scan.tags,
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
		tags:     displayTags(abs),
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
			Tags:     c.tags,
			Modified: c.modified,
		})
	}
	return files
}

func compileMatchers(terms []string, caseSensitive bool) ([]*regexp.Regexp, error) {
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
		if !caseSensitive {
			re, err = regexp.Compile("(?i:" + re.String() + ")")
			if err != nil {
				return nil, err
			}
		}
		matchers = append(matchers, re)
	}
	return matchers, nil
}

func scanFile(c candidate, matchers []*regexp.Regexp, maxSnippets int, titleSource TitleSource) (fileScan, error) {
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
	title := c.title
	tags := c.tags
	foundTitle := false
	foundFileTags := false
	for scanner.Scan() {
		lineNumber++
		text := scanner.Text()
		if titleSource == TitleSourceOrgTitle && !foundTitle {
			if orgTitle, ok := orgTitleFromLine(text); ok {
				title = orgTitle
				foundTitle = true
			}
		}
		if !foundFileTags {
			if fileTags, ok := orgFileTagsFromLine(text); ok {
				tags = fileTags
				foundFileTags = true
			}
		}
		ranges, ok := matchLine(text, matchers)
		if !ok {
			continue
		}
		count++
		if len(snippets) < maxSnippets {
			snippets = append(snippets, LineSnippet{
				Line:    lineNumber,
				Text:    text,
				Matches: ranges,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return fileScan{}, err
	}
	c.title = title
	c.tags = tags
	return fileScan{candidate: c, snippets: snippets, count: count}, nil
}

func readCandidateMetadata(candidates []candidate, titleSource TitleSource) []candidate {
	for i := range candidates {
		candidates[i] = readCandidateMetadataForCandidate(candidates[i], titleSource)
	}
	return candidates
}

func readCandidateMetadataForCandidate(c candidate, titleSource TitleSource) candidate {
	file, err := os.Open(c.path)
	if err != nil {
		return c
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	foundTitle := false
	foundFileTags := false
	for scanner.Scan() {
		text := scanner.Text()
		if titleSource == TitleSourceOrgTitle && !foundTitle {
			if title, ok := orgTitleFromLine(text); ok {
				c.title = title
				foundTitle = true
			}
		}
		if !foundFileTags {
			if tags, ok := orgFileTagsFromLine(text); ok {
				c.tags = tags
				foundFileTags = true
			}
		}
		if (titleSource == TitleSourceFilename || foundTitle) && foundFileTags {
			break
		}
	}
	return c
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
	titlePart, _ := displayMetadata(path)
	return strings.ReplaceAll(titlePart, "-", " ")
}

func displayTags(path string) []string {
	_, tags := displayMetadata(path)
	return tags
}

func displayMetadata(path string) (string, []string) {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	parts := strings.Split(base, "--")
	titlePart := base
	if len(parts) > 1 && parts[1] != "" {
		titlePart = parts[1]
	}
	if before, after, ok := strings.Cut(titlePart, "__"); ok {
		return before, uniqueTags(strings.Split(after, "_"))
	}
	return titlePart, nil
}

func orgTitleFromLine(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(strings.ToLower(trimmed), "#+title:") {
		return "", false
	}
	title := strings.TrimSpace(trimmed[len("#+title:"):])
	if title == "" {
		return "", false
	}
	return title, true
}

func orgFileTagsFromLine(line string) ([]string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(strings.ToLower(trimmed), "#+filetags:") {
		return nil, false
	}
	value := strings.TrimSpace(trimmed[len("#+filetags:"):])
	tags, ok := parseColonTags(value)
	if !ok {
		return nil, false
	}
	return tags, true
}

type parsedQuery struct {
	textTerms []string
	tagTerms  []string
}

func (q parsedQuery) empty() bool {
	return len(q.textTerms) == 0 && len(q.tagTerms) == 0
}

func parseQuery(query string) parsedQuery {
	var parsed parsedQuery
	for _, field := range strings.Fields(query) {
		if tags, ok := parseColonTags(field); ok && !strings.HasPrefix(field, `\`) {
			parsed.tagTerms = append(parsed.tagTerms, tags...)
			continue
		}
		parsed.textTerms = append(parsed.textTerms, unescapeQueryTerm(field))
	}
	parsed.tagTerms = uniqueTags(parsed.tagTerms)
	return parsed
}

func parseColonTags(value string) ([]string, bool) {
	if len(value) < 3 || !strings.HasPrefix(value, ":") || !strings.HasSuffix(value, ":") {
		return nil, false
	}
	rawTags := strings.Split(strings.Trim(value, ":"), ":")
	tags := uniqueTags(rawTags)
	if len(tags) == 0 {
		return nil, false
	}
	return tags, true
}

func uniqueTags(tags []string) []string {
	seen := map[string]bool{}
	unique := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, tag)
	}
	return unique
}

func unescapeQueryTerm(term string) string {
	var builder strings.Builder
	for i := 0; i < len(term); i++ {
		if term[i] == '\\' && i+1 < len(term) && (term[i+1] == ':' || term[i+1] == '\\') {
			i++
		}
		builder.WriteByte(term[i])
	}
	return builder.String()
}

func matchesTags(fileTags []string, queryTags []string) bool {
	if len(queryTags) == 0 {
		return true
	}
	fileTagSet := map[string]bool{}
	for _, tag := range fileTags {
		fileTagSet[strings.ToLower(tag)] = true
	}
	for _, tag := range queryTags {
		if !fileTagSet[strings.ToLower(tag)] {
			return false
		}
	}
	return true
}
