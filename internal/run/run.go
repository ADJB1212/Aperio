package run

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/ADJB1212/Aperio/internal/analyze"
	"github.com/ADJB1212/Aperio/internal/cli"
	"github.com/ADJB1212/Aperio/internal/ui/progress"
	"github.com/ADJB1212/Aperio/internal/util"
)

// Run coordinates the full aperio flow based on CLI flags.
// It returns a process exit code (0 = success).
func Run(version string) int {
	cfg, err := cli.Parse()
	if err != nil {
		// Differentiate invalid flag values from generic usage errors when possible.
		msg := err.Error()
		if strings.HasPrefix(msg, "Invalid --sort") || strings.HasPrefix(msg, "Invalid --format") {
			fmt.Fprintln(os.Stderr, msg)
			return 2
		}
		fmt.Fprintln(os.Stderr, msg)
		return 1
	}

	if cfg.ShowVersion {
		fmt.Println(version)
		return 0
	}

	files := cfg.Files

	// Concurrency limit
	jobs := cfg.Jobs
	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}
	if jobs < 1 {
		jobs = 1
	}

	// Analyze files concurrently
	results := make(chan analyze.FileStats, len(files))
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)

	for _, path := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer func() { <-sem }()
			analyze.AnalyzeFile(p, results, &wg)
		}(path)
	}

	// Close results when done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect with optional progress
	stats := make([]analyze.FileStats, 0, len(files))
	processed := 0
	var bar *progress.Bar
	if cfg.Progress && len(files) > 0 {
		bar = progress.New(os.Stderr, 40)
		bar.Render(processed, len(files))
	}
	for fs := range results {
		stats = append(stats, fs)
		if bar != nil {
			processed++
			bar.Render(processed, len(files))
		}
	}
	if bar != nil {
		bar.Finish()
	}

	// Sort results
	sortStats(stats, cfg.SortBy, cfg.Desc)

	// Output
	switch cfg.Format {
	case "json":
		if err := writeJSON(stats); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			return 1
		}
		return 0
	case "csv":
		if err := writeCSV(stats, !cfg.NoHeader); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing CSV: %v\n", err)
			return 1
		}
		return 0
	default: // table
		writeTable(stats, cfg.ShowSum, cfg.Plain, cfg.Commas)
		return 0
	}
}

func sortStats(stats []analyze.FileStats, sortBy string, desc bool) {
	sortBy = strings.ToLower(sortBy)
	sort.Slice(stats, func(i, j int) bool {
		a, b := stats[i], stats[j]
		// Errors at the end
		if a.HasError && !b.HasError {
			return false
		}
		if !a.HasError && b.HasError {
			return true
		}
		// Group text before binary
		if a.Kind != b.Kind {
			if a.Kind == "binary" {
				return false
			}
			if b.Kind == "binary" {
				return true
			}
		}
		var less bool
		switch sortBy {
		case "name":
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case "ext":
			less = strings.ToLower(a.Ext) < strings.ToLower(b.Ext)
		case "size":
			less = a.SizeBytes < b.SizeBytes
		case "lines":
			less = a.Lines < b.Lines
		case "words":
			less = a.Words < b.Words
		case "chars":
			less = a.Chars < b.Chars
		case "modified":
			// Prefer ModUnix if available (0 means unknown)
			if a.ModUnix != 0 || b.ModUnix != 0 {
				less = a.ModUnix < b.ModUnix
			} else {
				less = a.ModTime < b.ModTime
			}
		default:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if desc {
			return !less
		}
		return less
	})
}

func writeJSON(stats []analyze.FileStats) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(stats)
}

func writeCSV(stats []analyze.FileStats, header bool) error {
	w := csv.NewWriter(os.Stdout)
	if header {
		_ = w.Write([]string{"File", "Ext", "Kind", "SizeBytes", "Size", "Lines", "Words", "Chars", "Modified", "Error"})
	}
	for _, fs := range stats {
		if fs.HasError {
			_ = w.Write([]string{fs.Name, fs.Ext, "", "", "", "", "", "", fs.ModTime, fs.ErrorText})
			continue
		}
		ls := fmt.Sprintf("%d", fs.Lines)
		ws := fmt.Sprintf("%d", fs.Words)
		cs := fmt.Sprintf("%d", fs.Chars)
		if fs.Kind == "binary" {
			ls, ws, cs = "-", "-", "-"
		}
		_ = w.Write([]string{
			fs.Name,
			fs.Ext,
			fs.Kind,
			fmt.Sprintf("%d", fs.SizeBytes),
			fs.Size,
			ls,
			ws,
			cs,
			fs.ModTime,
			"",
		})
	}
	w.Flush()
	return w.Error()
}

