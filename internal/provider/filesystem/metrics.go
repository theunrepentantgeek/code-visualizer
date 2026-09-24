// Package filesystem provides metric providers for filesystem-derived metrics.
package filesystem

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/rotisserie/eris"
	"golang.org/x/text/encoding/unicode"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// Metric name constants for filesystem metrics.
const (
	FileSize  metric.Name = "file-size"
	FileLines metric.Name = "file-lines"
	FileType  metric.Name = "file-type"
)

// IsFilesystemMetric reports whether name is provided by the filesystem provider.
func IsFilesystemMetric(name metric.Name) bool {
	switch name {
	case FileSize, FileLines, FileType:
		return true
	default:
		return false
	}
}

// FileSizeProvider reports file size in bytes. Value is set during scan; Load is a no-op.
type FileSizeProvider struct{}

func (FileSizeProvider) Load(_ *model.Directory) error { return nil }

// FileTypeProvider reports the file type classification. Value is set during scan; Load is a no-op.
type FileTypeProvider struct{}

func (FileTypeProvider) Load(_ *model.Directory) error { return nil }

// FileLinesProvider counts lines in each text file.
type FileLinesProvider struct {
	onFile func()
	mu     sync.Mutex
}

func (p *FileLinesProvider) SetOnFileProcessed(fn func()) { p.onFile = fn }
func (p *FileLinesProvider) FileProgressMutex() *sync.Mutex {
	return &p.mu
}

func (p *FileLinesProvider) Load(root *model.Directory) error {
	model.WalkFiles(root, func(f *model.File) {
		if p.onFile != nil {
			defer p.onFile()
		}

		if f.IsBinary {
			return
		}

		count, err := countLinesFile(f)
		if err != nil {
			slog.Warn("could not count lines", "path", f.Path, "error", err)

			return
		}

		f.SetQuantity(FileLines, count)
	})

	return nil
}

func countLinesFile(file *model.File) (int64, error) {
	if file.Source == nil {
		return countLines(file.Path)
	}

	data, err := file.ReadAll()
	if err != nil {
		return 0, eris.Wrap(err, "reading file for line count")
	}

	return countLinesReader(bytes.NewReader(data))
}

// utf16Encoding indicates the UTF-16 byte-order of a file, if any.
type utf16Encoding int

const (
	notUTF16 utf16Encoding = iota
	utf16LE
	utf16BE
)

func countLines(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, eris.Wrap(err, "opening file for line count")
	}
	defer file.Close()

	return countLinesReader(file)
}

func countLinesReader(file io.ReadSeeker) (int64, error) {
	enc, err := detectFileEncoding(file)
	if err != nil {
		return 0, err
	}

	var r io.Reader = file
	if enc != notUTF16 {
		// Wrap in UTF-16 decoder so that newlines are decoded correctly.
		// UseBOM reads the BOM to determine endianness, overriding the default.
		r = unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Reader(file)
	}

	reader := bufio.NewReader(r)

	var count int64
	for {
		line, readErr := reader.ReadString('\n')
		if len(line) > 0 {
			count++
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return count, nil
			}

			return 0, eris.Wrap(readErr, "reading file lines")
		}
	}
}

// detectFileEncoding checks the byte-order mark and seeks back to the start.
func detectFileEncoding(f io.ReadSeeker) (utf16Encoding, error) {
	header := make([]byte, 4)

	n, readErr := f.Read(header)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return notUTF16, eris.Wrap(readErr, "reading file header")
	}

	if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
		return notUTF16, eris.Wrap(seekErr, "seeking back to start after encoding detection")
	}

	return detectUTF16Encoding(header[:n]), nil
}

// detectUTF16Encoding reports the UTF-16 byte-order of buf based on its BOM,
// or notUTF16 if no UTF-16 BOM is detected.
func detectUTF16Encoding(buf []byte) utf16Encoding {
	if len(buf) < 2 {
		return notUTF16
	}

	// UTF-16 LE: FF FE (but not FF FE 00 00, which is UTF-32 LE)
	if buf[0] == 0xFF && buf[1] == 0xFE {
		if len(buf) >= 4 && buf[2] == 0x00 && buf[3] == 0x00 {
			return notUTF16 // UTF-32 LE — not text we can handle
		}

		return utf16LE
	}

	// UTF-16 BE: FE FF
	if buf[0] == 0xFE && buf[1] == 0xFF {
		return utf16BE
	}

	return notUTF16
}
