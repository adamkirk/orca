package tui

import (
	"bytes"
	"io"
	"sync"
)

// indentWriter starts each line written to it with a prefix, e.g. to show a
// command's output under what's running it.
type indentWriter struct {
	mu     sync.Mutex
	w      io.Writer
	prefix []byte

	// Whether the next byte starts a line
	atLineStart bool
}

func (w *indentWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var out bytes.Buffer

	for _, b := range p {
		// Empty lines are left empty, rather than ending in spaces
		if w.atLineStart && b != '\n' {
			out.Write(w.prefix)
		}

		out.WriteByte(b)
		w.atLineStart = b == '\n'
	}

	if _, err := w.w.Write(out.Bytes()); err != nil {
		return 0, err
	}

	return len(p), nil
}

// Indented is a writer to the standard output, which starts every line with
// prefix. Output and errors written to it from different places (e.g. a
// command's stdout and stderr) are kept in whole writes.
func (t *Tui) Indented(prefix string) io.Writer {
	return &indentWriter{w: t.std, prefix: []byte(prefix), atLineStart: true}
}
