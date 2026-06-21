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
neft search --query QUERY --root DIR --extension EXT --format json
```

Multiple `--root` and `--extension` flags are accepted.  When no extension is
specified, the CLI searches `org` files.  JSON results include file path,
display title, modified time, total matching line count, snippet lines, line
numbers, and character ranges for highlighting.

Queries are split on whitespace.  Every term must match the same line for that
line to become a snippet.  Each term is expanded through gomigemo before
matching, so romanized Japanese queries can match Japanese text.

## Result Density

When many files match, the CLI returns one snippet per file by default.  When
the number of matching files drops below `neft-many-results-threshold`, it
returns more snippets per file.  This keeps broad searches scan-friendly while
making narrow searches more informative.
