package analyze

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/ADJB1212/Aperio/internal/util"
)

type FileStats struct {
	Name      string
	Ext       string
	Kind      string
	SizeBytes int64
	Size      string
	Lines     int
	Words     int
	Chars     int
	ModTime   string
	ModUnix   int64
	HasError  bool
	ErrorText string
}

func HumanBytes(bytes int64) string {
	return util.HumanBytes(bytes)
}

func AnalyzeFile(path string, out chan<- FileStats, wg *sync.WaitGroup) {
	defer wg.Done()
	stat := FileStats{Name: filepath.Base(path), Ext: filepath.Ext(path)}

	info, err := os.Stat(path)
	if err != nil {
		stat.HasError = true
		stat.ErrorText = err.Error()
		out <- stat
		return
	}

	stat.SizeBytes = info.Size()
	stat.Size = HumanBytes(info.Size())
	stat.ModTime = info.ModTime().Format("2006-01-02 15:04:05")
	stat.ModUnix = info.ModTime().Unix()

	f, err := os.Open(path)
	if err != nil {
		stat.HasError = true
		stat.ErrorText = err.Error()
		out <- stat
		return
	}
	defer f.Close()

	// Detect binary files by scanning a small prefix for NUL bytes or invalid UTF-8.
	// If binary, skip expensive text scanning.
	stat.Kind = "text"
	if stat.SizeBytes > 0 {
		const sniffSize = 8192
		toRead := min(stat.SizeBytes, int64(sniffSize))
		sniff := make([]byte, toRead)
		if n, _ := f.ReadAt(sniff, 0); n > 0 {
			isBin := false
			// Quick NUL check
			for i := range n {
				if sniff[i] == 0x00 {
					isBin = true
					break
				}
			}
			// UTF-8 sanity check
			if !isBin {
				i := 0
				invalid := 0
				for i < n {
					r, size := utf8.DecodeRune(sniff[i:n])
					if r == utf8.RuneError && size == 1 {
						if !utf8.FullRune(sniff[i:n]) {
							// Incomplete at end; stop checking.
							break
						}
						invalid++
						i++
					} else {
						i += size
					}
					if invalid > 0 {
						isBin = true
						break
					}
				}
			}
			if isBin {
				stat.Kind = "binary"
				stat.Lines = 0
				stat.Words = 0
				stat.Chars = 0
				out <- stat
				return
			}
		}
	}

	lines, words, chars := 0, 0, 0
	inWord := false
	lastWasNewline := false

	buf := make([]byte, 64*1024)
	var leftover [4]byte
	leftN := 0

	for {
		n, err := f.Read(buf)
		if n > 0 {
			i := 0

			// Complete a partial UTF-8 rune from previous chunk, if any.
			if leftN > 0 {
				need := min(4-leftN, n)
				copy(leftover[leftN:], buf[:need])
				seq := leftover[:leftN+need]
				r, size := utf8.DecodeRune(seq)
				if r == utf8.RuneError && size == 1 && !utf8.FullRune(seq) {
					// Still incomplete; carry over and continue with next read.
					leftN += need
					i = need
				} else {
					chars++
					if r == '\n' {
						lines++
						inWord = false
						lastWasNewline = true
					} else {
						lastWasNewline = false
						if unicode.IsSpace(r) {
							if inWord {
								inWord = false
							}
						} else {
							if !inWord {
								words++
								inWord = true
							}
						}
					}
					i = size - leftN
					leftN = 0
				}
			}

			// Process current chunk.
			for i < n {
				r, size := utf8.DecodeRune(buf[i:n])
				if r == utf8.RuneError && size == 1 && !utf8.FullRune(buf[i:n]) {
					// Incomplete rune at end of buffer; stash for next read.
					leftN = copy(leftover[:], buf[i:n])
					i = n
					break
				}
				chars++
				if r == '\n' {
					lines++
					inWord = false
					lastWasNewline = true
				} else {
					lastWasNewline = false
					if unicode.IsSpace(r) {
						if inWord {
							inWord = false
						}
					} else {
						if !inWord {
							words++
							inWord = true
						}
					}
				}
				i += size
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			stat.HasError = true
			stat.ErrorText = err.Error()
			out <- stat
			return
		}
	}

	// Handle any incomplete UTF-8 sequence at EOF as a single replacement rune.
	if leftN > 0 {
		chars++
		lastWasNewline = false
		if !inWord {
			words++
			inWord = true
		}
	}

	// Count the final line if the file doesn't end with a newline and has content.
	if chars > 0 && !lastWasNewline {
		lines++
	}

	stat.Lines = lines
	stat.Words = words
	stat.Chars = chars
	out <- stat
}
