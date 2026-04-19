package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/bubbletree"
	"github.com/bevan/code-visualizer/internal/config"
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

	Size metric.Name `enum:"file-size,file-lines,file-age,file-freshness,author-count" help:"Metric for circle size." required:"true" short:"s"` //nolint:revive // kong struct tags require long lines

	Fill          string `default:"" enum:",file-size,file-lines,file-type,file-age,file-freshness,author-count" help:"Metric for fill colour." optional:"" short:"f"`   //nolint:revive // kong struct tags require long lines
	FillPalette   string `default:"" enum:",categorization,temperature,good-bad,neutral,foliage" help:"Palette for fill colour." name:"fill-palette" optional:""`        //nolint:revive // kong struct tags require long lines
	Border        string `default:"" enum:",file-size,file-lines,file-type,file-age,file-freshness,author-count" help:"Metric for border colour." optional:"" short:"b"` //nolint:revive // kong struct tags require long lines
	BorderPalette string `default:"" enum:",categorization,temperature,good-bad,neutral,foliage" help:"Palette for border colour." name:"border-palette" optional:""`    //nolint:revive // kong struct tags require long lines

	Labels string `enum:",all,folders,none" default:"" help:"Labels to display: all, folders, or none."`

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1080" help:"Image height in pixels."`

	Filter []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)."` //nolint:revive // kong struct tags require long lines
}

