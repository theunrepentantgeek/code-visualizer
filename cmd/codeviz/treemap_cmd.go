package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/config"
	"github.com/bevan/code-visualizer/internal/export"
	"github.com/bevan/code-visualizer/internal/filter"
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/provider/git"
	"github.com/bevan/code-visualizer/internal/render"
	"github.com/bevan/code-visualizer/internal/scan"
	"github.com/bevan/code-visualizer/internal/treemap"
)

type TreemapCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	Size metric.Name `default:"" enum:",file-size,file-lines,file-age,file-freshness,author-count" help:"Metric for rectangle area." short:"s"` //nolint:revive // kong struct tags require long lines

	Fill          string `default:"" enum:",file-size,file-lines,file-type,file-age,file-freshness,author-count" help:"Metric for fill colour." optional:"" short:"f"`   //nolint:revive // kong struct tags require long lines
	FillPalette   string `default:"" enum:",categorization,temperature,good-bad,neutral,foliage" help:"Palette for fill colour." name:"fill-palette" optional:""`        //nolint:revive // kong struct tags require long lines
	Border        string `default:"" enum:",file-size,file-lines,file-type,file-age,file-freshness,author-count" help:"Metric for border colour." optional:"" short:"b"` //nolint:revive // kong struct tags require long lines
	BorderPalette string `default:"" enum:",categorization,temperature,good-bad,neutral,foliage" help:"Palette for border colour." name:"border-palette" optional:""`    //nolint:revive // kong struct tags require long lines

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive // kong struct tags require long lines

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1080" help:"Image height in pixels."`

	Filter []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)."` //nolint:revive // kong struct tags require long lines
}

func (c *TreemapCmd) Validate() error {
	for _, f := range c.Filter {
		if _, err := filter.ParseFilterFlag(f); err != nil {
			return eris.Wrapf(err, "invalid filter %q", f)
		}
	}

	return nil
}

// validateConfig checks the effective configuration after all sources have been
// merged. Called from mergeConfigAndValidate() after TryAutoLoad + applyOverrides.
func (*TreemapCmd) validateConfig(cfg *config.Treemap) error {
	size := ptrString(cfg.Size)

	p, ok := provider.Get(metric.Name(size))
	if !ok {
		return eris.Errorf("unknown size metric %q; available metrics: %s", size, formatMetricNames())
	}

	if p.Kind() != metric.Quantity {
		return eris.Errorf("size metric must be numeric, got %q (kind: %d)", size, p.Kind())
	}

	if err := validateMetricPalette(ptrString(cfg.Fill), ptrString(cfg.FillPalette), "fill"); err != nil {
		return err
	}

	if err := validateMetricPalette(ptrString(cfg.Border), ptrString(cfg.BorderPalette), "border"); err != nil {
		return err
	}

	if ptrString(cfg.BorderPalette) != "" && ptrString(cfg.Border) == "" {
		return eris.New("--border-palette requires --border to be specified")
	}

	return nil
}

