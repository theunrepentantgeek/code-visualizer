package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/config"
	"github.com/bevan/code-visualizer/internal/filter"
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/render"
	"github.com/bevan/code-visualizer/internal/scan"
	"github.com/bevan/code-visualizer/internal/treemap"
)

type TreemapCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output PNG file path." required:"true" short:"o"`

	Size metric.Name `enum:"file-size,file-lines,file-age,file-freshness,author-count" help:"Metric for rectangle area." required:"true" short:"s"` //nolint:revive // kong struct tags require long lines

	Fill          string `default:"" enum:",file-size,file-lines,file-type,file-age,file-freshness,author-count" help:"Metric for fill colour." optional:"" short:"f"`   //nolint:revive // kong struct tags require long lines
	FillPalette   string `default:"" enum:",categorization,temperature,good-bad,neutral" help:"Palette for fill colour." name:"fill-palette" optional:""`                //nolint:revive // kong struct tags require long lines
	Border        string `default:"" enum:",file-size,file-lines,file-type,file-age,file-freshness,author-count" help:"Metric for border colour." optional:"" short:"b"` //nolint:revive // kong struct tags require long lines
	BorderPalette string `default:"" enum:",categorization,temperature,good-bad,neutral" help:"Palette for border colour." name:"border-palette" optional:""`            //nolint:revive // kong struct tags require long lines

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1080" help:"Image height in pixels."`

	Filter []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)."` //nolint:revive // kong struct tags require long lines
}

