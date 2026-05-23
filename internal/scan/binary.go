package scan

import (
	"bytes"
	"errors"
	"io"
	"os"

	"github.com/rotisserie/eris"
)

// binaryProbeSize is the number of bytes read from the start of a file to
// detect binary content. This matches the heuristic used by Git.
const binaryProbeSize = 8000

// IsBinaryFile reports whether the file at path appears to be binary.
// It reads the first binaryProbeSize bytes and checks for null bytes,
// skipping files that start with a UTF-16 BOM (which legitimately
// contain null bytes).
func IsBinaryFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, eris.Wrap(err, "opening file for binary probe")
	}
	defer f.Close()

	header := make([]byte, binaryProbeSize)

	n, readErr := f.Read(header)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return false, eris.Wrap(readErr, "reading file header for binary probe")
	}

	if n == 0 {
		return false, nil
	}

	buf := header[:n]

	if hasUTF16BOM(buf) {
		return false, nil
	}

	return bytes.IndexByte(buf, 0) >= 0, nil
}

// hasUTF16BOM reports whether buf starts with a UTF-16 byte-order mark.
func hasUTF16BOM(buf []byte) bool {
	if len(buf) < 2 {
		return false
	}

	// UTF-16 LE: FF FE (but not FF FE 00 00, which is UTF-32 LE)
	if buf[0] == 0xFF && buf[1] == 0xFE {
		if len(buf) >= 4 && buf[2] == 0x00 && buf[3] == 0x00 {
			return false // UTF-32 LE
		}

		return true
	}

	// UTF-16 BE: FE FF
	if buf[0] == 0xFE && buf[1] == 0xFF {
		return true
	}

	return false
}
