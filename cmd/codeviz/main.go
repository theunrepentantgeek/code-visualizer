package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/render"
	"github.com/bevan/code-visualizer/internal/scan"
	"github.com/bevan/code-visualizer/internal/treemap"
)

type CLI struct {
	TargetPath    string            `arg:"" help:"Path to directory to scan."`
	Output        string            `help:"Output PNG file path." short:"o" required:""`
	Size          metric.MetricName `help:"Metric for rectangle area (file-size, file-lines, file-age, file-freshness, author-count)." short:"s" required:"" enum:"file-size,file-lines,file-age,file-freshness,author-count"`
	Fill          string            `help:"Metric for fill colour (file-size, file-lines, file-type, file-age, file-freshness, author-count)." short:"f" optional:"" default:"" enum:",file-size,file-lines,file-type,file-age,file-freshness,author-count"`
	FillPalette   string            `help:"Palette for fill colour (categorization, temperature, good-bad, neutral)." optional:"" default:"" name:"fill-palette" enum:",categorization,temperature,good-bad,neutral"`
	Border        string            `help:"Metric for border colour (file-size, file-lines, file-type, file-age, file-freshness, author-count)." short:"b" optional:"" default:"" enum:",file-size,file-lines,file-type,file-age,file-freshness,author-count"`
	BorderPalette string            `help:"Palette for border colour (categorization, temperature, good-bad, neutral)." optional:"" default:"" name:"border-palette" enum:",categorization,temperature,good-bad,neutral"`
	Verbose       bool              `help:"Enable debug-level logging." short:"v"`
	Format        string            `help:"Diagnostic/error output format (text, json)." enum:"text,json" default:"text"`
	Width         int               `help:"Image width in pixels." default:"1920"`
	Height        int               `help:"Image height in pixels." default:"1080"`
}

func (c *CLI) Validate() error {
	// Validate args/metrics (exit code 1 errors)
	// Size metric must be numeric (already enforced by enum, but belt-and-suspenders)
	if !c.Size.IsNumeric() {
		return eris.Errorf("size metric must be numeric, got %q", c.Size)
	}

	// Validate fill metric if specified
	if c.Fill != "" {
		fm := metric.MetricName(c.Fill)
		if !fm.IsValid() {
			return eris.Errorf("invalid fill metric %q", c.Fill)
		}
	}

	// Validate fill palette if specified
	if c.FillPalette != "" {
		fp := palette.PaletteName(c.FillPalette)
		if !fp.IsValid() {
			return eris.Errorf("invalid fill palette %q", c.FillPalette)
		}
	}

	// Validate border metric if specified
	if c.Border != "" {
		bm := metric.MetricName(c.Border)
		if !bm.IsValid() {
			return eris.Errorf("invalid border metric %q", c.Border)
		}
	}

	// Validate border palette if specified
	if c.BorderPalette != "" {
		if c.Border == "" {
			return eris.New("--border-palette requires --border to be specified")
		}
		bp := palette.PaletteName(c.BorderPalette)
		if !bp.IsValid() {
			return eris.Errorf("invalid border palette %q", c.BorderPalette)
		}
	}

	return nil
}

