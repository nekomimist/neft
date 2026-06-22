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
- Multiple search roots via `neft-directories`.
- Recursive `.org` search by default, with configurable file extensions.
- Space-separated AND terms.
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
- `neft-many-results-threshold`: switch point for compact snippets.
- `neft-snippets-when-many`: snippets per file for broad matches.
- `neft-snippets-when-few`: snippets per file for narrow matches.
- `neft-restore-window-configuration`: restore the previous window layout when
  quitting or killing the neft buffer.

## CLI

The executable can be used directly:

```sh
neft search --query "kensaku memo" --root ~/notes --extension org --format json
```

The output is JSON containing matched files, snippet lines, line numbers, and
match ranges.

## License

MIT License.  See `LICENSE`.
