package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/export"
)

// ExportConfig writes the merged effective config to disk when
// Flags.ExportConfig is non-empty.
func ExportConfig(c *CommonState) error {
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
func ExportData(c *CommonState) error {
	if err := export.Export(c.Root, c.Requested.LegacyNames(), c.Flags.ExportData); err != nil {
		return eris.Wrap(err, "failed to export data")
	}

	return nil
}
