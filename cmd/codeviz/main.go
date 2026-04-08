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
	Verbose bool   `help:"Enable debug-level logging." short:"v"`
	Format  string `default:"text" enum:"text,json" help:"Diagnostic/error output format (text, json)."`

	Render RenderCmd `cmd:"" help:"Render a visualization."`
}

type RenderCmd struct {
	Treemap TreemapCmd `cmd:"" help:"Generate a treemap visualization."`
}

type TreemapCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output PNG file path." required:"true" short:"o"`

	Size metric.MetricName `enum:"file-size,file-lines,file-age,file-freshness,author-count" help:"Metric for rectangle area." required:"true" short:"s"` //nolint:revive // kong struct tags require long lines

	Fill          string `default:"" enum:",file-size,file-lines,file-type,file-age,file-freshness,author-count" help:"Metric for fill colour." optional:"" short:"f"`   //nolint:revive // kong struct tags require long lines
	FillPalette   string `default:"" enum:",categorization,temperature,good-bad,neutral" help:"Palette for fill colour." name:"fill-palette" optional:""`                //nolint:revive // kong struct tags require long lines
	Border        string `default:"" enum:",file-size,file-lines,file-type,file-age,file-freshness,author-count" help:"Metric for border colour." optional:"" short:"b"` //nolint:revive // kong struct tags require long lines
	BorderPalette string `default:"" enum:",categorization,temperature,good-bad,neutral" help:"Palette for border colour." name:"border-palette" optional:""`            //nolint:revive // kong struct tags require long lines

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1080" help:"Image height in pixels."`
}

func (c *TreemapCmd) Validate() error {
	if !c.Size.IsNumeric() {
		return eris.Errorf("size metric must be numeric, got %q", c.Size)
	}

	if err := validateMetricPalette(c.Fill, c.FillPalette, "fill"); err != nil {
		return err
	}

	if err := validateMetricPalette(c.Border, c.BorderPalette, "border"); err != nil {
		return err
	}

	if c.BorderPalette != "" && c.Border == "" {
		return eris.New("--border-palette requires --border to be specified")
	}

	return nil
}

func validateMetricPalette(metricStr, paletteStr, label string) error {
	if metricStr != "" {
		m := metric.MetricName(metricStr)
		if !m.IsValid() {
			return eris.Errorf("invalid %s metric %q", label, metricStr)
		}
	}

	if paletteStr != "" {
		p := palette.PaletteName(paletteStr)
		if !p.IsValid() {
			return eris.Errorf("invalid %s palette %q", label, paletteStr)
		}
	}

	return nil
}

func (c *TreemapCmd) Run(cli *CLI) error {
	setupLogger(cli.Verbose)

	if err := c.validatePaths(); err != nil {
		return err
	}

	fillMetric := c.resolveFillMetric()
	fillPaletteName := c.resolveFillPalette(fillMetric)

	slog.Debug("scanning directory", "path", c.TargetPath)

	root, err := scan.Scan(c.TargetPath)
	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	if err := c.enrichGitMetadata(&root, fillMetric); err != nil {
		return err
	}

	borderMetricName := metric.MetricName(c.Border)

	if c.needsLineCounts(fillMetric, borderMetricName) {
		scan.PopulateLineCounts(&root)
	}

	if err := c.filterBinaryFiles(&root); err != nil {
		return err
	}

	files, dirs := countAll(root)
	slog.Debug("scan complete", "files", files, "directories", dirs)

	rects := treemap.Layout(root, c.Width, c.Height)

	applyFillColours(&rects, root, fillMetric, fillPaletteName)

	borderMetric, borderPaletteName := c.applyBorderColours(&rects, root, fillMetric)

	slog.Debug("rendering", "width", c.Width, "height", c.Height, "output", c.Output)

	if err := render.RenderPNG(rects, c.Width, c.Height, c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	return c.printResult(cli.Format, files, dirs, fillMetric, fillPaletteName, borderMetric, borderPaletteName)
}

func (c *TreemapCmd) validatePaths() error {
	info, err := os.Stat(c.TargetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &targetPathError{msg: "target path does not exist: " + c.TargetPath}
		}

		return &targetPathError{msg: fmt.Sprintf("cannot access target path: %s", err)}
	}

	if !info.IsDir() {
		return &targetPathError{msg: "target path is not a directory: " + c.TargetPath}
	}

	outDir := filepath.Dir(c.Output)
	if outDir == "." {
		return nil
	}

	info, err = os.Stat(outDir)
	if err != nil {
		return &outputPathError{msg: "output directory does not exist: " + outDir}
	}

	if !info.IsDir() {
		return &outputPathError{msg: "output parent is not a directory: " + outDir}
	}

	return nil
}

