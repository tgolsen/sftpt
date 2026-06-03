package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	minUpdateInterval = 80 * time.Millisecond
	barWidth          = 30
	maxFilenameLen    = 30
)

// Writer wraps an io.Writer, tracking bytes written and rendering a progress bar.
type Writer struct {
	writer io.Writer
	total  int64
	done   int64
	name   string
	start  time.Time
	last   time.Time
	termW  int
}

// NewWriter creates a progress Writer. total can be 0 if unknown.
func NewWriter(w io.Writer, total int64, name string) *Writer {
	termW, _, _ := term.GetSize(int(os.Stderr.Fd()))
	if termW < 40 {
		termW = 40
	}
	return &Writer{
		writer: w,
		total:  total,
		name:   truncateName(name, maxFilenameLen),
		start:  time.Now(),
		termW:  termW,
	}
}

// Write writes data to the underlying writer and updates the progress bar.
func (w *Writer) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.done += int64(n)
	w.maybeRender()
	return n, err
}

// Done finalises the progress bar (forces 100% and prints a newline).
func (w *Writer) Done() {
	w.render()
	fmt.Fprint(os.Stderr, "\n")
}

// maybeRender renders only if enough time has passed since the last render.
func (w *Writer) maybeRender() {
	now := time.Now()
	if now.Sub(w.last) < minUpdateInterval {
		return // rate limit
	}
	w.last = now
	w.render()
}

// render draws a single progress bar frame to stderr.
func (w *Writer) render() {
	elapsed := time.Since(w.start)
	var speed float64
	if elapsed.Seconds() > 0 {
		speed = float64(w.done) / elapsed.Seconds()
	}

	// Build the compact progress line.
	// Format: "name  45% |████......|  4.5 MB / 10.0 MB  12.3 MB/s     "
	pct := 0
	if w.total > 0 {
		pct = int(float64(w.done) / float64(w.total) * 100)
		if pct > 100 {
			pct = 100
		}
	}

	var bar string
	if w.total > 0 {
		filled := int(float64(barWidth) * float64(w.done) / float64(w.total))
		if filled > barWidth {
			filled = barWidth
		}
		bar = strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	} else {
		// Indeterminate: a moving segment
		seg := int(elapsed.Seconds()*4) % (barWidth - 4)
		bar = strings.Repeat("░", seg) + "████" + strings.Repeat("░", barWidth-seg-4)
	}

	// Build the line
	line := fmt.Sprintf("\r%s %3d%% |%s| %s",
		w.name,
		pct,
		bar,
		formatBytes(w.done),
	)

	if w.total > 0 {
		line += fmt.Sprintf(" / %s", formatBytes(w.total))
	}

	line += fmt.Sprintf("  %s/s", formatBytes(int64(speed)))

	// Pad/truncate to terminal width (leave room for the \r)
	line = padToWidth(line, w.termW-1)

	fmt.Fprint(os.Stderr, line)
}

// Reader wraps an io.Reader, tracking bytes read and rendering a progress bar.
type Reader struct {
	reader io.Reader
	total  int64
	done   int64
	name   string
	start  time.Time
	last   time.Time
	termW  int
}

// NewReader creates a progress Reader. total can be 0 if unknown.
func NewReader(r io.Reader, total int64, name string) *Reader {
	termW, _, _ := term.GetSize(int(os.Stderr.Fd()))
	if termW < 40 {
		termW = 40
	}
	return &Reader{
		reader: r,
		total:  total,
		name:   truncateName(name, maxFilenameLen),
		start:  time.Now(),
		termW:  termW,
	}
}

// Read reads data from the underlying reader and updates the progress bar.
func (r *Reader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.done += int64(n)
	r.maybeRender()
	return n, err
}

// Done finalises the progress bar (forces 100% and prints a newline).
func (r *Reader) Done() {
	r.render()
	fmt.Fprint(os.Stderr, "\n")
}

// maybeRender renders only if enough time has passed since the last render.
func (r *Reader) maybeRender() {
	now := time.Now()
	if now.Sub(r.last) < minUpdateInterval {
		return
	}
	r.last = now
	r.render()
}

// render draws a single progress bar frame to stderr.
func (r *Reader) render() {
	elapsed := time.Since(r.start)
	var speed float64
	if elapsed.Seconds() > 0 {
		speed = float64(r.done) / elapsed.Seconds()
	}

	pct := 0
	if r.total > 0 {
		pct = int(float64(r.done) / float64(r.total) * 100)
		if pct > 100 {
			pct = 100
		}
	}

	var bar string
	if r.total > 0 {
		filled := int(float64(barWidth) * float64(r.done) / float64(r.total))
		if filled > barWidth {
			filled = barWidth
		}
		bar = strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	} else {
		seg := int(elapsed.Seconds()*4) % (barWidth - 4)
		bar = strings.Repeat("░", seg) + "████" + strings.Repeat("░", barWidth-seg-4)
	}

	line := fmt.Sprintf("\r%s %3d%% |%s| %s",
		r.name,
		pct,
		bar,
		formatBytes(r.done),
	)

	if r.total > 0 {
		line += fmt.Sprintf(" / %s", formatBytes(r.total))
	}

	line += fmt.Sprintf("  %s/s", formatBytes(int64(speed)))
	line = padToWidth(line, r.termW-1)

	fmt.Fprint(os.Stderr, line)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < 0 {
		return "0 B"
	}
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func truncateName(name string, max int) string {
	if len(name) <= max {
		return name
	}
	if max < 4 {
		return name[:max]
	}
	// Show beginning...end
	half := (max - 3) / 2
	return name[:half] + "..." + name[len(name)-(max-3-half):]
}

func padToWidth(s string, width int) string {
	runeCount := utf8.RuneCountInString(s)
	if runeCount >= width {
		// Truncate to fit (unlikely for a progress bar)
		runes := []rune(s)
		if width > 0 {
			return string(runes[:width])
		}
		return ""
	}
	return s + strings.Repeat(" ", width-runeCount)
}
