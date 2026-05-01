package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/bubbletree"
	"github.com/bevan/code-visualizer/internal/config"
	"github.com/bevan/code-visualizer/internal/export"
	"github.com/bevan/code-visualizer/internal/filter"
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/render"
	"github.com/bevan/code-visualizer/internal/scan"
)

type BubbletreeCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	//nolint:revive // kong struct tags require long lines
	Size metric.Name `default:"" help:"Metric for circle size; run 'codeviz help-metrics' for available metrics." short:"s"`

	Fill config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"`

	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`

	Labels string `enum:",all,folders,none" default:"" help:"Labels to display: all, folders, or none."`

	//nolint:revive // kong struct tags require long lines
	Legend string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""`

	//nolint:revive // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1080" help:"Image height in pixels."`

	Filter []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)."` //nolint:revive // kong struct tags require long lines
}

func (c *BubbletreeCmd) Validate() error {
	for _, f := range c.Filter {
		if _, err := filter.ParseFilterFlag(f); err != nil {
			return eris.Wrapf(err, "invalid filter %q", f)
		}
	}

	return nil
}

// validateConfig checks the effective configuration after all sources have been
// merged. Called from mergeConfigAndValidate() after TryAutoLoad + applyOverrides.
func (*BubbletreeCmd) validateConfig(cfg *config.Bubbletree) error {
	size := ptrString(cfg.Size)

	p, ok := provider.Get(metric.Name(size))
	if !ok {
		return eris.Errorf("unknown size metric %q; available metrics: %s", size, formatMetricNames())
	}

	if p.Kind() != metric.Quantity && p.Kind() != metric.Measure {
		return eris.Errorf("size metric must be numeric, got %q (kind: %d)", size, p.Kind())
	}

	if err := cfg.Fill.Validate("fill"); err != nil {
		return eris.Wrap(err, "invalid fill spec")
	}

	if err := cfg.Border.Validate("border"); err != nil {
		return eris.Wrap(err, "invalid border spec")
	}

	return nil
}

// mergeConfigAndValidate loads the config file, merges CLI overrides on top,
// and validates the effective configuration. Called at the start of Run().
func (c *BubbletreeCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Bubbletree)
}

//nolint:dupl // parallel Run methods on different config types share the same workflow
func (c *BubbletreeCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	cfg := flags.Config.Bubbletree
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

	scanProg, stopScanTicker := buildScanProgress(flags)

	root, err := scan.Scan(c.TargetPath, filterRules, scanProg)

	stopScanTicker()

	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	requested := collectRequestedMetrics(size, cfg.Fill, cfg.Border)

	if err := c.checkGitRequirement(requested); err != nil {
		return err
	}

	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := buildMetricProgress(flags, model.CountFiles(root))

	if err := provider.Run(root, requested, metricProg); err != nil {
		stopMetricTicker()

		return eris.Wrap(err, "failed to load metrics")
	}

	stopMetricTicker()

	if err := c.filterBinaryFiles(cfg, root); err != nil {
		return err
	}

	if err := export.Export(root, requested, flags.ExportData); err != nil {
		return eris.Wrap(err, "failed to export data")
	}

	width := ptrInt(flags.Config.Width, 1920)
	height := ptrInt(flags.Config.Height, 1080)

	return c.renderAndLog(root, cfg, width, height, fillMetric, fillPaletteName)
}

func (c *BubbletreeCmd) renderAndLog(
	root *model.Directory,
	cfg *config.Bubbletree,
	width, height int,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
) error {
	size := metric.Name(ptrString(cfg.Size))
	files, dirs := countAll(root)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	borderMetric, borderPaletteName, err := c.applyColoursAndRender(
		cfg, root, width, height, fillMetric, fillPaletteName,
	)
	if err != nil {
		return err
	}

	slog.Info("Rendered bubble tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", width,
		"height", height,
		"size_metric", string(size),
		"fill_metric", string(fillMetric),
		"fill_palette", string(fillPaletteName),
		"border_metric", string(borderMetric),
		"border_palette", string(borderPaletteName),
	)

	return nil
}