func (c *TreemapCmd) resolveFillMetric() metric.MetricName {
	if c.Fill == "" {
		return c.Size
	}

	return metric.MetricName(c.Fill)
}

func (c *TreemapCmd) resolveFillPalette(fillMetric metric.MetricName) palette.PaletteName {
	if c.FillPalette != "" {
		return palette.PaletteName(c.FillPalette)
	}

	if p, ok := metric.DefaultPaletteFor(fillMetric); ok {
		return p
	}

	return palette.Neutral
}

func (c *TreemapCmd) resolveGitMetric(fillMetric metric.MetricName) metric.MetricName {
	borderMetricName := metric.MetricName(c.Border)

	switch {
	case c.Size.IsGitRequired():
		return c.Size
	case fillMetric.IsGitRequired():
		return fillMetric
	case c.Border != "" && borderMetricName.IsGitRequired():
		return borderMetricName
	default:
		return ""
	}
}

func (c *TreemapCmd) enrichGitMetadata(root *scan.DirectoryNode, fillMetric metric.MetricName) error {
	gitMetric := c.resolveGitMetric(fillMetric)
	needsGit := gitMetric != ""
	needsBinaryDetection := c.Size == metric.FileLines && !needsGit

	if !needsGit && !needsBinaryDetection {
		return nil
	}

	absPath, err := filepath.Abs(c.TargetPath)
	if err != nil {
		return eris.Wrap(err, "failed to resolve absolute path")
	}

	isGit, err := scan.IsGitRepo(absPath)
	if err != nil {
		return eris.Wrap(err, "git check failed")
	}

	if !isGit && needsGit {
		return &gitRequiredError{metric: gitMetric, target: c.TargetPath}
	}

	if isGit {
		info, err := scan.NewGitInfo(absPath)
		if err != nil {
			return eris.Wrap(err, "failed to open git repo")
		}

		scan.EnrichWithGitMetadata(root, info, absPath)
		info.ClearCache()
	}

	return nil
}

func (c *TreemapCmd) needsLineCounts(fillMetric, borderMetric metric.MetricName) bool {
	return c.Size == metric.FileLines ||
		fillMetric == metric.FileLines ||
		(c.Border != "" && borderMetric == metric.FileLines)
}

func (c *TreemapCmd) filterBinaryFiles(root *scan.DirectoryNode) error {
	if c.Size != metric.FileLines {
		return nil
	}

	beforeCount, _ := countAll(*root)
	*root = scan.FilterBinaryFiles(*root)
	afterCount, _ := countAll(*root)
	excluded := beforeCount - afterCount
	slog.Debug("binary file filter complete", "excluded", excluded, "remaining", afterCount)

	if afterCount == 0 {
		return &noFilesAfterFilterError{
			msg: "no files available for visualization after excluding binary files",
		}
	}

	return nil
}

func applyFillColours(
	rects *treemap.TreemapRectangle,
	root scan.DirectoryNode,
	fillMetric metric.MetricName,
	fillPaletteName palette.PaletteName,
) {
	fillPalette := palette.GetPalette(fillPaletteName)

	if fillMetric.IsNumeric() {
		values := collectNumericValues(root, fillMetric)
		if len(values) > 0 {
			buckets := metric.ComputeBuckets(values, len(fillPalette.Colours))
			numBuckets := len(buckets.Boundaries) + 1
			applyNumericFillColours(rects, root, fillMetric, buckets, numBuckets, fillPalette)
		}
	} else {
		types := collectDistinctTypes(root)
		mapper := palette.NewCategoricalMapper(types, fillPalette)
		applyCategoricalFillColours(rects, root, mapper)
	}
}

