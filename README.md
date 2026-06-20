# neft

neft is a Deft-inspired Emacs package for searching org note trees, including
Denote directories.  It uses a small external Go executable for fast recursive
search and migemo expansion.

## Features

- Dedicated `*neft*` buffer with a query row at the top.
- Multiple search roots via `neft-directories`.
- Recursive `.org` search by default.
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
```

Run `M-x neft`.

## Customization

- `neft-directories`: roots or `.org` files to search.
- `neft-recursive`: search child directories.
- `neft-many-results-threshold`: switch point for compact snippets.
- `neft-snippets-when-many`: snippets per file for broad matches.
- `neft-snippets-when-few`: snippets per file for narrow matches.

## CLI

The executable can be used directly:

```sh
neft search --query "kensaku memo" --root ~/notes --format json
```

The output is JSON containing matched files, snippet lines, line numbers, and
match ranges.

## License

MIT License.  See `LICENSE`.