func validateMetricPalette(metricStr, paletteStr, label string) error {
	if metricStr != "" {
		if _, ok := provider.Get(metric.Name(metricStr)); !ok {
			return eris.Errorf("invalid %s metric %q; available metrics: %s", label, metricStr, formatMetricNames())
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

func formatMetricNames() string {
	names := provider.Names()
	strs := make([]string, len(names))

	for i, n := range names {
		strs[i] = string(n)
	}

	return strings.Join(strs, ", ")
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

// mergeConfigAndValidate loads the config file, merges CLI overrides on top,
// and validates the effective configuration. Called at the start of Run().
func (c *TreemapCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Treemap)
}

func (c *TreemapCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	cfg := flags.Config.Treemap
	size := metric.Name(ptrString(cfg.Size))

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

	slog.Info("Scanning filesystem", "path", c.TargetPath)

	scanProg, stopTicker := buildScanProgress(flags)
	defer stopTicker()

	root, err := scan.Scan(c.TargetPath, filterRules, scanProg)
	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	// Collect all requested metrics and run providers
	requested := collectRequestedMetrics(size, ptrString(cfg.Fill), ptrString(cfg.Border))

	// Check git requirement before running providers
	if err := c.checkGitRequirement(requested); err != nil {
		return err
	}

	slog.Info("Calculating metrics")

	metricProg := buildMetricProgress(flags)

	if err := provider.Run(root, requested, metricProg); err != nil {
		return eris.Wrap(err, "failed to load metrics")
	}

	if err := c.filterBinaryFiles(cfg, root); err != nil {
		return err
	}

	if flags.ExportData != "" {
		if err := export.Export(root, requested, flags.ExportData); err != nil {
			return eris.Wrap(err, "failed to export data")
		}
	}

	width := ptrInt(flags.Config.Width, 1920)
	height := ptrInt(flags.Config.Height, 1080)

	return c.renderAndLog(root, cfg, width, height, fillMetric, fillPaletteName)
}

// minReservableSize is the smallest treemap dimension (px) that still
// produces a usable visualization. If reserving legend space would shrink
// either dimension below this, we fall back to overlay behavior.
const minReservableSize = 100

// reserveAndLayout computes the effective layout dimensions after reserving
// space for the legend. Falls back to full canvas if the remaining area
// would be too small for a useful treemap.
func reserveAndLayout(legend *render.LegendInfo, width, height int) (layoutW, layoutH int) {
	wReduce, hReduce := render.ReserveLegendSpace(legend)

	w := width - int(wReduce)
	h := height - int(hReduce)

	if w < minReservableSize || h < minReservableSize {
		return width, height
	}

	return w, h
}

// legendLayoutOffset returns the (dx, dy) offset to apply to treemap rects
// when space has been reserved for the legend.
func legendLayoutOffset(info *render.LegendInfo, wReduce, hReduce float64) (dx, dy float64) {
	if info == nil {
		return 0, 0
	}

	switch info.Position {
	case render.LegendPositionTopCenter:
		return 0, hReduce
	case render.LegendPositionCenterLeft:
		return wReduce, 0
	default:
		return cornerLegendOffset(info, wReduce, hReduce)
	}
}

// cornerLegendOffset returns the offset for corner legend positions,
// where orientation determines the carve-out direction.
func cornerLegendOffset(info *render.LegendInfo, wReduce, hReduce float64) (dx, dy float64) {
	isTop := info.Position == render.LegendPositionTopLeft || info.Position == render.LegendPositionTopRight
	isLeft := info.Position == render.LegendPositionTopLeft || info.Position == render.LegendPositionBottomLeft

	if info.Orientation == render.LegendOrientationVertical {
		if isLeft {
			return wReduce, 0
		}

		return 0, 0
	}

	if isTop {
		return 0, hReduce
	}

	return 0, 0
}

// resolveBorderPaletteName determines the effective border metric name and
// palette, using provider defaults when no explicit palette is configured.
// Shared by legend building and colour application to ensure consistency.
func resolveBorderPaletteName(cfg *config.Treemap) (metric.Name, palette.PaletteName) {
	border := ptrString(cfg.Border)
	if border == "" {
		return "", ""
	}

	borderMetric := metric.Name(border)

	if bp := ptrString(cfg.BorderPalette); bp != "" {
		return borderMetric, palette.PaletteName(bp)
	}

	if p, ok := provider.Get(borderMetric); ok {
		return borderMetric, p.DefaultPalette()
	}

	return borderMetric, palette.Neutral
}

func (c *TreemapCmd) renderAndLog(
	root *model.Directory,
	cfg *config.Treemap,
	width, height int,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
) error {
	size := metric.Name(ptrString(cfg.Size))
	files, dirs := countAll(root)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	// Build legend info before layout so we can reserve space for it.
	legendPos, legendOrient := resolveLegendOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	borderName, borderPaletteName := resolveBorderPaletteName(cfg)
	legend := buildLegendInfo(
		legendPos, legendOrient, fillMetric, fillPaletteName,
		borderName, borderPaletteName, size, root,
	)

	layoutW, layoutH := reserveAndLayout(legend, width, height)

	rects := treemap.Layout(root, layoutW, layoutH, size)

	applyFillColours(&rects, root, fillMetric, fillPaletteName)
	c.applyBorderColours(&rects, root, cfg)

	if layoutW < width || layoutH < height {
		wReduce, hReduce := render.ReserveLegendSpace(legend)
		dx, dy := legendLayoutOffset(legend, wReduce, hReduce)
		treemap.OffsetRects(&rects, dx, dy)
	}

	slog.Debug("rendering", "width", width, "height", height, "output", c.Output)

	if err := render.Render(rects, width, height, c.Output, legend); err != nil {
		return eris.Wrap(err, "render failed")
	}

	slog.Info("Rendered treemap",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", width,
		"height", height,
		"size_metric", string(size),
		"fill_metric", string(fillMetric),
		"fill_palette", string(fillPaletteName),
		"border_metric", string(borderName),
		"border_palette", string(borderPaletteName),
	)

	return nil
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
	for _, name := range requested {
		if git.IsGitMetric(name) {
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

	size := string(c.Size)
	if size != "" {
		cfg.Treemap.Size = &size
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

	if c.Legend != "" {
		cfg.Treemap.Legend = &c.Legend
	}

	if c.LegendOrientation != "" {
		cfg.Treemap.LegendOrientation = &c.LegendOrientation
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

//nolint:dupl // mirrors RadialCmd.validatePaths by design
func (c *TreemapCmd) validatePaths() error {
	if _, err := render.FormatFromPath(c.Output); err != nil {
		return &outputPathError{msg: err.Error()}
	}

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

func (*TreemapCmd) resolveFillMetric(cfg *config.Treemap) metric.Name {
	if fill := ptrString(cfg.Fill); fill != "" {
		return metric.Name(fill)
	}

	return metric.Name(ptrString(cfg.Size))
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

func (*TreemapCmd) filterBinaryFiles(cfg *config.Treemap, root *model.Directory) error {
	if metric.Name(ptrString(cfg.Size)) != filesystem.FileLines {
		return nil
	}

	beforeCount, _ := countAll(root)
	filtered := scan.FilterBinaryFiles(root)
	afterCount, _ := countAll(filtered)
	excluded := beforeCount - afterCount
	slog.Debug("binary file filter", "excluded", excluded, "remaining", afterCount)

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
			applyNumericFillColours(rects, root, fillMetric, buckets, fillPalette)
		}
	} else {
		types := collectDistinctTypes(root, fillMetric)
		mapper := palette.NewCategoricalMapper(types, fillPalette)
		applyCategoricalFillColours(rects, root, fillMetric, mapper)
	}
}

//nolint:dupl // structurally identical to RadialCmd.applyBorderColours by design
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
			applyNumericBorderColours(rects, root, borderMetric, buckets, borderPalette)
		}
	} else {
		types := collectDistinctTypes(root, borderMetric)
		mapper := palette.NewCategoricalMapper(types, borderPalette)
		applyCategoricalBorderColours(rects, root, borderMetric, mapper)
	}

	return borderMetric, borderPaletteName
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
	p palette.ColourPalette,
) {
	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := &rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			applyNumericFillColours(child, node.Dirs[dirIdx], m, buckets, p)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			val := extractNumeric(node.Files[fileIdx], m)
			idx := buckets.BucketIndex(val)
			child.FillColour = palette.MapNumericToColour(idx, buckets.NumBuckets(), p)
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
	p palette.ColourPalette,
) {
	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := &rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			applyNumericBorderColours(child, node.Dirs[dirIdx], m, buckets, p)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			val := extractNumeric(node.Files[fileIdx], m)
			idx := buckets.BucketIndex(val)
			col := palette.MapNumericToColour(idx, buckets.NumBuckets(), p)
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
