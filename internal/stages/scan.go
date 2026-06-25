package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// ScanFilesystem walks c.TargetPath, populates c.Root, and wires progress
// reporting based on Flags verbosity.
func ScanFilesystem(c *CommonState) error {
	slog.Info("Scanning filesystem", "path", c.TargetPath)

	scanProg, stopScanTicker := BuildScanProgress(c.Flags)

	root, err := scan.Scan(c.TargetPath, c.FilterRules, scanProg)

	stopScanTicker()

	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	c.Root = root
	c.FileCount = model.CountFiles(root)

	return nil
}