func (c *BubbletreeCmd) Validate() error {
	p, ok := provider.Get(c.Size)
	if !ok {
		return eris.Errorf("unknown size metric %q; available metrics: %s", c.Size, formatMetricNames())
	}

	if p.Kind() != metric.Quantity && p.Kind() != metric.Measure {
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

func (c *BubbletreeCmd) Run(flags *Flags) error {
	c.applyOverrides(flags.Config)

	cfg := flags.Config.Bubbletree

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

	requested := collectRequestedMetrics(c.Size, ptrString(cfg.Fill), ptrString(cfg.Border))

	err = c.checkGitRequirement(requested)
	if err != nil {
		return err
	}

	slog.Info("Calculating metrics")

	metricProg := buildMetricProgress(flags)

	err = provider.Run(root, requested, metricProg)
	if err != nil {
		return eris.Wrap(err, "failed to load metrics")
	}

	err = c.filterBinaryFiles(cfg, root)
	if err != nil {
		return err
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
	files, dirs := countAll(root)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	borderMetric, borderPaletteName, err := c.applyColoursAndRender(cfg, root, width, height, fillMetric, fillPaletteName)
	if err != nil {
		return err
	}

	slog.Info("Rendered bubble tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", width,
		"height", height,
		"size_metric", string(c.Size),
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
	labels := c.resolveLabels(cfg)
	nodes := bubbletree.Layout(root, width, height, c.Size, labels)
	applyBubbleFillColoursTop(&nodes, root, fillMetric, fillPaletteName)
	borderMetric, borderPaletteName := c.applyBorderColours(&nodes, root, cfg)

	slog.Debug("rendering bubble tree", "width", width, "height", height, "output", c.Output)

	if err := render.RenderBubble(&nodes, width, height, c.Output); err != nil {
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

	if c.Fill != "" {
		cfg.Bubbletree.Fill = &c.Fill
	}

	if c.FillPalette != "" {
		cfg.Bubbletree.FillPalette = &c.FillPalette
	}

	if c.Border != "" {
		cfg.Bubbletree.Border = &c.Border
	}

	if c.BorderPalette != "" {
		cfg.Bubbletree.BorderPalette = &c.BorderPalette
	}

	if c.Labels != "" {
		cfg.Bubbletree.Labels = &c.Labels
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

func (c *BubbletreeCmd) resolveFillMetric(cfg *config.Bubbletree) metric.Name {
	if fill := ptrString(cfg.Fill); fill != "" {
		return metric.Name(fill)
	}

	return c.Size
}

func (*BubbletreeCmd) resolveFillPalette(cfg *config.Bubbletree, fillMetric metric.Name) palette.PaletteName {
	if fp := ptrString(cfg.FillPalette); fp != "" {
		return palette.PaletteName(fp)
	}

	if p, ok := provider.Get(fillMetric); ok {
		return p.DefaultPalette()
	}

	return palette.Neutral
}

func (c *BubbletreeCmd) filterBinaryFiles(_ *config.Bubbletree, root *model.Directory) error {
	if c.Size != filesystem.FileLines {
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
			numBuckets := len(buckets.Boundaries) + 1
			applyBubbleFillColours(nodes, root, fillMetric, buckets, numBuckets, fillPalette)
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
			applyBubbleBorderColours(nodes, root, borderMetric, buckets, numBuckets, borderPalette)
		}
	} else {
		types := collectDistinctTypes(root, borderMetric)
		mapper := palette.NewCategoricalMapper(types, borderPalette)
		applyCategoricalBubbleBorderColours(nodes, root, borderMetric, mapper)
	}

	return borderMetric, borderPaletteName
}

// applyBubbleFillColours assigns fill colours to the BubbleNode tree.
// INVARIANT: node.Children must be ordered files-first, then subdirectories —
// matching the order produced by Layout. fileIdx and dirIdx rely on this
// ordering to correctly pair nodes with their model counterparts.
func applyBubbleFillColours(
	node *bubbletree.BubbleNode,
	dir *model.Directory,
	m metric.Name,
	buckets metric.BucketBoundaries,
	numBuckets int,
	p palette.ColourPalette,
) {
	fileIdx := 0
	dirIdx := 0

	for i := range node.Children {
		child := &node.Children[i]
		if child.IsDirectory && dirIdx < len(dir.Dirs) {
			applyBubbleFillColours(child, dir.Dirs[dirIdx], m, buckets, numBuckets, p)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(dir.Files) {
			val := extractNumeric(dir.Files[fileIdx], m)
			idx := buckets.BucketIndex(val)
			child.FillColour = palette.MapNumericToColour(idx, numBuckets, p)
			fileIdx++
		}
	}
}

// applyCategoricalBubbleFillColours assigns categorical fill colours to the BubbleNode tree.
// INVARIANT: node.Children must be ordered files-first, then subdirectories —
// matching the order produced by Layout. fileIdx and dirIdx rely on this
// ordering to correctly pair nodes with their model counterparts.
func applyCategoricalBubbleFillColours(
	node *bubbletree.BubbleNode,
	dir *model.Directory,
	m metric.Name,
	mapper *palette.CategoricalMapper,
) {
	fileIdx := 0
	dirIdx := 0

	for i := range node.Children {
		child := &node.Children[i]
		if child.IsDirectory && dirIdx < len(dir.Dirs) {
			applyCategoricalBubbleFillColours(child, dir.Dirs[dirIdx], m, mapper)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(dir.Files) {
			if v, ok := dir.Files[fileIdx].Classification(m); ok {
				child.FillColour = mapper.Map(v)
			}

			fileIdx++
		}
	}
}

// applyBubbleBorderColours assigns border colours to the BubbleNode tree.
// INVARIANT: node.Children must be ordered files-first, then subdirectories —
// matching the order produced by Layout. fileIdx and dirIdx rely on this
// ordering to correctly pair nodes with their model counterparts.
func applyBubbleBorderColours(
	node *bubbletree.BubbleNode,
	dir *model.Directory,
	m metric.Name,
	buckets metric.BucketBoundaries,
	numBuckets int,
	p palette.ColourPalette,
) {
	fileIdx := 0
	dirIdx := 0

	for i := range node.Children {
		child := &node.Children[i]
		if child.IsDirectory && dirIdx < len(dir.Dirs) {
			applyBubbleBorderColours(child, dir.Dirs[dirIdx], m, buckets, numBuckets, p)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(dir.Files) {
			val := extractNumeric(dir.Files[fileIdx], m)
			idx := buckets.BucketIndex(val)
			col := palette.MapNumericToColour(idx, numBuckets, p)
			child.BorderColour = &col
			fileIdx++
		}
	}
}

// applyCategoricalBubbleBorderColours assigns categorical border colours to the BubbleNode tree.
// INVARIANT: node.Children must be ordered files-first, then subdirectories —
// matching the order produced by Layout. fileIdx and dirIdx rely on this
// ordering to correctly pair nodes with their model counterparts.
func applyCategoricalBubbleBorderColours(
	node *bubbletree.BubbleNode,
	dir *model.Directory,
	m metric.Name,
	mapper *palette.CategoricalMapper,
) {
	fileIdx := 0
	dirIdx := 0

	for i := range node.Children {
		child := &node.Children[i]
		if child.IsDirectory && dirIdx < len(dir.Dirs) {
			applyCategoricalBubbleBorderColours(child, dir.Dirs[dirIdx], m, mapper)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(dir.Files) {
			if v, ok := dir.Files[fileIdx].Classification(m); ok {
				col := mapper.Map(v)
				child.BorderColour = &col
			}

			fileIdx++
		}
	}
}