// applyColoursAndRender lays out, colours, and renders the bubble tree to disk.
func (c *BubbletreeCmd) applyColoursAndRender(
	cfg *config.Bubbletree,
	root *model.Directory,
	width, height int,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
) (metric.Name, palette.PaletteName, error) {
	size := metric.Name(ptrString(cfg.Size))
	labels := c.resolveLabels(cfg)
	nodes := bubbletree.Layout(root, width, height, size, labels)
	applyBubbleFillColoursTop(&nodes, root, fillMetric, fillPaletteName)
	borderMetric, borderPaletteName := c.applyBorderColours(&nodes, root, cfg)

	legendPos, legendOrient := resolveLegendOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	borderName := specMetric(cfg.Border)
	legend := buildLegendInfo(
		legendPos, legendOrient, fillMetric, fillPaletteName,
		borderName, borderPaletteName, size, root,
	)

	slog.Debug("rendering bubble tree", "width", width, "height", height, "output", c.Output)

	if err := render.RenderBubble(&nodes, width, height, c.Output, legend); err != nil {
		return "", "", eris.Wrap(err, "render failed")
	}

	return borderMetric, borderPaletteName, nil
}

// applyOverrides writes non-zero CLI flag values on top of the config layer.
// Zero-valued CLI fields are transparent — the config value passes through unchanged.
func (c *BubbletreeCmd) applyOverrides(cfg *config.Config) {
	if c.Width != 0 {
		cfg.Width = &c.Width
	}

	if c.Height != 0 {
		cfg.Height = &c.Height
	}

	if cfg.Bubbletree == nil {
		cfg.Bubbletree = &config.Bubbletree{}
	}

	size := string(c.Size)
	if size != "" {
		cfg.Bubbletree.Size = &size
	}

	if !c.Fill.IsZero() {
		cfg.Bubbletree.Fill = &c.Fill
	}

	if !c.Border.IsZero() {
		cfg.Bubbletree.Border = &c.Border
	}

	if c.Labels != "" {
		cfg.Bubbletree.Labels = &c.Labels
	}

	c.applyLegendOverrides(cfg.Bubbletree)
}

func (c *BubbletreeCmd) applyLegendOverrides(cfg *config.Bubbletree) {
	if c.Legend != "" {
		cfg.Legend = &c.Legend
	}

	if c.LegendOrientation != "" {
		cfg.LegendOrientation = &c.LegendOrientation
	}
}

