# neft

neft is a Deft-inspired Emacs package for searching text note trees, including
Denote directories.  It uses a small external Go executable for fast recursive
search and migemo expansion.

## Features

- Dedicated `*neft*` buffer with a query row at the top.
- Full-window search session that restores the previous window layout on quit.
- Read-only result area, with editing limited to the query text.
- Query-row editing commands keep the prompt and result separator intact.
- `C-<down>` and `C-<up>` move between file results.
- `C-c C-s` toggles case-sensitive searching.
- Result headings show modified dates, note titles, and tags, preferring
  `#+title:` and `#+filetags:` metadata by default; moving point over a result
  shows its file path in the echo area.
- Multiple search roots via `neft-directories`.
- Recursive `.org` search by default, with configurable file extensions.
- Space-separated AND terms.
- Tag filters with `:tag:` syntax, including mixed text-and-tag searches such
  as `needle :work:`.
- Case-insensitive search by default, with an optional sensitive mode.
- Japanese migemo search handled by the external executable.
- Real-time result updates and match highlighting.
- Compact snippets when many files match, richer snippets when results narrow.

## Setup

Build the executable:

```sh
make build
```

Configure Emacs:

```elisp
(add-to-list 'load-path "/path/to/neft")
(require 'neft)
(setq neft-program "/path/to/neft/bin/neft")
(setq neft-directories '("~/notes" "~/work/notes"))
;; Optional: also search plain text notes.
(setq neft-file-extensions '("org" "txt"))
```

Run `M-x neft`.

## Customization

- `neft-directories`: roots or files to search.
- `neft-file-extensions`: file extensions to search.
- `neft-recursive`: search child directories.
- `neft-case-sensitive`: match case sensitively by default.
- `neft-use-org-title`: prefer `#+title:` metadata over filename-derived
  titles.
- `neft-many-results-threshold`: switch point for compact snippets.
- `neft-snippets-when-many`: snippets per file for broad matches.
- `neft-snippets-when-few`: snippets per file for narrow matches.
- `neft-restore-window-configuration`: restore the previous window layout when
  quitting or killing the neft buffer.
- `neft-show-file-path-in-echo-area`: show the file path in the echo area when
  point moves over a result.
- `neft-compact-result-spacing`: omit blank lines between file results.

## CLI

The executable can be used directly:

```sh
neft search --query "kensaku memo" --root ~/notes --extension org --case-sensitive=false --use-org-title=true --format json
```

The output is JSON containing matched files, tags, snippet lines, line numbers,
and match ranges.

Search queries are split on whitespace.  A token like `:tag1:tag2:` filters
files to notes that have all listed tags, case-insensitively.  Text terms and
tag filters can be mixed, so `needle :work:` matches notes tagged `work` that
also contain `needle`.  To search for a literal tag-looking string in text,
escape it with a backslash, for example `\:work:`; use `\\:work:` to search for
a literal `\:work:`.

## License

MIT License.  See `LICENSE`.
