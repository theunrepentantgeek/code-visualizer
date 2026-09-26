package progress

import (
	"bytes"
	"io"
)

type diagnosticWriter struct {
	reporter *reporter
	buffer   bytes.Buffer
}

func (w *diagnosticWriter) Write(p []byte) (int, error) {
	w.reporter.mu.Lock()
	defer w.reporter.mu.Unlock()

	w.buffer.Write(p)
	for {
		line, err := w.buffer.ReadString('\n')
		if err != nil {
			w.buffer.WriteString(line)

			break
		}

		if err := w.reporter.renderer.writeDiagnostic(line[:len(line)-1]); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

func (w *diagnosticWriter) flushLocked() error {
	if w == nil || w.buffer.Len() == 0 {
		return nil
	}

	line := w.buffer.String()
	w.buffer.Reset()

	return w.reporter.renderer.writeDiagnostic(line)
}

var _ io.Writer = (*diagnosticWriter)(nil)
