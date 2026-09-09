package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// ScanFilesystem walks c.TargetPath, populates c.Root, and wires progress
// reporting based on Flags verbosity.
func ScanFilesystem(c *CommonState) error {
	slog.Info("Scanning filesystem", "path", c.TargetPath)

	if c.Source.FS == nil {
		if err := ResolveSource(c); err != nil {
			return eris.Wrap(err, "failed to resolve scan source")
		}
	}

	scanProg, stopScanTicker := BuildScanProgress(c.Flags)

	root, err := scan.ScanTree(c.Source, c.FilterRules, scanProg, c.IncludeBinaryFiles)

	stopScanTicker()

	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	c.Root = root

	return nil
}
