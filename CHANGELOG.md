# Changelog

## Unreleased

- Initial neft package with Emacs search UI and Go-based org search executable.
- Add migemo-backed space-separated AND search.
- Add recursive multi-root `.org` discovery and JSON CLI output.
- Add full-window neft sessions that restore the previous window layout on quit.
- Add configurable search file extensions, defaulting to `.org`.
- Fix match highlighting for indented snippet lines.
- Disable completion previews in the neft search buffer.
- Make the prompt and search results read-only.
- Prevent `kill-line` on an empty query from briefly deleting the search row
  newline.
- Add `C-<down>` and `C-<up>` file-result navigation.
- Show only note titles in result headings, with file paths available through
  hover help.
- Show result file paths in the echo area when point moves over a result,
  independent of global idle help settings.
- Match case-insensitively by default, with `C-c C-s` and `--case-sensitive`
  for sensitive searches.
- Prefix result headings with each file's modified date.
- Add `neft-compact-result-spacing` to omit blank lines between file results.
- Prefer `#+title:` metadata for result headings by default, with
  `neft-use-org-title` and `--use-org-title=false` to restore filename-derived
  titles.
