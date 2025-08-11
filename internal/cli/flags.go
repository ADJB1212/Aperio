package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config captures all command-line options and resolved inputs for aperio.
type Config struct {
	// Behavior
	ShowSum     bool
	ShowVersion bool

	// Sorting
	SortBy string // name, ext, size, lines, words, chars, modified
	Desc   bool   // reverse

	// Output
	Format   string // table, csv, json
	NoHeader bool   // CSV only
	Plain    bool   // ASCII table

	// Performance
	Jobs     int
	Progress bool
	Commas   bool

	// Inputs
	Files []string
}

// Usage returns a concise usage string suitable for errors/help.
func Usage() string {
	return "Usage: aperio [options] <file1> [file2] …\n" +
		"   or: <producer> | aperio [options]   (read newline-delimited paths from stdin)"
}

var (
	validSortBy = map[string]struct{}{
		"name": {}, "ext": {}, "size": {}, "lines": {}, "words": {}, "chars": {}, "modified": {},
	}
	validFormat = map[string]struct{}{
		"table": {}, "csv": {}, "json": {},
	}
)

// UsageError indicates improper CLI usage or invalid flag values.
type UsageError struct {
	Msg string
}

func (e *UsageError) Error() string { return e.Msg }

// Parse parses flags from os.Args and resolves the input file list from args or stdin.
//
// Notes:
// - If --version is set, Files may be empty and no further validation is performed.
// - If no args are provided, and stdin is a pipe, newline-delimited paths are read from stdin.
// - On invalid flag values, a *UsageError is returned.
// - On absence of inputs (no args and no stdin), a *UsageError is returned.
func Parse() (Config, error) {
	return ParseArgs(os.Args[1:], os.Stdin)
}

// ParseArgs parses CLI arguments and reads stdin if needed, returning a Config.
func ParseArgs(args []string, stdin *os.File) (Config, error) {
	var cfg Config

	// Defaults
	cfg.SortBy = "name"
	cfg.Format = "table"
	cfg.Jobs = defaultJobs()

	fs := flag.NewFlagSet("aperio", flag.ContinueOnError)
	fs.SetOutput(new(strings.Builder)) // suppress default printing; caller formats errors

	// Primary flags
	fs.BoolVar(&cfg.ShowSum, "sum", false, "Show totals for size, lines, words, and chars")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "Print version and exit")
	fs.StringVar(&cfg.SortBy, "sort", cfg.SortBy, "Sort by: name, ext, size, lines, words, chars, modified")
	fs.BoolVar(&cfg.Desc, "desc", false, "Sort descending")
	fs.StringVar(&cfg.Format, "format", cfg.Format, "Output format: table, csv, json")
	fs.BoolVar(&cfg.NoHeader, "no-header", false, "Omit header row in CSV output")
	fs.BoolVar(&cfg.Plain, "plain", false, "Use plain ASCII table borders")
	fs.IntVar(&cfg.Jobs, "jobs", cfg.Jobs, "Maximum concurrent file analyses")
	fs.BoolVar(&cfg.Progress, "progress", false, "Show progress bar on stderr")
	fs.BoolVar(&cfg.Commas, "commas", false, "Format counts (lines, words, chars) with commas")

	// Aliases
	fs.BoolVar(&cfg.ShowSum, "s", cfg.ShowSum, "Alias for --sum")
	fs.BoolVar(&cfg.ShowVersion, "v", cfg.ShowVersion, "Alias for --version")
	fs.StringVar(&cfg.SortBy, "S", cfg.SortBy, "Alias for --sort")
	fs.BoolVar(&cfg.Desc, "r", cfg.Desc, "Alias for --desc")
	fs.StringVar(&cfg.Format, "f", cfg.Format, "Alias for --format")
	fs.BoolVar(&cfg.Plain, "p", cfg.Plain, "Alias for --plain")
	fs.IntVar(&cfg.Jobs, "j", cfg.Jobs, "Alias for --jobs")
	fs.BoolVar(&cfg.Progress, "P", cfg.Progress, "Alias for --progress")
	fs.BoolVar(&cfg.Commas, "c", cfg.Commas, "Alias for --commas")

	if err := fs.Parse(args); err != nil {
		return Config{}, &UsageError{Msg: Usage()}
	}

	// Early exit for version
	if cfg.ShowVersion {
		return cfg, nil
	}

	// Normalize and validate
	cfg.SortBy = strings.ToLower(cfg.SortBy)
	if _, ok := validSortBy[cfg.SortBy]; !ok {
		return Config{}, &UsageError{Msg: fmt.Sprintf("Invalid --sort value: %q\n\n%s", cfg.SortBy, Usage())}
	}
	cfg.Format = strings.ToLower(cfg.Format)
	if _, ok := validFormat[cfg.Format]; !ok {
		return Config{}, &UsageError{Msg: fmt.Sprintf("Invalid --format value: %q\n\n%s", cfg.Format, Usage())}
	}
	if cfg.Jobs < 1 {
		cfg.Jobs = 1
	}

	// Resolve files from remaining args or from stdin when piped
	cfg.Files = fs.Args()
	if len(cfg.Files) == 0 {
		if stdin != nil {
			if hasPipedInput(stdin) {
				paths, err := readPathsFrom(stdin)
				if err != nil {
					return Config{}, err
				}
				if len(paths) == 0 {
					return Config{}, &UsageError{Msg: "No file paths provided via stdin"}
				}
				cfg.Files = paths
			} else {
				// No args and no piped stdin
				return Config{}, &UsageError{Msg: Usage()}
			}
		} else {
			return Config{}, &UsageError{Msg: Usage()}
		}
	}

	return cfg, nil
}

func defaultJobs() int {
	// Keep this minimal here; leave runtime.NumCPU() to the caller if desired.
	// We return 0 here to signal "use runtime.NumCPU()" if the caller prefers.
	// But for simplicity, set to 1 and let caller override if they want CPUs.
	return 1
}

func hasPipedInput(stdin *os.File) bool {
	info, err := stdin.Stat()
	if err != nil {
		return false
	}
	// If stdin is not a char device, then it has piped data or a file.
	return (info.Mode() & os.ModeCharDevice) == 0
}

func readPathsFrom(r *os.File) ([]string, error) {
	sc := bufio.NewScanner(r)
	// Increase scanner buffer for very long paths (rare but safe).
	const maxCapacity = 1024 * 1024 // 1 MiB
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, maxCapacity)

	var out []string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			out = append(out, line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// IsUsageError helps callers identify usage-related parse failures.
func IsUsageError(err error) bool {
	var ue *UsageError
	return errors.As(err, &ue)
}
