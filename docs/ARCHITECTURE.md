# Architecture

neft has two parts:

- `neft.el`: Emacs UI, process orchestration, result rendering, and navigation.
- `cmd/neft`: external search executable implemented in Go.

The Emacs package owns the interactive experience.  It opens a dedicated
`*neft*` buffer, keeps the query on the first line, debounces edits, starts the
external executable asynchronously with `make-process`, and ignores stale
process results using a generation counter.  The search buffer disables
completion-at-point and Emacs 30 completion previews because the query row is a
purpose-built input field that is redrawn as search results arrive.

Only the query text after the case-mode search label is writable.  The prompt
and rendered results use read-only text properties and sticky boundaries so
normal editing commands cannot insert text outside the query marker range.
Query-row commands that would otherwise cross the newline, such as `kill-line`,
are remapped to operate only within the query marker range.

Rendered file headings show the modified date and display title, and carry
dedicated text properties for navigation.  File paths stay out of the visible
result list and are exposed through `help-echo` for standard hover help.
`neft-mode` also uses a buffer-local `post-command-hook` to show the file path
in the echo area when point moves over a result, so path visibility does not
depend on global idle help settings.  File-result spacing is controlled by the
Emacs UI; `neft-compact-result-spacing` omits the blank line between file
results without changing snippets within a file.
`forward-paragraph` and `backward-paragraph`, commonly bound to `C-<down>` and
`C-<up>`, are remapped to move by file result instead of by visual paragraph.

By default, `M-x neft` treats the search buffer as a temporary full-window
session.  It saves the current window configuration before displaying `*neft*`,
deletes other windows, and restores the saved configuration once when neft is
quit or when the neft buffer is killed.

The Go executable owns file discovery, migemo expansion, matching, snippet
selection, and JSON serialization.  It does not keep a daemon, index, cache, or
file watcher in the initial design.  Each search recursively scans configured
roots for files with configured extensions.

## Search Contract

Emacs calls:

```sh
neft search --query QUERY --root DIR --extension EXT --case-sensitive BOOL --use-org-title BOOL --format json
```

Multiple `--root` and `--extension` flags are accepted.  When no extension is
specified, the CLI searches `org` files.  JSON results include file path,
display title, modified time, total matching line count, snippet lines, line
numbers, and character ranges for highlighting.

Display titles prefer the first non-empty `#+title:` metadata line by default.
The metadata keyword is matched case-insensitively with leading whitespace
allowed, and the value preserves case and punctuation after trimming surrounding
whitespace.  When `--use-org-title=false` is specified, or no title metadata is
present, titles are derived from filenames using the Denote-style suffix after
`--` with hyphens rendered as spaces.

Queries are split on whitespace.  Every term must match the same line for that
line to become a snippet.  Matching is case-insensitive by default and can be
made case-sensitive through `--case-sensitive`; each term is expanded through
gomigemo before matching, so romanized Japanese queries can match Japanese
text.

## Result Density

Results are sorted by file modification time descending, with path order as the
tie-breaker.

When many files match, the CLI returns one snippet per file by default.  When
the number of matching files drops below `neft-many-results-threshold`, it
returns more snippets per file.  This keeps broad searches scan-friendly while
making narrow searches more informative.