func (c *TreemapCmd) Validate() error {
	p, ok := provider.Get(c.Size)
	if !ok {
		return eris.Errorf("unknown size metric %q", c.Size)
	}

	if p.Kind() != metric.Quantity {
		return eris.Errorf("size metric must be numeric, got %q (kind: %d)", c.Size, p.Kind())
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

	for _, f := range c.Filter {
		if _, err := filter.ParseFilterFlag(f); err != nil {
			return eris.Wrapf(err, "invalid filter %q", f)
		}
	}

	return nil
}

func validateMetricPalette(metricStr, paletteStr, label string) error {
	if metricStr != "" {
		if _, ok := provider.Get(metric.Name(metricStr)); !ok {
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

func collectRequestedMetrics(size metric.Name, fill, border string) []metric.Name {
	seen := map[metric.Name]bool{size: true}
	names := []metric.Name{size}

	for _, s := range []string{fill, border} {
		if s != "" {
			n := metric.Name(s)
			if !seen[n] {
				seen[n] = true
				names = append(names, n)
			}
		}
	}

	return names
}

func (c *TreemapCmd) Run(flags *Flags) error {
	c.applyOverrides(flags.Config)
	cfg := flags.Config.Treemap

	if err := c.validatePaths(); err != nil {
		return err
	}

	if flags.ExportConfig != "" {
		if err := flags.Config.Save(flags.ExportConfig); err != nil {
			return eris.Wrap(err, "failed to save config")
		}
	}

	fillMetric := c.resolveFillMetric(cfg)
	fillPaletteName := c.resolveFillPalette(cfg, fillMetric)

	filterRules := c.buildFilterRules(flags.Config)

	slog.Debug("scanning directory", "path", c.TargetPath)

	root, err := scan.Scan(c.TargetPath, filterRules)
	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	// Collect all requested metrics and run providers
	requested := collectRequestedMetrics(c.Size, ptrString(cfg.Fill), ptrString(cfg.Border))

	// Check git requirement before running providers
	if err := c.checkGitRequirement(requested); err != nil {
		return err
	}

	if err := provider.Run(root, requested); err != nil {
		return eris.Wrap(err, "failed to load metrics")
	}

	if err := c.filterBinaryFiles(cfg, root); err != nil {
		return err
	}

	files, dirs := countAll(root)
	slog.Debug("scan complete", "files", files, "directories", dirs)

	width := ptrInt(flags.Config.Width, 1920)
	height := ptrInt(flags.Config.Height, 1080)

	rects := treemap.Layout(root, width, height, c.Size)

	applyFillColours(&rects, root, fillMetric, fillPaletteName)

	borderMetric, borderPaletteName := c.applyBorderColours(&rects, root, cfg)

	slog.Debug("rendering", "width", width, "height", height, "output", c.Output)

	if err := render.RenderPNG(rects, width, height, c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	return c.printResult(flags, renderResult{
		files: files, dirs: dirs,
		width: width, height: height,
		fillMetric: fillMetric, fillPaletteName: fillPaletteName,
		borderMetric: borderMetric, borderPaletteName: borderPaletteName,
	})
}

func (c *TreemapCmd) buildFilterRules(cfg *config.Config) []filter.Rule {
	rules := make([]filter.Rule, 0, len(cfg.FileFilter)+len(c.Filter))
	rules = append(rules, cfg.FileFilter...)

	for _, f := range c.Filter {
		// Already validated in Validate()
		rule, _ := filter.ParseFilterFlag(f)
		rules = append(rules, rule)
	}

	return rules
}

func (c *TreemapCmd) checkGitRequirement(requested []metric.Name) error {
	name, needsGit := findGitMetric(requested)
	if !needsGit {
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

	if !isGit {
		return &gitRequiredError{metric: name, target: c.TargetPath}
	}

	return nil
}

func findGitMetric(requested []metric.Name) (metric.Name, bool) {
	gitMetrics := map[metric.Name]bool{
		"file-age": true, "file-freshness": true, "author-count": true,
	}

	for _, name := range requested {
		if gitMetrics[name] {
			return name, true
		}
	}

	return "", false
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

func (c *TreemapCmd) resolveFillMetric(cfg *config.Treemap) metric.Name {
	if fill := ptrString(cfg.Fill); fill != "" {
		return metric.Name(fill)
	}

	return c.Size
}

func (*TreemapCmd) resolveFillPalette(cfg *config.Treemap, fillMetric metric.Name) palette.PaletteName {
	if fp := ptrString(cfg.FillPalette); fp != "" {
		return palette.PaletteName(fp)
	}

	if p, ok := provider.Get(fillMetric); ok {
		return p.DefaultPalette()
	}

	return palette.Neutral
}

func (c *TreemapCmd) filterBinaryFiles(_ *config.Treemap, root *model.Directory) error {
	if c.Size != filesystem.FileLines {
		return nil
	}

	beforeCount, _ := countAll(root)
	filtered := scan.FilterBinaryFiles(root)
	afterCount, _ := countAll(filtered)
	excluded := beforeCount - afterCount
	slog.Debug("binary file filter complete", "excluded", excluded, "remaining", afterCount)

	if afterCount == 0 {
		return &noFilesAfterFilterError{
			msg: "no files available for visualization after excluding binary files",
		}
	}
	// Update root in place — avoid struct copy which would copy the mutex.
	root.Files = filtered.Files
	root.Dirs = filtered.Dirs

	return nil
}

func applyFillColours(
	rects *treemap.TreemapRectangle,
	root *model.Directory,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
) {
	fillPalette := palette.GetPalette(fillPaletteName)

	p, ok := provider.Get(fillMetric)
	if !ok {
		return
	}

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
		values := collectNumericValues(root, fillMetric)
		if len(values) > 0 {
			buckets := metric.ComputeBuckets(values, len(fillPalette.Colours))
			numBuckets := len(buckets.Boundaries) + 1
			applyNumericFillColours(rects, root, fillMetric, buckets, numBuckets, fillPalette)
		}
	} else {
		types := collectDistinctTypes(root, fillMetric)
		mapper := palette.NewCategoricalMapper(types, fillPalette)
		applyCategoricalFillColours(rects, root, fillMetric, mapper)
	}
}

func (*TreemapCmd) applyBorderColours(
	rects *treemap.TreemapRectangle,
	root *model.Directory,
	cfg *config.Treemap,
) (metric.Name, palette.PaletteName) {
	border := ptrString(cfg.Border)
	if border == "" {
		return "", ""
	}

	borderMetric := metric.Name(border)

	borderPaletteName := palette.PaletteName(ptrString(cfg.BorderPalette))
	if ptrString(cfg.BorderPalette) == "" {
		if p, ok := provider.Get(borderMetric); ok {
			borderPaletteName = p.DefaultPalette()
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	borderPalette := palette.GetPalette(borderPaletteName)

	p, ok := provider.Get(borderMetric)
	if !ok {
		return borderMetric, borderPaletteName
	}

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
		values := collectNumericValues(root, borderMetric)
		if len(values) > 0 {
			buckets := metric.ComputeBuckets(values, len(borderPalette.Colours))
			numBuckets := len(buckets.Boundaries) + 1
			applyNumericBorderColours(rects, root, borderMetric, buckets, numBuckets, borderPalette)
		}
	} else {
		types := collectDistinctTypes(root, borderMetric)
		mapper := palette.NewCategoricalMapper(types, borderPalette)
		applyCategoricalBorderColours(rects, root, borderMetric, mapper)
	}

	return borderMetric, borderPaletteName
}

type renderResult struct {
	files, dirs       int
	width, height     int
	fillMetric        metric.Name
	fillPaletteName   palette.PaletteName
	borderMetric      metric.Name
	borderPaletteName palette.PaletteName
}

func (c *TreemapCmd) printResult(flags *Flags, r renderResult) error {
	if flags.Format != "json" {
		fmt.Printf("Rendered treemap: %d files, %d directories → %s (%d×%d)\n",
			r.files, r.dirs, c.Output, r.width, r.height)

		return nil
	}

	var bm, bp any
	if r.borderMetric != "" {
		bm = string(r.borderMetric)
		bp = string(r.borderPaletteName)
	}

	out := map[string]any{
		"files":          r.files,
		"directories":    r.dirs,
		"output":         c.Output,
		"width":          r.width,
		"height":         r.height,
		"size_metric":    string(c.Size),
		"fill_metric":    string(r.fillMetric),
		"fill_palette":   string(r.fillPaletteName),
		"border_metric":  bm,
		"border_palette": bp,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	return eris.Wrap(enc.Encode(out), "failed to encode JSON output")
}

func extractNumeric(f *model.File, m metric.Name) float64 {
	if v, ok := f.Quantity(m); ok {
		return float64(v)
	}

	if v, ok := f.Measure(m); ok {
		return v
	}

	return 0
}

func collectNumericValues(root *model.Directory, m metric.Name) []float64 {
	var values []float64

	model.WalkFiles(root, func(f *model.File) {
		values = append(values, extractNumeric(f, m))
	})

	return values
}

func collectDistinctTypes(root *model.Directory, m metric.Name) []string {
	seen := map[string]bool{}

	model.WalkFiles(root, func(f *model.File) {
		if v, ok := f.Classification(m); ok {
			seen[v] = true
		}
	})

	types := make([]string, 0, len(seen))
	for t := range seen {
		types = append(types, t)
	}

	sort.Strings(types)

	return types
}

func applyNumericFillColours(
	rect *treemap.TreemapRectangle,
	node *model.Directory,
	m metric.Name,
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
	node *model.Directory,
	m metric.Name,
	mapper *palette.CategoricalMapper,
) {
	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := &rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			applyCategoricalFillColours(child, node.Dirs[dirIdx], m, mapper)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			if v, ok := node.Files[fileIdx].Classification(m); ok {
				child.FillColour = mapper.Map(v)
			}

			fileIdx++
		}
	}
}

func applyNumericBorderColours(
	rect *treemap.TreemapRectangle,
	node *model.Directory,
	m metric.Name,
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
	node *model.Directory,
	m metric.Name,
	mapper *palette.CategoricalMapper,
) {
	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := &rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			applyCategoricalBorderColours(child, node.Dirs[dirIdx], m, mapper)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			if v, ok := node.Files[fileIdx].Classification(m); ok {
				col := mapper.Map(v)
				child.BorderColour = &col
			}

			fileIdx++
		}
	}
}