//nolint:dupl // mirrors TreemapCmd.validatePaths by design
func (c *BubbletreeCmd) validatePaths() error {
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

func (c *BubbletreeCmd) buildFilterRules(cfg *config.Config) []filter.Rule {
	rules := make([]filter.Rule, 0, len(cfg.FileFilter)+len(c.Filter))
	rules = append(rules, cfg.FileFilter...)

	for _, f := range c.Filter {
		// Already validated in Validate()
		rule, _ := filter.ParseFilterFlag(f)
		rules = append(rules, rule)
	}

	return rules
}

func (c *BubbletreeCmd) checkGitRequirement(requested []metric.Name) error {
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

func (*BubbletreeCmd) resolveFillMetric(cfg *config.Bubbletree) metric.Name {
	if fill := specMetric(cfg.Fill); fill != "" {
		return fill
	}

	return metric.Name(ptrString(cfg.Size))
}

func (*BubbletreeCmd) resolveFillPalette(cfg *config.Bubbletree, fillMetric metric.Name) palette.PaletteName {
	if fp := specPalette(cfg.Fill); fp != "" {
		return fp
	}

	if p, ok := provider.Get(fillMetric); ok {
		return p.DefaultPalette()
	}

	return palette.Neutral
}

func (*BubbletreeCmd) filterBinaryFiles(cfg *config.Bubbletree, root *model.Directory) error {
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

// resolveLabels converts the string labels flag to a bubbletree.LabelMode.
func (*BubbletreeCmd) resolveLabels(cfg *config.Bubbletree) bubbletree.LabelMode {
	if lbl := ptrString(cfg.Labels); lbl != "" {
		return bubbletree.LabelMode(lbl)
	}

	return bubbletree.LabelFoldersOnly
}

func applyBubbleFillColoursTop(
	nodes *bubbletree.BubbleNode,
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
			applyBubbleFillColours(nodes, root, fillMetric, buckets, fillPalette)
		}
	} else {
		types := collectDistinctTypes(root, fillMetric)
		mapper := palette.NewCategoricalMapper(types, fillPalette)
		applyCategoricalBubbleFillColours(nodes, root, fillMetric, mapper)
	}
}

//nolint:dupl // structurally identical to TreemapCmd.applyBorderColours by design
func (*BubbletreeCmd) applyBorderColours(
	nodes *bubbletree.BubbleNode,
	root *model.Directory,
	cfg *config.Bubbletree,
) (metric.Name, palette.PaletteName) {
	border := specMetric(cfg.Border)
	if border == "" {
		return "", ""
	}

	borderPaletteName := specPalette(cfg.Border)
	if borderPaletteName == "" {
		if p, ok := provider.Get(border); ok {
			borderPaletteName = p.DefaultPalette()
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	borderPalette := palette.GetPalette(borderPaletteName)

	p, ok := provider.Get(border)
	if !ok {
		return border, borderPaletteName
	}

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
		values := collectNumericValues(root, border)
		if len(values) > 0 {
			buckets := metric.ComputeBuckets(values, len(borderPalette.Colours))
			applyBubbleBorderColours(nodes, root, border, buckets, borderPalette)
		}
	} else {
		types := collectDistinctTypes(root, border)
		mapper := palette.NewCategoricalMapper(types, borderPalette)
		applyCategoricalBubbleBorderColours(nodes, root, border, mapper)
	}

	return border, borderPaletteName
}

// indexBubbleNodesByPath recursively walks the BubbleNode tree and indexes
// all nodes by their Path, separating directories and files.
func indexBubbleNodesByPath(
	node *bubbletree.BubbleNode,
	dirs map[string]*bubbletree.BubbleNode,
	files map[string]*bubbletree.BubbleNode,
) {
	for i := range node.Children {
		child := &node.Children[i]
		if child.IsDirectory {
			dirs[child.Path] = child
			indexBubbleNodesByPath(child, dirs, files)
		} else {
			files[child.Path] = child
		}
	}
}

// applyBubbleFillColours assigns fill colours to the BubbleNode tree using
// path-based lookup, decoupled from Children ordering.
func applyBubbleFillColours(
	node *bubbletree.BubbleNode,
	root *model.Directory,
	m metric.Name,
	buckets metric.BucketBoundaries,
	p palette.ColourPalette,
) {
	dirs := make(map[string]*bubbletree.BubbleNode)
	files := make(map[string]*bubbletree.BubbleNode)
	indexBubbleNodesByPath(node, dirs, files)

	applyBubbleFillColoursWalk(root, m, buckets, p, dirs, files)
}

func applyBubbleFillColoursWalk(
	dir *model.Directory,
	m metric.Name,
	buckets metric.BucketBoundaries,
	p palette.ColourPalette,
	dirs map[string]*bubbletree.BubbleNode,
	files map[string]*bubbletree.BubbleNode,
) {
	for _, f := range dir.Files {
		if bn, ok := files[f.Path]; ok {
			val := extractNumeric(f, m)
			idx := buckets.BucketIndex(val)
			bn.FillColour = palette.MapNumericToColour(idx, buckets.NumBuckets(), p)
		}
	}

	for _, d := range dir.Dirs {
		if _, ok := dirs[d.Path]; ok {
			applyBubbleFillColoursWalk(d, m, buckets, p, dirs, files)
		}
	}
}

// applyCategoricalBubbleFillColours assigns categorical fill colours to the
// BubbleNode tree using path-based lookup.
func applyCategoricalBubbleFillColours(
	node *bubbletree.BubbleNode,
	root *model.Directory,
	m metric.Name,
	mapper *palette.CategoricalMapper,
) {
	dirs := make(map[string]*bubbletree.BubbleNode)
	files := make(map[string]*bubbletree.BubbleNode)
	indexBubbleNodesByPath(node, dirs, files)

	applyCategoricalBubbleFillColoursWalk(root, m, mapper, dirs, files)
}

func applyCategoricalBubbleFillColoursWalk(
	dir *model.Directory,
	m metric.Name,
	mapper *palette.CategoricalMapper,
	dirs map[string]*bubbletree.BubbleNode,
	files map[string]*bubbletree.BubbleNode,
) {
	for _, f := range dir.Files {
		if bn, ok := files[f.Path]; ok {
			if v, ok := f.Classification(m); ok {
				bn.FillColour = mapper.Map(v)
			}
		}
	}

	for _, d := range dir.Dirs {
		if _, ok := dirs[d.Path]; ok {
			applyCategoricalBubbleFillColoursWalk(d, m, mapper, dirs, files)
		}
	}
}

// applyBubbleBorderColours assigns border colours to the BubbleNode tree using
// path-based lookup.
func applyBubbleBorderColours(
	node *bubbletree.BubbleNode,
	root *model.Directory,
	m metric.Name,
	buckets metric.BucketBoundaries,
	p palette.ColourPalette,
) {
	dirs := make(map[string]*bubbletree.BubbleNode)
	files := make(map[string]*bubbletree.BubbleNode)
	indexBubbleNodesByPath(node, dirs, files)

	applyBubbleBorderColoursWalk(root, m, buckets, p, dirs, files)
}

func applyBubbleBorderColoursWalk(
	dir *model.Directory,
	m metric.Name,
	buckets metric.BucketBoundaries,
	p palette.ColourPalette,
	dirs map[string]*bubbletree.BubbleNode,
	files map[string]*bubbletree.BubbleNode,
) {
	for _, f := range dir.Files {
		if bn, ok := files[f.Path]; ok {
			val := extractNumeric(f, m)
			idx := buckets.BucketIndex(val)
			col := palette.MapNumericToColour(idx, buckets.NumBuckets(), p)
			bn.BorderColour = &col
		}
	}

	for _, d := range dir.Dirs {
		if _, ok := dirs[d.Path]; ok {
			applyBubbleBorderColoursWalk(d, m, buckets, p, dirs, files)
		}
	}
}

// applyCategoricalBubbleBorderColours assigns categorical border colours to the
// BubbleNode tree using path-based lookup.
func applyCategoricalBubbleBorderColours(
	node *bubbletree.BubbleNode,
	root *model.Directory,
	m metric.Name,
	mapper *palette.CategoricalMapper,
) {
	dirs := make(map[string]*bubbletree.BubbleNode)
	files := make(map[string]*bubbletree.BubbleNode)
	indexBubbleNodesByPath(node, dirs, files)

	applyCategoricalBubbleBorderColoursWalk(root, m, mapper, dirs, files)
}

func applyCategoricalBubbleBorderColoursWalk(
	dir *model.Directory,
	m metric.Name,
	mapper *palette.CategoricalMapper,
	dirs map[string]*bubbletree.BubbleNode,
	files map[string]*bubbletree.BubbleNode,
) {
	for _, f := range dir.Files {
		if bn, ok := files[f.Path]; ok {
			if v, ok := f.Classification(m); ok {
				col := mapper.Map(v)
				bn.BorderColour = &col
			}
		}
	}

	for _, d := range dir.Dirs {
		if _, ok := dirs[d.Path]; ok {
			applyCategoricalBubbleBorderColoursWalk(d, m, mapper, dirs, files)
		}
	}
}