func writeTable(stats []analyze.FileStats, showSum bool, plain bool, commas bool) {
	// Headers: include Kind
	headers := []string{"File", "Ext", "Kind", "Size", "Lines", "Words", "Chars", "Modified"}

	// helpers
	displayWidth := func(s string) int {
		return utf8.RuneCountInString(s)
	}
	padRight := func(s string, width int) string {
		pad := width - displayWidth(s)
		if pad > 0 {
			return s + strings.Repeat(" ", pad)
		}
		return s
	}
	padLeft := func(s string, width int) string {
		pad := width - displayWidth(s)
		if pad > 0 {
			return strings.Repeat(" ", pad) + s
		}
		return s
	}
	fmtInt := func(n int) string {
		if commas {
			return util.CommaInt(n)
		}
		return fmt.Sprintf("%d", n)
	}
	// columns 3..6 right-aligned: Size, Lines, Words, Chars (0-based index)
	rightAligned := map[int]bool{3: true, 4: true, 5: true, 6: true}

	var rows [][]string
	var totalBytes int64
	var totalLines, totalWords, totalChars int

	for _, fs := range stats {
		if fs.HasError {
			rows = append(rows, []string{fs.Name, fs.Ext, "-", "-", "-", "-", "-", fs.ErrorText})
			continue
		}
		lstr, wstr, cstr := fmtInt(fs.Lines), fmtInt(fs.Words), fmtInt(fs.Chars)
		if fs.Kind == "binary" {
			lstr, wstr, cstr = "-", "-", "-"
		}
		rows = append(rows, []string{
			fs.Name,
			fs.Ext,
			fs.Kind,
			fs.Size,
			lstr,
			wstr,
			cstr,
			fs.ModTime,
		})
		totalBytes += fs.SizeBytes
		if fs.Kind != "binary" {
			totalLines += fs.Lines
			totalWords += fs.Words
			totalChars += fs.Chars
		}
	}

	// compute column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = displayWidth(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := displayWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	// optional footer
	var hasFooter bool
	var footer []string
	if showSum {
		hasFooter = true
		footer = []string{
			fmt.Sprintf("TOTAL (%d files)", len(stats)),
			"",
			"",
			analyze.HumanBytes(totalBytes),
			fmtInt(totalLines),
			fmtInt(totalWords),
			fmtInt(totalChars),
			"",
		}
		for i, cell := range footer {
			if w := displayWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	// drawing characters
	vert := "│"
	horiz := "─"
	topLeft, topSep, topRight := "╭", "┬", "╮"
	midLeft, midSep, midRight := "├", "┼", "┤"
	botLeft, botSep, botRight := "╰", "┴", "╯"
	if plain {
		vert = "|"
		horiz = "-"
		topLeft, topSep, topRight = "+", "+", "+"
		midLeft, midSep, midRight = "+", "+", "+"
		botLeft, botSep, botRight = "+", "+", "+"
	}

	// draw line helpers
	drawLine := func(left, sep, right string) {
		var b strings.Builder
		b.WriteString(left)
		for i, w := range widths {
			b.WriteString(strings.Repeat(horiz, w+2))
			if i < len(widths)-1 {
				b.WriteString(sep)
			}
		}
		b.WriteString(right)
		fmt.Fprintln(os.Stdout, b.String())
	}
	// draw row helper
	drawRow := func(cells []string) {
		var b strings.Builder
		b.WriteString(vert)
		for i, w := range widths {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			if rightAligned[i] {
				b.WriteString(" " + padLeft(cell, w) + " ")
			} else {
				b.WriteString(" " + padRight(cell, w) + " ")
			}
			if i < len(widths)-1 {
				b.WriteString(vert)
			}
		}
		b.WriteString(vert)
		fmt.Fprintln(os.Stdout, b.String())
	}

	// render table
	drawLine(topLeft, topSep, topRight)
	drawRow(headers)
	drawLine(midLeft, midSep, midRight)
	for i, row := range rows {
		drawRow(row)
		if i < len(rows)-1 {
			drawLine(midLeft, midSep, midRight)
		}
	}
	if hasFooter {
		if len(rows) > 0 {
			drawLine(midLeft, midSep, midRight)
		}
		drawRow(footer)
	}
	drawLine(botLeft, botSep, botRight)
}
