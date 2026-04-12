// Package filesystem provides metric providers for filesystem-derived metrics.
package filesystem

import (
	"bufio"
	"errors"
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
func (FileSizeProvider) Dependencies() []metric.Name         { return nil }
func (FileSizeProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }
func (FileSizeProvider) Load(_ *model.Directory) error       { return nil }

// FileTypeProvider reports the file type classification. Value is set during scan; Load is a no-op.
type FileTypeProvider struct{}

func (FileTypeProvider) Name() metric.Name                   { return FileType }
func (FileTypeProvider) Kind() metric.Kind                   { return metric.Classification }
func (FileTypeProvider) Dependencies() []metric.Name         { return nil }
func (FileTypeProvider) DefaultPalette() palette.PaletteName { return palette.Categorization }
func (FileTypeProvider) Load(_ *model.Directory) error       { return nil }

// FileLinesProvider counts lines in each text file.
type FileLinesProvider struct{}

func (FileLinesProvider) Name() metric.Name                   { return FileLines }
func (FileLinesProvider) Kind() metric.Kind                   { return metric.Quantity }
func (FileLinesProvider) Dependencies() []metric.Name         { return nil }
func (FileLinesProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }

func (FileLinesProvider) Load(root *model.Directory) error {
	model.WalkFiles(root, func(f *model.File) {
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

var errBinaryFile = errors.New("file appears to be binary (line exceeds 64KB)")

func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, eris.Wrap(err, "opening file for line count")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	count := 0
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
