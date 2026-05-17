package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// ScanFilesystem walks Common().TargetPath, populates Common().Root, and
// wires progress reporting based on Flags verbosity.
func ScanFilesystem[S VizState](s S) error {
	c := s.Common()

	slog.Info("Scanning filesystem", "path", c.TargetPath)

	scanProg, stopScanTicker := BuildScanProgress(c.Flags)

	root, err := scan.Scan(c.TargetPath, c.FilterRules, scanProg)

	stopScanTicker()

	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	c.Root = root

	return nil
}

var _ pipeline.Stage[VizState] = ScanFilesystem[VizState]
