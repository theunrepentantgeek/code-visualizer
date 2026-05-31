package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/export"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
)

// ExportConfig writes the merged effective config to disk when
// Flags.ExportConfig is non-empty.
func ExportConfig[S VizState](s S) error {
	c := s.Common()
	if c.Flags.ExportConfig == "" {
		return nil
	}

	exportCfg := c.RootConfig.ForExport(c.VizName)
	if err := exportCfg.Save(c.Flags.ExportConfig); err != nil {
		return eris.Wrap(err, "failed to save config")
	}

	return nil
}

// ExportData writes computed metric data to disk when Flags.ExportData is
// non-empty.
func ExportData[S VizState](s S) error {
	c := s.Common()
	if err := export.Export(c.Root, c.Requested, c.Flags.ExportData); err != nil {
		return eris.Wrap(err, "failed to export data")
	}

	return nil
}

var (
	_ pipeline.Stage[VizState] = ExportConfig[VizState]
	_ pipeline.Stage[VizState] = ExportData[VizState]
)
