package main

import (
	"fmt"
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/table"
)

// RunCmd runs a named preset — a predefined combination of visualization,
// metrics, and palette that generates a useful image without requiring the
// caller to know which metrics and palettes to combine.
//
// Usage:
//
//	codeviz run                                  # list available presets
//	codeviz run <preset> <target> -o <output>    # run a preset
type RunCmd struct {
	Preset     string `arg:"" optional:"" name:"preset" help:"Name of the preset to run; omit to list available presets."`
	TargetPath string `arg:"" optional:"" name:"target" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." optional:"" short:"o"`

	Title      string `help:"Override the preset's default title." optional:""`
	Width      int    `default:"1920" help:"Image width in pixels."`
	Height     int    `default:"1080" help:"Image height in pixels."`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`
}

// presetDef describes a named preset.
type presetDef struct {
	Name         string
	Description  string
	DefaultTitle string
}

// presets is the registry of all available presets.
var presets = []presetDef{
	{
		Name:         "structure-treemap",
		Description:  "Treemap sized by file lines; colour shows file type. Quick overview of code structure.",
		DefaultTitle: "Code Structure",
	},
	{
		Name:         "structure-bubbletree",
		Description:  "Bubble tree sized by file lines; colour shows file type. Alternative overview of code structure.",
		DefaultTitle: "Code Structure",
	},
	{
		Name:         "history-treemap",
		Description:  "Treemap sized by file lines; colour shows commit count. Highlights frequently-changed hotspots.",
		DefaultTitle: "Commit Hotspots",
	},
	{
		Name:         "age-treemap",
		Description:  "Treemap sized by file lines; colour shows file age. Reveals stale and actively-maintained areas.",
		DefaultTitle: "File Age",
	},
	{
		Name:         "contributors-treemap",
		Description:  "Treemap sized by file lines; colour shows distinct author count. Useful for bus-factor analysis.",
		DefaultTitle: "Author Coverage",
	},
}

// findPreset looks up a preset by name, returning nil if not found.
func findPreset(name string) *presetDef {
	for i := range presets {
		if presets[i].Name == name {
			return &presets[i]
		}
	}

	return nil
}

// presetNames returns a comma-separated list of all preset names.
func presetNames() string {
	names := make([]string, len(presets))
	for i, p := range presets {
		names[i] = p.Name
	}

	return strings.Join(names, ", ")
}

// Validate is called by Kong before Run; it enforces argument consistency.
func (r *RunCmd) Validate() error {
	if r.Preset == "" {
		// list mode: no further arguments required
		return nil
	}

	if findPreset(r.Preset) == nil {
		return eris.Errorf("unknown preset %q; available presets: %s", r.Preset, presetNames())
	}

	if r.TargetPath == "" {
		return eris.Errorf("target path is required when a preset is specified")
	}

	if r.Output == "" {
		return eris.Errorf("output path (-o) is required when a preset is specified")
	}

	return nil
}

// Run either lists available presets (when no preset name is given) or
// executes the named preset.
func (r *RunCmd) Run(flags *Flags) error {
	if r.Preset == "" {
		return r.listPresets()
	}

	preset := findPreset(r.Preset)
	if preset == nil {
		// Should not reach here; Validate() would have caught this.
		return eris.Errorf("unknown preset %q", r.Preset)
	}

	return r.runPreset(preset, flags)
}

func (r *RunCmd) listPresets() error {
	tbl := table.New("Preset", "Description")
	tbl.SetMaxWidth(120)

	for _, p := range presets {
		tbl.AddRow(p.Name, p.Description)
	}

	sb := &strings.Builder{}
	tbl.WriteTo(sb)
	fmt.Print(sb.String())

	return nil
}

// effectiveTitle returns the user-supplied title if set, otherwise the preset's default.
func (r *RunCmd) effectiveTitle(preset *presetDef) string {
	if r.Title != "" {
		return r.Title
	}

	return preset.DefaultTitle
}

// runPreset dispatches execution to the appropriate viz command.
func (r *RunCmd) runPreset(preset *presetDef, flags *Flags) error {
	title := r.effectiveTitle(preset)

	switch preset.Name {
	case "structure-treemap":
		return (&TreemapCmd{
			TargetPath: r.TargetPath,
			Output:     r.Output,
			Size:       metric.Name("file-lines"),
			Fill:       config.MetricSpec{Metric: "file-type"},
			Width:      r.Width,
			Height:     r.Height,
			HideFooter: r.HideFooter,
			Title:      title,
		}).Run(flags)

	case "structure-bubbletree":
		return (&BubbletreeCmd{
			TargetPath: r.TargetPath,
			Output:     r.Output,
			Size:       metric.Name("file-lines"),
			Fill:       config.MetricSpec{Metric: "file-type"},
			Width:      r.Width,
			Height:     r.Height,
			HideFooter: r.HideFooter,
			Title:      title,
		}).Run(flags)

	case "history-treemap":
		return (&TreemapCmd{
			TargetPath: r.TargetPath,
			Output:     r.Output,
			Size:       metric.Name("file-lines"),
			Fill:       config.MetricSpec{Metric: "commit-count"},
			Width:      r.Width,
			Height:     r.Height,
			HideFooter: r.HideFooter,
			Title:      title,
		}).Run(flags)

	case "age-treemap":
		return (&TreemapCmd{
			TargetPath: r.TargetPath,
			Output:     r.Output,
			Size:       metric.Name("file-lines"),
			Fill:       config.MetricSpec{Metric: "file-age"},
			Width:      r.Width,
			Height:     r.Height,
			HideFooter: r.HideFooter,
			Title:      title,
		}).Run(flags)

	case "contributors-treemap":
		return (&TreemapCmd{
			TargetPath: r.TargetPath,
			Output:     r.Output,
			Size:       metric.Name("file-lines"),
			Fill:       config.MetricSpec{Metric: "author-count"},
			Width:      r.Width,
			Height:     r.Height,
			HideFooter: r.HideFooter,
			Title:      title,
		}).Run(flags)

	default:
		return eris.Errorf("unhandled preset %q", preset.Name)
	}
}
