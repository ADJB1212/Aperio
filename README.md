<div align="center">

# Aperio

**A blazing-fast, zero-dependency CLI for beautiful file statistics**

_Stream files, count text metrics, and generate stunning tables, CSV, or JSON output_

[![Zero Dependencies](https://img.shields.io/badge/zero--deps-✨%20clean-brightgreen?style=for-the-badge)](.)
[![License: GPL-3.0+](https://img.shields.io/badge/License-GPL--3.0+-blue?style=for-the-badge)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](./)
[![Platform Support](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-lightgrey?style=for-the-badge)](./)

</div>

What it reports:

- File name and extension
- File kind (text or binary)
- Size (human-readable in table; raw bytes also available in CSV/JSON)
- Line, word, and character counts (UTF-8 rune count for text files)
- Last modified timestamp

No external packages. Standard library only.

---

## Why Aperio

- Blazing fast:
  - Streams files in large chunks and decodes UTF-8 runes directly
  - Counts words via a simple state machine, avoiding allocations
  - Concurrency with a configurable job limit
- Pleasant output:
  - Unicode table borders by default (ASCII optional)
  - Optional thousands-separator formatting for counts
  - Optional progress bar with smooth Unicode fractional blocks
- Robust:
  - Detects binary files up-front and skips expensive text analysis
  - Handles stdin or argv seamlessly
  - Machine-readable CSV and JSON modes

---

## Install

Homebrew:

```
brew install adjb1212/aperio/aperio
```

Build with Makefile (must have Go toolchain):

```
make build
```

Check version:

```
aperio --version
```

---

## Usage

Synopsis:

```
aperio [options] <file1> [file2] ...
# Or read paths from stdin:
find . -type f -name '*.go' | aperio [options]
```

Options:

- Sorting
  - `--sort` name|ext|size|lines|words|chars|modified (default: name)
  - `--desc, -r` reverse (descending)
- Output
  - `--format, -f` table (default), csv, json
  - `--plain` use ASCII borders for table
  - `--no-header` omit header row in CSV
  - `--commas, -c` format counts (lines, words, chars) with commas in table
- Totals and progress
  - `--sum, -s` show totals footer (size, lines, words, chars)
  - `--progress, -p` show a progress bar on stderr
- Performance
  - `--jobs, -j` maximum concurrent file analyses (default: number of CPUs)
- Other
  - `--version, -v` print version and exit

Notes:

- When `--format=csv` or `--format=json`, data is printed to stdout. The progress bar (if enabled) always goes to stderr.
- Individual file errors are shown per-row; they do not change the process exit code.

---

## Examples

Basic table:

```
aperio README.md LICENSE
```

With totals:

```
aperio --sum README.md LICENSE
```

Sort by size descending:

```
aperio --sort size -r --sum README.md LICENSE
```

Plain ASCII table:

```
aperio --plain README.md LICENSE
```

Progress bar and comma formatting:

```
find . -type f -name '*.go' | aperio -j 8 -p -c -s --sort lines -r
```

CSV:

```
aperio --format csv README.md LICENSE > stats.csv
# Without header:
aperio -f csv --no-header README.md LICENSE > stats_no_header.csv
```

JSON:

```
aperio --format json README.md LICENSE | jq .
```

---

## Output details

- Table columns: File, Ext, Kind, Size, Lines, Words, Chars, Modified
  - Unicode borders by default; ASCII with `--plain`
  - Counts optionally formatted with commas via `--commas`
  - Binary files show `Kind=binary` and `-` for counts
- CSV columns:
  - File, Ext, Kind, SizeBytes, Size, Lines, Words, Chars, Modified, Error
  - Unreadable paths report their error text in the `Error` column
- JSON:
  - Array of objects mirroring the same fields (including numeric `SizeBytes`)

---

## Performance

- Streaming analysis:
  - Reads in large (64 KiB) chunks
  - Decodes UTF-8 runes without line splitting
  - Tracks word boundaries using `unicode.IsSpace`
- Binary handling:
  - Sniffs the first bytes for NUL or invalid UTF-8; binary files skip text analysis
- Concurrency:
  - Limit concurrent analyses with `--jobs` for best throughput

Tips:

- Use stdin pipelines (e.g., `find`, `fd`, `rg -l`) for large file sets
- Tune `--jobs` to avoid I/O saturation (often equals CPU cores works well)
- Enable `--progress` for long runs; it prints to stderr and won’t corrupt stdout

---

## Exit codes

- 0: success (including “no files selected” from stdin)
- 1: usage or runtime error (e.g., no input, I/O failure writing output)
- 2: invalid flag value

Individual file errors are surfaced per-row and do not change the overall exit code.

---

## Counting details

- Lines: Number of newline-terminated lines plus a final line if the file does not end with a newline.
- Words: Counted using `unicode.IsSpace` to detect word boundaries.
- Chars: Counted as UTF-8 runes (not bytes).
- Size: Binary (IEC) units (KiB, MiB, …) with exact multiples shown without decimals.

---

## License

GPL-3.0-or-later. See LICENSE.
