// Package filesystem provides metric providers for filesystem-derived metrics.
package filesystem

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

// Metric name constants for filesystem metrics.
const (
	FileSize  metric.Name = "file-size"
	FileLines metric.Name = "file-lines"
	FileType  metric.Name = "file-type"
)

// FileSizeProvider reports file size in bytes. Value is set during scan; Load is a no-op.
type FileSizeProvider struct{}

func (FileSizeProvider) Name() metric.Name                   { return FileSize }
func (FileSizeProvider) Kind() metric.Kind                   { return metric.Quantity }
func (FileSizeProvider) Description() string                 { return "Size of each file in bytes." }
func (FileSizeProvider) Dependencies() []metric.Name         { return nil }
func (FileSizeProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }
func (FileSizeProvider) Load(_ *model.Directory) error       { return nil }

// FileTypeProvider reports the file type classification. Value is set during scan; Load is a no-op.
type FileTypeProvider struct{}

func (FileTypeProvider) Name() metric.Name                   { return FileType }
func (FileTypeProvider) Kind() metric.Kind                   { return metric.Classification }
func (FileTypeProvider) Description() string                 { return "File extension category (e.g. go, md, png)." }
func (FileTypeProvider) Dependencies() []metric.Name         { return nil }
func (FileTypeProvider) DefaultPalette() palette.PaletteName { return palette.Categorization }
func (FileTypeProvider) Load(_ *model.Directory) error       { return nil }

// FileLinesProvider counts lines in each text file.
type FileLinesProvider struct {
	onFile func()
}

func (*FileLinesProvider) Name() metric.Name                   { return FileLines }
func (*FileLinesProvider) Kind() metric.Kind                   { return metric.Quantity }
func (*FileLinesProvider) Description() string                 { return "Number of lines in each text file." }
func (*FileLinesProvider) Dependencies() []metric.Name         { return nil }
func (*FileLinesProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }

func (p *FileLinesProvider) SetOnFileProcessed(fn func()) { p.onFile = fn }

func (p *FileLinesProvider) Load(root *model.Directory) error {
	model.WalkFiles(root, func(f *model.File) {
		if p.onFile != nil {
			defer p.onFile()
		}

		if f.IsBinary {
			return
		}

		count, err := countLines(f.Path)
		if err != nil {
			if errors.Is(err, errBinaryFile) {
				f.IsBinary = true

				return
			}

			slog.Warn("could not count lines", "path", f.Path, "error", err)

			return
		}

		f.SetQuantity(FileLines, count)
	})

	return nil
}

var errBinaryFile = errors.New("file appears to be binary")

// binaryProbeSize is the number of bytes read from the start of a file to
// detect binary content. This matches the heuristic used by Git.
const binaryProbeSize = 8000

func countLines(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, eris.Wrap(err, "opening file for line count")
	}
	defer file.Close()

	if isBinary, err := probeBinary(file); err != nil {
		return 0, err
	} else if isBinary {
		return 0, errBinaryFile
	}

	scanner := bufio.NewScanner(file)

	var count int64
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return 0, errBinaryFile
		}

		return 0, eris.Wrap(err, "reading file lines")
	}

	return count, nil
}

// probeBinary reads the first binaryProbeSize bytes of f and reports whether
// the content looks like a binary file. It uses a null-byte heuristic (same
// approach as Git) but skips the check for files that start with a UTF-16 BOM,
// since UTF-16 text legitimately contains null bytes.
//
// On return the file is seeked back to the start, ready for line counting.
func probeBinary(f *os.File) (bool, error) {
	header := make([]byte, binaryProbeSize)

	n, err := f.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, eris.Wrap(err, "reading file header for binary probe")
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, eris.Wrap(err, "seeking back to start after binary probe")
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
			return false // UTF-32 LE — not text we can handle
		}

		return true
	}

	// UTF-16 BE: FE FF
	return buf[0] == 0xFE && buf[1] == 0xFF
}