func (c *CLI) Run() error {
	setupLogger(c.Verbose)

	// Validate target path (exit code 2)
	info, err := os.Stat(c.TargetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &targetPathError{msg: fmt.Sprintf("target path does not exist: %s", c.TargetPath)}
		}
		return &targetPathError{msg: fmt.Sprintf("cannot access target path: %s", err)}
	}
	if !info.IsDir() {
		return &targetPathError{msg: fmt.Sprintf("target path is not a directory: %s", c.TargetPath)}
	}

	// Validate output parent directory (exit code 4)
	outDir := filepath.Dir(c.Output)
	if outDir != "." {
		info, err = os.Stat(outDir)
		if err != nil {
			return &outputPathError{msg: fmt.Sprintf("output directory does not exist: %s", outDir)}
		}
		if !info.IsDir() {
			return &outputPathError{msg: fmt.Sprintf("output parent is not a directory: %s", outDir)}
		}
	}

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
		return eris.Wrap(err, "scan failed")
	}

	// Check if any metric requires git
	borderMetricName := metric.MetricName(c.Border)
	var gitMetric metric.MetricName
	switch {
	case c.Size.IsGitRequired():
		gitMetric = c.Size
	case fillMetric.IsGitRequired():
		gitMetric = fillMetric
	case c.Border != "" && borderMetricName.IsGitRequired():
		gitMetric = borderMetricName
	}
	needsGit := gitMetric != ""

	if needsGit {
		absPath, err := filepath.Abs(c.TargetPath)
		if err != nil {
			return eris.Wrap(err, "failed to resolve absolute path")
		}
		isGit, err := scan.IsGitRepo(absPath)
		if err != nil {
			return eris.Wrap(err, "git check failed")
		}
		if !isGit {
			return &gitRequiredError{metric: gitMetric, target: c.TargetPath}
		}
		info, err := scan.NewGitInfo(absPath)
		if err != nil {
			return eris.Wrap(err, "failed to open git repo")
		}
		scan.EnrichWithGitMetadata(&root, info, absPath)
		info.ClearCache()
	}

	// Count lines for all files if needed
	needsLines := c.Size == metric.FileLines || fillMetric == metric.FileLines || (c.Border != "" && borderMetricName == metric.FileLines)
	if needsLines {
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

	// Apply border colours if --border specified
	var borderMetric metric.MetricName
	var borderPaletteName palette.PaletteName
	if c.Border != "" {
		borderMetric = metric.MetricName(c.Border)
		borderPaletteName = palette.PaletteName(c.BorderPalette)
		if c.BorderPalette == "" {
			if p, ok := metric.DefaultPaletteFor(borderMetric); ok {
				borderPaletteName = p
			} else {
				borderPaletteName = palette.Neutral
			}
		}

		// Count lines if border metric needs it
		if borderMetric == metric.FileLines && c.Size != metric.FileLines && fillMetric != metric.FileLines {
			scan.PopulateLineCounts(&root)
		}

		borderPalette := palette.GetPalette(borderPaletteName)
		if borderMetric.IsNumeric() {
			values := collectNumericValues(root, borderMetric)
			if len(values) > 0 {
				buckets := metric.ComputeBuckets(values, len(borderPalette.Colours))
				numBuckets := len(buckets.Boundaries) + 1
				applyNumericBorderColours(&rects, root, borderMetric, buckets, numBuckets, borderPalette)
			}
		} else {
			types := collectDistinctTypes(root)
			mapper := palette.NewCategoricalMapper(types, borderPalette)
			applyCategoricalBorderColours(&rects, root, mapper)
		}
	}

	slog.Debug("rendering", "width", c.Width, "height", c.Height, "output", c.Output)

	if err := render.RenderPNG(rects, c.Width, c.Height, c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	// Success output
	if c.Format == "json" {
		var bm, bp any
		if c.Border != "" {
			bm = string(borderMetric)
			bp = string(borderPaletteName)
		}
		out := map[string]any{
			"files":          files,
			"directories":    dirs,
			"output":         c.Output,
			"width":          c.Width,
			"height":         c.Height,
			"size_metric":    string(c.Size),
			"fill_metric":    string(fillMetric),
			"fill_palette":   string(fillPaletteName),
			"border_metric":  bm,
			"border_palette": bp,
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
	case metric.FileAge:
		if f.Age != nil {
			return f.Age.Seconds()
		}
		return 0
	case metric.FileFreshness:
		if f.Freshness != nil {
			return f.Freshness.Seconds()
		}
		return 0
	case metric.AuthorCount:
		if f.AuthorCount != nil {
			return float64(*f.AuthorCount)
		}
		return 0
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

func applyNumericBorderColours(rect *treemap.TreemapRectangle, node scan.DirectoryNode, m metric.MetricName, buckets metric.BucketBoundaries, numBuckets int, p palette.ColourPalette) {
	fileIdx := 0
	dirIdx := 0
	for i := range rect.Children {
		child := &rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			applyNumericBorderColours(child, node.Dirs[dirIdx], m, buckets, numBuckets, p)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			val := extractNumeric(node.Files[fileIdx], m)
			idx := buckets.BucketIndex(val)
			col := palette.MapNumericToColour(idx, numBuckets, p)
			child.BorderColour = &col
			fileIdx++
		}
	}
}

func applyCategoricalBorderColours(rect *treemap.TreemapRectangle, node scan.DirectoryNode, mapper *palette.CategoricalMapper) {
	fileIdx := 0
	dirIdx := 0
	for i := range rect.Children {
		child := &rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			applyCategoricalBorderColours(child, node.Dirs[dirIdx], mapper)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			col := mapper.Map(node.Files[fileIdx].FileType)
			child.BorderColour = &col
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
	parser, err := kong.New(&cli,
		kong.Name("codeviz"),
		kong.Description("Generate treemap visualizations of file trees."),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(5)
	}

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		// Kong parse/validation errors are argument failures → exit 1
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	err = ctx.Run(&cli)
	if err != nil {
		code := classifyError(err, cli.Format)
		exitWithError(cli.Format, err, code)
	}
}

func classifyError(err error, format string) int {
	var gitErr *gitRequiredError
	var targetErr *targetPathError
	var outputErr *outputPathError
	var noFilesErr *noFilesAfterFilterError
	switch {
	case errors.As(err, &targetErr):
		return 2
	case errors.As(err, &gitErr):
		return 3
	case errors.As(err, &outputErr):
		return 4
	case errors.As(err, &noFilesErr):
		return 6
	default:
		return 5
	}
}

type gitRequiredError struct {
	metric metric.MetricName
	target string
}

func (e *gitRequiredError) Error() string {
	return fmt.Sprintf("metric %q requires a git repository, but %q is not a git repository", e.metric, e.target)
}

type targetPathError struct {
	msg string
}

func (e *targetPathError) Error() string { return e.msg }

type outputPathError struct {
	msg string
}

func (e *outputPathError) Error() string { return e.msg }

type noFilesAfterFilterError struct {
	msg string
}

func (e *noFilesAfterFilterError) Error() string { return e.msg }

func exitWithError(format string, err error, code int) {
	if format == "json" {
		out := map[string]any{
			"error": err.Error(),
			"code":  code,
		}
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	}
	os.Exit(code)
}
