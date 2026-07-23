package executor

import "bytes"

// maxStreamLine bounds the amount of a single line lineStreamer will buffer
// for RunOpts.Stream delivery. It does NOT bound Result.Stdout capture: raw
// bytes are always written to buf verbatim, regardless of line length. A
// line longer than maxStreamLine is simply dropped from the Stream
// callbacks (never delivered, never an error) rather than growing without
// bound or overflowing a scanner.
const maxStreamLine = 1 << 20 // 1MB

// lineStreamer is an io.Writer that captures every byte written to it
// verbatim into buf — so Result.Stdout ends up byte-exact: no fabricated
// trailing newline, and \r\n is preserved exactly as produced — while also
// splitting the same bytes on '\n' to invoke fn once per complete line, in
// order, as they arrive. This replaces a bufio.Scanner-based capture, whose
// fixed max-token-size makes a single oversized line fail the whole Run
// with bufio.ErrTooLong; lineStreamer can never error.
type lineStreamer struct {
	buf      *bytes.Buffer
	fn       StreamFunc
	line     []byte
	overflow bool
}

func (w *lineStreamer) Write(p []byte) (int, error) {
	w.buf.Write(p)
	if w.fn == nil {
		return len(p), nil
	}
	for _, b := range p {
		if b == '\n' {
			w.emit()
			continue
		}
		if len(w.line) >= maxStreamLine {
			// Drop bytes past the cap for streaming purposes only; buf
			// above already has them verbatim.
			w.overflow = true
			continue
		}
		w.line = append(w.line, b)
	}
	return len(p), nil
}

// emit delivers the currently buffered line to fn (trimming a trailing '\r'
// to match text-mode line splitting), unless it overflowed maxStreamLine, in
// which case it is silently dropped, and resets state for the next line.
func (w *lineStreamer) emit() {
	if !w.overflow {
		line := w.line
		if n := len(line); n > 0 && line[n-1] == '\r' {
			line = line[:n-1]
		}
		w.fn(string(line))
	}
	w.line = w.line[:0]
	w.overflow = false
}

// Flush delivers a final line that was never terminated by '\n' (i.e. the
// command's output didn't end in a newline). Call once after the writer has
// seen all input.
func (w *lineStreamer) Flush() {
	if w.fn != nil && !w.overflow && len(w.line) > 0 {
		w.fn(string(w.line))
	}
	w.line = w.line[:0]
	w.overflow = false
}
