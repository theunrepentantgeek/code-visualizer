package stages

import (
	"context"
	"errors"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// ScanFilesystem walks c.TargetPath, populates c.Root, and wires progress
// reporting based on Flags verbosity.
func ScanFilesystem(c *CommonState, ctx context.Context, sink progress.Sink) error {
	if c.Source.FS == nil {
		if err := ResolveSource(c); err != nil {
			return eris.Wrap(err, "failed to resolve scan source")
		}
	}

	scanProg := newScanProgress(sink)
	root, err := scan.ScanTree(ctx, c.Source, c.FilterRules, scanProg, c.IncludeBinaryFiles)

	if err != nil {
		if errors.Is(err, ctx.Err()) {
			return ctx.Err()
		}

		return eris.Wrap(err, "scan failed")
	}
	if err := scanProg.Err(); err != nil {
		return eris.Wrap(err, "report scan progress")
	}

	c.Root = root

	return nil
}
