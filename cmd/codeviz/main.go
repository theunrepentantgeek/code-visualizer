package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/render"
	"github.com/bevan/code-visualizer/internal/scan"
	"github.com/bevan/code-visualizer/internal/treemap"
)

type CLI struct {
	TargetPath  string            `arg:"" help:"Path to directory to scan."`
	Output      string            `help:"Output PNG file path." short:"o" required:""`
	Size        metric.MetricName `help:"Metric for rectangle area (file-size, file-lines, file-age, file-freshness, author-count)." short:"s" required:"" enum:"file-size,file-lines,file-age,file-freshness,author-count"`
	Fill        string            `help:"Metric for fill colour (file-size, file-lines, file-type, file-age, file-freshness, author-count)." short:"f" optional:"" default:""`
	FillPalette string            `help:"Palette for fill colour (categorization, temperature, good-bad, neutral)." optional:"" default:"" name:"fill-palette"`
	Verbose     bool              `help:"Enable debug-level logging." short:"v"`
	Format      string            `help:"Diagnostic/error output format (text, json)." enum:"text,json" default:"text"`
	Width       int               `help:"Image width in pixels." default:"1920"`
	Height      int               `help:"Image height in pixels." default:"1080"`
}

func (c *CLI) Validate() error {
	// Check target path exists and is a directory
	info, err := os.Stat(c.TargetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target path does not exist: %s", c.TargetPath)
		}
		return fmt.Errorf("cannot access target path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target path is not a directory: %s", c.TargetPath)
	}

	// Check output parent directory exists
	outDir := filepath.Dir(c.Output)
	if outDir != "." {
		info, err = os.Stat(outDir)
		if err != nil {
			return fmt.Errorf("output directory does not exist: %s", outDir)
		}
		if !info.IsDir() {
			return fmt.Errorf("output parent is not a directory: %s", outDir)
		}
	}

	// Size metric must be numeric (already enforced by enum, but belt-and-suspenders)
	if !c.Size.IsNumeric() {
		return fmt.Errorf("size metric must be numeric, got %q", c.Size)
	}

	// Validate fill metric if specified
	if c.Fill != "" {
		fm := metric.MetricName(c.Fill)
		if !fm.IsValid() {
			return fmt.Errorf("invalid fill metric %q", c.Fill)
		}
	}

	// Validate fill palette if specified
	if c.FillPalette != "" {
		fp := palette.PaletteName(c.FillPalette)
		if !fp.IsValid() {
			return fmt.Errorf("invalid fill palette %q", c.FillPalette)
		}
	}

	return nil
}

