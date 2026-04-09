package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/config"
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/render"
	"github.com/bevan/code-visualizer/internal/scan"
	"github.com/bevan/code-visualizer/internal/treemap"
)

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

func (c *TreemapCmd) Run(flags *Flags) error {
	c.applyOverrides(flags.Config)

	cfg := flags.Config.Treemap

	if err := c.validatePaths(); err != nil {
		return err
	}

	if flags.ExportConfig != "" {
		if err := flags.Config.Save(flags.ExportConfig); err != nil {
			return err
		}
	}

	fillMetric := c.resolveFillMetric(cfg)
	fillPaletteName := c.resolveFillPalette(cfg, fillMetric)

	slog.Debug("scanning directory", "path", c.TargetPath)

	root, err := scan.Scan(c.TargetPath)
	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	if err := c.enrichGitMetadata(&root, cfg, fillMetric); err != nil {
		return err
	}

	borderMetricName := metric.MetricName(ptrString(cfg.Border))

	if c.needsLineCounts(cfg, fillMetric, borderMetricName) {
		scan.PopulateLineCounts(&root)
	}

	if err := c.filterBinaryFiles(cfg, &root); err != nil {
		return err
	}

	files, dirs := countAll(root)
	slog.Debug("scan complete", "files", files, "directories", dirs)

	width := ptrInt(flags.Config.Width, 1920)
	height := ptrInt(flags.Config.Height, 1080)

	rects := treemap.Layout(root, width, height)

	applyFillColours(&rects, root, fillMetric, fillPaletteName)

	borderMetric, borderPaletteName := c.applyBorderColours(&rects, root, cfg, fillMetric)

	slog.Debug("rendering", "width", width, "height", height, "output", c.Output)

	if err := render.RenderPNG(rects, width, height, c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	return c.printResult(flags.Format, files, dirs, width, height, fillMetric, fillPaletteName, borderMetric, borderPaletteName)
}

// applyOverrides writes non-zero CLI flag values on top of the config layer.
// Zero-valued CLI fields are transparent — the config value passes through unchanged.
func (c *TreemapCmd) applyOverrides(cfg *config.Config) {
	if c.Width != 0 {
		cfg.Width = &c.Width
	}

	if c.Height != 0 {
		cfg.Height = &c.Height
	}

	if c.Fill != "" {
		cfg.Treemap.Fill = &c.Fill
	}

	if c.FillPalette != "" {
		cfg.Treemap.FillPalette = &c.FillPalette
	}

	if c.Border != "" {
		cfg.Treemap.Border = &c.Border
	}

	if c.BorderPalette != "" {
		cfg.Treemap.BorderPalette = &c.BorderPalette
	}
}

// ptrString safely dereferences a *string, returning "" if nil.
func ptrString(p *string) string {
	if p == nil {
		return ""
	}

	return *p
}

// ptrInt safely dereferences a *int, returning fallback if nil.
func ptrInt(p *int, fallback int) int {
	if p == nil {
		return fallback
	}

	return *p
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

func (c *TreemapCmd) resolveFillMetric(cfg *config.Treemap) metric.MetricName {
	if fill := ptrString(cfg.Fill); fill != "" {
		return metric.MetricName(fill)
	}

	return c.Size
}

func (c *TreemapCmd) resolveFillPalette(cfg *config.Treemap, fillMetric metric.MetricName) palette.PaletteName {
	if fp := ptrString(cfg.FillPalette); fp != "" {
		return palette.PaletteName(fp)
	}

	if p, ok := metric.DefaultPaletteFor(fillMetric); ok {
		return p
	}

	return palette.Neutral
}

func (c *TreemapCmd) resolveGitMetric(cfg *config.Treemap, fillMetric metric.MetricName) metric.MetricName {
	borderMetricName := metric.MetricName(ptrString(cfg.Border))

	switch {
	case c.Size.IsGitRequired():
		return c.Size
	case fillMetric.IsGitRequired():
		return fillMetric
	case ptrString(cfg.Border) != "" && borderMetricName.IsGitRequired():
		return borderMetricName
	default:
		return ""
	}
}

func (c *TreemapCmd) enrichGitMetadata(root *scan.DirectoryNode, cfg *config.Treemap, fillMetric metric.MetricName) error {
	gitMetric := c.resolveGitMetric(cfg, fillMetric)
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

func (c *TreemapCmd) needsLineCounts(cfg *config.Treemap, fillMetric, borderMetric metric.MetricName) bool {
	return c.Size == metric.FileLines ||
		fillMetric == metric.FileLines ||
		(ptrString(cfg.Border) != "" && borderMetric == metric.FileLines)
}

func (c *TreemapCmd) filterBinaryFiles(cfg *config.Treemap, root *scan.DirectoryNode) error {
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
	cfg *config.Treemap,
	fillMetric metric.MetricName,
) (metric.MetricName, palette.PaletteName) {
	border := ptrString(cfg.Border)
	if border == "" {
		return "", ""
	}

	borderMetric := metric.MetricName(border)

	borderPaletteName := palette.PaletteName(ptrString(cfg.BorderPalette))
	if ptrString(cfg.BorderPalette) == "" {
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
	width, height int,
	fillMetric metric.MetricName,
	fillPaletteName palette.PaletteName,
	borderMetric metric.MetricName,
	borderPaletteName palette.PaletteName,
) error {
	if format != "json" {
		fmt.Printf("Rendered treemap: %d files, %d directories → %s (%d×%d)\n",
			files, dirs, c.Output, width, height)

		return nil
	}

	var bm, bp any
	if borderMetric != "" {
		bm = string(borderMetric)
		bp = string(borderPaletteName)
	}

	out := map[string]any{
		"files":          files,
		"directories":    dirs,
		"output":         c.Output,
		"width":          width,
		"height":         height,
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