func (c *TreemapCmd) applyBorderColours(
	rects *treemap.TreemapRectangle,
	root scan.DirectoryNode,
	fillMetric metric.MetricName,
) (metric.MetricName, palette.PaletteName) {
	if c.Border == "" {
		return "", ""
	}

	borderMetric := metric.MetricName(c.Border)

	borderPaletteName := palette.PaletteName(c.BorderPalette)
	if c.BorderPalette == "" {
		if p, ok := metric.DefaultPaletteFor(borderMetric); ok {
			borderPaletteName = p
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	if borderMetric == metric.FileLines &&
		c.Size != metric.FileLines &&
		fillMetric != metric.FileLines {
		scan.PopulateLineCounts(&root)
	}

	borderPalette := palette.GetPalette(borderPaletteName)

	if borderMetric.IsNumeric() {
		values := collectNumericValues(root, borderMetric)
		if len(values) > 0 {
			buckets := metric.ComputeBuckets(values, len(borderPalette.Colours))
			numBuckets := len(buckets.Boundaries) + 1
			applyNumericBorderColours(rects, root, borderMetric, buckets, numBuckets, borderPalette)
		}
	} else {
		types := collectDistinctTypes(root)
		mapper := palette.NewCategoricalMapper(types, borderPalette)
		applyCategoricalBorderColours(rects, root, mapper)
	}

	return borderMetric, borderPaletteName
}

func (c *TreemapCmd) printResult(
	format string,
	files, dirs int,
	fillMetric metric.MetricName,
	fillPaletteName palette.PaletteName,
	borderMetric metric.MetricName,
	borderPaletteName palette.PaletteName,
) error {
	if format != "json" {
		fmt.Printf("Rendered treemap: %d files, %d directories → %s (%d×%d)\n",
			files, dirs, c.Output, c.Width, c.Height)

		return nil
	}

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

	return eris.Wrap(enc.Encode(out), "failed to encode JSON output")
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

func applyNumericFillColours(
	rect *treemap.TreemapRectangle,
	node scan.DirectoryNode,
	m metric.MetricName,
	buckets metric.BucketBoundaries,
	numBuckets int,
	p palette.ColourPalette,
) {
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

func applyCategoricalFillColours(
	rect *treemap.TreemapRectangle,
	node scan.DirectoryNode,
	mapper *palette.CategoricalMapper,
) {
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

func applyNumericBorderColours(
	rect *treemap.TreemapRectangle,
	node scan.DirectoryNode,
	m metric.MetricName,
	buckets metric.BucketBoundaries,
	numBuckets int,
	p palette.ColourPalette,
) {
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

func applyCategoricalBorderColours(
	rect *treemap.TreemapRectangle,
	node scan.DirectoryNode,
	mapper *palette.CategoricalMapper,
) {
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

func setupLogger(verbose bool) { //nolint:revive // flag-parameter: boolean toggle is idiomatic for log verbosity
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

	return files, dirs
}

func main() {
	cli := CLI{}

	parser, err := kong.New(&cli,
		kong.Name("codeviz"),
		kong.Description("Generate visualizations of file trees."),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(5)
	}

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		// Kong parse/validation errors are argument failures → show help, then exit 1
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) && parseErr.Context != nil {
			_ = parseErr.Context.PrintUsage(false)
		}

		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	err = ctx.Run(&cli)
	if err != nil {
		code := classifyError(err)
		exitWithError(cli.Format, err, code)
	}
}

func classifyError(err error) int {
	var (
		gitErr     *gitRequiredError
		targetErr  *targetPathError
		outputErr  *outputPathError
		noFilesErr *noFilesAfterFilterError
	)

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
		out := struct {
			Error string `json:"error"`
			Code  int    `json:"code"`
		}{
			Error: err.Error(),
			Code:  code,
		}

		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")

		if encErr := enc.Encode(out); encErr != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		}
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	}

	os.Exit(code) //nolint:revive // deep-exit: intentional exit from CLI error handler called by main
}