func (c *CLI) Run() error {
	setupLogger(c.Verbose)

	// Default fill to size metric if not specified
	fillMetric := metric.MetricName(c.Fill)
	if c.Fill == "" {
		fillMetric = c.Size
	}

	// Resolve fill palette
	fillPaletteName := palette.PaletteName(c.FillPalette)
	if c.FillPalette == "" {
		if p, ok := metric.DefaultPaletteFor(fillMetric); ok {
			fillPaletteName = p
		} else {
			fillPaletteName = palette.Neutral
		}
	}

	slog.Debug("scanning directory", "path", c.TargetPath)

	root, err := scan.Scan(c.TargetPath)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Count lines for all files if needed
	if c.Size == metric.FileLines || fillMetric == metric.FileLines {
		scan.PopulateLineCounts(&root)
	}

	files, dirs := countAll(root)
	slog.Debug("scan complete", "files", files, "directories", dirs)

	rects := treemap.Layout(root, c.Width, c.Height)

	// Apply fill colours
	fillPalette := palette.GetPalette(fillPaletteName)
	if fillMetric.IsNumeric() {
		values := collectNumericValues(root, fillMetric)
		if len(values) > 0 {
			buckets := metric.ComputeBuckets(values, len(fillPalette.Colours))
			numBuckets := len(buckets.Boundaries) + 1
			applyNumericFillColours(&rects, root, fillMetric, buckets, numBuckets, fillPalette)
		}
	} else {
		// Categorical (file-type)
		types := collectDistinctTypes(root)
		mapper := palette.NewCategoricalMapper(types, fillPalette)
		applyCategoricalFillColours(&rects, root, mapper)
	}

	slog.Debug("rendering", "width", c.Width, "height", c.Height, "output", c.Output)

	if err := render.RenderPNG(rects, c.Width, c.Height, c.Output); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	// Success output
	if c.Format == "json" {
		out := map[string]any{
			"files":          files,
			"directories":    dirs,
			"output":         c.Output,
			"width":          c.Width,
			"height":         c.Height,
			"size_metric":    string(c.Size),
			"fill_metric":    string(fillMetric),
			"fill_palette":   string(fillPaletteName),
			"border_metric":  nil,
			"border_palette": nil,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("Rendered treemap: %d files, %d directories → %s (%d×%d)\n",
		files, dirs, c.Output, c.Width, c.Height)

	return nil
}

func collectNumericValues(root scan.DirectoryNode, m metric.MetricName) []float64 {
	var values []float64
	collectNumericValuesRecursive(root, m, &values)
	return values
}

func collectNumericValuesRecursive(node scan.DirectoryNode, m metric.MetricName, values *[]float64) {
	for _, f := range node.Files {
		*values = append(*values, extractNumeric(f, m))
	}
	for _, d := range node.Dirs {
		collectNumericValuesRecursive(d, m, values)
	}
}

func extractNumeric(f scan.FileNode, m metric.MetricName) float64 {
	switch m {
	case metric.FileSize:
		return metric.ExtractFileSize(f)
	case metric.FileLines:
		return metric.ExtractFileLines(f)
	default:
		return 0
	}
}

func collectDistinctTypes(root scan.DirectoryNode) []string {
	seen := map[string]bool{}
	collectTypesRecursive(root, seen)
	types := make([]string, 0, len(seen))
	for t := range seen {
		types = append(types, t)
	}
	return types
}

func collectTypesRecursive(node scan.DirectoryNode, seen map[string]bool) {
	for _, f := range node.Files {
		seen[f.FileType] = true
	}
	for _, d := range node.Dirs {
		collectTypesRecursive(d, seen)
	}
}

func applyNumericFillColours(rect *treemap.TreemapRectangle, node scan.DirectoryNode, m metric.MetricName, buckets metric.BucketBoundaries, numBuckets int, p palette.ColourPalette) {
	fileIdx := 0
	dirIdx := 0
	for i := range rect.Children {
		child := &rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			applyNumericFillColours(child, node.Dirs[dirIdx], m, buckets, numBuckets, p)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			val := extractNumeric(node.Files[fileIdx], m)
			idx := buckets.BucketIndex(val)
			child.FillColour = palette.MapNumericToColour(idx, numBuckets, p)
			fileIdx++
		}
	}
}

func applyCategoricalFillColours(rect *treemap.TreemapRectangle, node scan.DirectoryNode, mapper *palette.CategoricalMapper) {
	fileIdx := 0
	dirIdx := 0
	for i := range rect.Children {
		child := &rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			applyCategoricalFillColours(child, node.Dirs[dirIdx], mapper)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			child.FillColour = mapper.Map(node.Files[fileIdx].FileType)
			fileIdx++
		}
	}
}

func setupLogger(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

func countAll(node scan.DirectoryNode) (files int, dirs int) {
	files = len(node.Files)
	for _, d := range node.Dirs {
		dirs++
		f, d2 := countAll(d)
		files += f
		dirs += d2
	}
	return
}

func main() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("codeviz"),
		kong.Description("Generate treemap visualizations of file trees."),
		kong.UsageOnError(),
	)

	err := ctx.Run(&cli)
	if err != nil {
		switch {
		case isValidationError(err):
			exitWithError(cli.Format, err, 1)
		case isPathError(err):
			exitWithError(cli.Format, err, 2)
		case isOutputError(err):
			exitWithError(cli.Format, err, 4)
		default:
			exitWithError(cli.Format, err, 5)
		}
	}
}

func exitWithError(format string, err error, code int) {
	if format == "json" {
		out := map[string]any{
			"error": err.Error(),
			"code":  code,
		}
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(out)
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	}
	os.Exit(code)
}

func isValidationError(err error) bool {
	// Validation errors from Kong or our Validate()
	return false
}

func isPathError(err error) bool {
	return false
}

func isOutputError(err error) bool {
	return false
}
