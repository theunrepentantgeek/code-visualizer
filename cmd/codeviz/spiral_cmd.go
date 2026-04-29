package main

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/rotisserie/eris"

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
	"github.com/bevan/code-visualizer/internal/spiral"
)

type SpiralCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	Resolution string `short:"r" help:"Time resolution (hourly or daily)." enum:",hourly,daily" default:""`

	Size metric.Name `default:"" enum:",file-size,file-lines,file-age,file-freshness,author-count" help:"Numeric metric for disc size." short:"s"` //nolint:revive,nolintlint // kong struct tags require long lines

	Fill   config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines
	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`     //nolint:revive,nolintlint // kong struct tags require long lines

	Labels string `help:"Label mode: all, laps, or none." enum:",all,laps,none" default:""`

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive // kong struct tags require long lines

	Width  int `default:"1920" help:"Canvas width in pixels."`
	Height int `default:"1920" help:"Canvas height in pixels."`

	Filter []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)."` //nolint:revive // kong struct tags require long lines
}

func (c *SpiralCmd) Validate() error {
	for _, f := range c.Filter {
		if _, err := filter.ParseFilterFlag(f); err != nil {
			return eris.Wrapf(err, "invalid filter %q", f)
		}
	}

	return nil
}

// validateConfig checks the effective configuration after all sources have been
// merged. Called from mergeConfigAndValidate() after TryAutoLoad + applyOverrides.
func (*SpiralCmd) validateConfig(cfg *config.Spiral) error {
	size := ptrString(cfg.Size)
	if size != "" {
		p, ok := provider.Get(metric.Name(size))
		if !ok {
			return eris.Errorf("unknown size metric %q; available metrics: %s", size, formatMetricNames())
		}

		if p.Kind() != metric.Quantity && p.Kind() != metric.Measure {
			return eris.Errorf("size metric must be numeric, got %q (kind: %d)", size, p.Kind())
		}
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
func (c *SpiralCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Spiral)
}

func (c *SpiralCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	cfg := flags.Config.Spiral

	if err := c.validatePaths(); err != nil {
		return err
	}

	if flags.ExportConfig != "" {
		if err := flags.Config.Save(flags.ExportConfig); err != nil {
			return eris.Wrap(err, "failed to save config")
		}
	}

	root, err := c.scanAndRunProviders(flags, cfg)
	if err != nil {
		return err
	}

	buckets, err := c.buildTimeBuckets(root, cfg)
	if err != nil {
		return err
	}

	c.aggregateBucketMetrics(buckets, cfg)

	return c.layoutAndRender(flags, cfg, root, buckets)
}

func (c *SpiralCmd) scanAndRunProviders(flags *Flags, cfg *config.Spiral) (*model.Directory, error) {
	filterRules := c.buildFilterRules(flags.Config)

	slog.Info("Scanning filesystem", "path", c.TargetPath)

	scanProg, stopScanTicker := buildScanProgress(flags)

	root, err := scan.Scan(c.TargetPath, filterRules, scanProg)

	stopScanTicker()

	if err != nil {
		return nil, eris.Wrap(err, "scan failed")
	}

	requested := c.collectSpiralMetrics(cfg)

	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := buildMetricProgress(flags, model.CountFiles(root))

	if err := provider.Run(root, requested, metricProg); err != nil {
		stopMetricTicker()

		return nil, eris.Wrap(err, "failed to load metrics")
	}

	stopMetricTicker()

	if err := c.filterBinaryFiles(cfg, root); err != nil {
		return nil, err
	}

	if err := export.Export(root, requested, flags.ExportData); err != nil {
		return nil, eris.Wrap(err, "failed to export data")
	}

	return root, nil
}

func (c *SpiralCmd) buildTimeBuckets(root *model.Directory, cfg *config.Spiral) ([]spiral.TimeBucket, error) {
	if err := c.checkGitRepo(); err != nil {
		return nil, err
	}

	records, err := spiral.LoadCommitHistory(root)
	if err != nil {
		return nil, eris.Wrap(err, "failed to load commit history")
	}

	if len(records) == 0 {
		return nil, eris.New("no commit history found; spiral requires git commits")
	}

	startTime, endTime := commitTimeRange(records)
	resolution := c.resolveResolution(cfg)

	buckets := spiral.BuildTimeBuckets(resolution, startTime, endTime)
	assignFilesToBuckets(buckets, records)

	return buckets, nil
}

func (c *SpiralCmd) layoutAndRender(
	flags *Flags,
	cfg *config.Spiral,
	root *model.Directory,
	buckets []spiral.TimeBucket,
) error {
	width := ptrInt(flags.Config.Width, 1920)
	height := ptrInt(flags.Config.Height, 1920)
	resolution := c.resolveResolution(cfg)
	labels := c.resolveLabels(cfg)

	nodes := spiral.Layout(buckets, width, height, resolution, labels)
	applySpiralDiscSizes(nodes, buckets)

	fillMetric, fillPaletteName := c.applyFill(nodes, buckets, cfg)
	borderMetric, borderPaletteName := c.applyBorder(nodes, buckets, cfg)

	legendPos, legendOrient := resolveLegendOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	sizeMetric := metric.Name(ptrString(cfg.Size))
	legend := buildLegendInfo(
		legendPos, legendOrient, fillMetric, fillPaletteName,
		borderMetric, borderPaletteName, sizeMetric, root,
	)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	if err := render.RenderSpiral(nodes, width, height, c.Output, legend); err != nil {
		return eris.Wrap(err, "render failed")
	}

	c.logRendered(root, width, height, sizeMetric, fillMetric, fillPaletteName, borderMetric, borderPaletteName)

	return nil
}

func (*SpiralCmd) logRendered(
	root *model.Directory,
	width, height int,
	sizeMetric, fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) {
	files, dirs := countAll(root)

	slog.Info("Rendered spiral",
		"files", files,
		"directories", dirs,
		"width", width,
		"height", height,
		"size_metric", string(sizeMetric),
		"fill_metric", string(fillMetric),
		"fill_palette", string(fillPaletteName),
		"border_metric", string(borderMetric),
		"border_palette", string(borderPaletteName),
	)
}

// applyOverrides writes non-zero CLI flag values on top of the config layer.
// Zero-valued CLI fields are transparent — the config value passes through unchanged.
func (c *SpiralCmd) applyOverrides(cfg *config.Config) {
	if c.Width != 0 {
		cfg.Width = &c.Width
	}

	if c.Height != 0 {
		cfg.Height = &c.Height
	}

	if cfg.Spiral == nil {
		cfg.Spiral = &config.Spiral{}
	}

	if c.Resolution != "" {
		cfg.Spiral.Resolution = &c.Resolution
	}

	size := string(c.Size)
	if size != "" {
		cfg.Spiral.Size = &size
	}

	if !c.Fill.IsZero() {
		cfg.Spiral.Fill = &c.Fill
	}

	if !c.Border.IsZero() {
		cfg.Spiral.Border = &c.Border
	}

	if c.Labels != "" {
		cfg.Spiral.Labels = &c.Labels
	}

	c.applyLegendOverrides(cfg.Spiral)
}

func (c *SpiralCmd) applyLegendOverrides(cfg *config.Spiral) {
	if c.Legend != "" {
		cfg.Legend = &c.Legend
	}

	if c.LegendOrientation != "" {
		cfg.LegendOrientation = &c.LegendOrientation
	}
}

//nolint:dupl // mirrors TreemapCmd.validatePaths by design
func (c *SpiralCmd) validatePaths() error {
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

func (c *SpiralCmd) buildFilterRules(cfg *config.Config) []filter.Rule {
	rules := make([]filter.Rule, 0, len(cfg.FileFilter)+len(c.Filter))
	rules = append(rules, cfg.FileFilter...)

	for _, f := range c.Filter {
		// Already validated in Validate()
		rule, _ := filter.ParseFilterFlag(f)
		rules = append(rules, rule)
	}

	return rules
}

// checkGitRepo verifies the target path is inside a git repository.
// Spiral always requires git for commit history.
func (c *SpiralCmd) checkGitRepo() error {
	absPath, err := filepath.Abs(c.TargetPath)
	if err != nil {
		return eris.Wrap(err, "failed to resolve absolute path")
	}

	isGit, err := scan.IsGitRepo(absPath)
	if err != nil {
		return eris.Wrap(err, "git check failed")
	}

	if !isGit {
		return &gitRequiredError{metric: "spiral", target: c.TargetPath}
	}

	return nil
}

// collectSpiralMetrics gathers all metrics requested by the spiral configuration.
// Unlike other commands, size is optional — when omitted, disc size defaults to commit count.
func (*SpiralCmd) collectSpiralMetrics(cfg *config.Spiral) []metric.Name {
	size := metric.Name(ptrString(cfg.Size))
	if size != "" {
		return collectRequestedMetrics(size, cfg.Fill, cfg.Border)
	}

	seen := map[metric.Name]bool{}

	var names []metric.Name

	for _, spec := range []*config.MetricSpec{cfg.Fill, cfg.Border} {
		if spec != nil && spec.Metric != "" && !seen[spec.Metric] {
			seen[spec.Metric] = true
			names = append(names, spec.Metric)
		}
	}

	return names
}

func (*SpiralCmd) resolveResolution(cfg *config.Spiral) spiral.Resolution {
	if r := ptrString(cfg.Resolution); r == "hourly" {
		return spiral.Hourly
	}

	return spiral.Daily
}

func (*SpiralCmd) resolveLabels(cfg *config.Spiral) spiral.LabelMode {
	if lbl := ptrString(cfg.Labels); lbl != "" {
		return spiral.LabelMode(lbl)
	}

	return spiral.LabelLaps
}

func (*SpiralCmd) resolveFillMetric(cfg *config.Spiral) metric.Name {
	return specMetric(cfg.Fill)
}

func (*SpiralCmd) resolveFillPalette(cfg *config.Spiral, fillMetric metric.Name) palette.PaletteName {
	if fp := specPalette(cfg.Fill); fp != "" {
		return fp
	}

	if p, ok := provider.Get(fillMetric); ok {
		return p.DefaultPalette()
	}

	return palette.Neutral
}

func (*SpiralCmd) filterBinaryFiles(cfg *config.Spiral, root *model.Directory) error {
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

// aggregateBucketMetrics fills in the aggregated metric values for each time bucket.
func (c *SpiralCmd) aggregateBucketMetrics(buckets []spiral.TimeBucket, cfg *config.Spiral) {
	sizeMetric := metric.Name(ptrString(cfg.Size))
	fillMetric := specMetric(cfg.Fill)
	borderMetric := specMetric(cfg.Border)

	for i := range buckets {
		c.aggregateBucket(&buckets[i], sizeMetric, fillMetric, borderMetric)
	}
}

func (*SpiralCmd) aggregateBucket(
	b *spiral.TimeBucket,
	sizeMetric, fillMetric, borderMetric metric.Name,
) {
	if sizeMetric != "" {
		b.SizeValue = sumNumericMetric(b.Files, sizeMetric)
	} else {
		b.SizeValue = float64(len(b.Files))
	}

	aggregateColourMetric(b.Files, fillMetric, &b.FillValue, &b.FillLabel)
	aggregateColourMetric(b.Files, borderMetric, &b.BorderValue, &b.BorderLabel)
}

func aggregateColourMetric(files []*model.File, m metric.Name, numVal *float64, catLabel *string) {
	if m == "" {
		return
	}

	p, ok := provider.Get(m)
	if !ok {
		return
	}

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
		*numVal = sumNumericMetric(files, m)
	} else {
		*catLabel = modeCategory(files, m)
	}
}

func sumNumericMetric(files []*model.File, m metric.Name) float64 {
	var total float64

	for _, f := range files {
		total += extractNumeric(f, m)
	}

	return total
}

// modeCategory returns the most frequent classification value among the given files.
func modeCategory(files []*model.File, m metric.Name) string {
	counts := map[string]int{}

	for _, f := range files {
		if cat, ok := f.Classification(m); ok {
			counts[cat]++
		}
	}

	best := ""
	bestCount := 0

	for cat, count := range counts {
		if count > bestCount {
			best = cat
			bestCount = count
		}
	}

	return best
}

// commitTimeRange returns the earliest and latest timestamps from commit records.
func commitTimeRange(records []spiral.CommitRecord) (time.Time, time.Time) {
	minT := records[0].Timestamp
	maxT := records[0].Timestamp

	for _, r := range records[1:] {
		if r.Timestamp.Before(minT) {
			minT = r.Timestamp
		}

		if r.Timestamp.After(maxT) {
			maxT = r.Timestamp
		}
	}

	return minT, maxT
}

// assignFilesToBuckets places each commit record's file into the appropriate time bucket.
func assignFilesToBuckets(buckets []spiral.TimeBucket, records []spiral.CommitRecord) {
	for _, rec := range records {
		for i := range buckets {
			if !rec.Timestamp.Before(buckets[i].Start) && rec.Timestamp.Before(buckets[i].End) {
				buckets[i].Files = append(buckets[i].Files, rec.File)

				break
			}
		}
	}
}

// applySpiralDiscSizes sets disc radii on nodes proportional to their size values.
func applySpiralDiscSizes(nodes []spiral.SpiralNode, buckets []spiral.TimeBucket) {
	maxSize := 0.0

	for _, b := range buckets {
		if b.SizeValue > maxSize {
			maxSize = b.SizeValue
		}
	}

	if maxSize == 0 {
		return
	}

	for i := range nodes {
		ratio := buckets[i].SizeValue / maxSize
		// sqrt scaling gives area-proportional discs
		nodes[i].DiscRadius *= math.Sqrt(ratio)
	}
}

// applyFill applies fill colours to spiral nodes based on the configured fill metric.
func (c *SpiralCmd) applyFill(
	nodes []spiral.SpiralNode,
	buckets []spiral.TimeBucket,
	cfg *config.Spiral,
) (metric.Name, palette.PaletteName) {
	fillMetric := c.resolveFillMetric(cfg)
	if fillMetric == "" {
		return "", ""
	}

	fillPaletteName := c.resolveFillPalette(cfg, fillMetric)
	fillPalette := palette.GetPalette(fillPaletteName)

	p, ok := provider.Get(fillMetric)
	if !ok {
		return fillMetric, fillPaletteName
	}

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
		applySpiralNumericFill(nodes, buckets, fillPalette)
	} else {
		applySpiralCategoricalFill(nodes, buckets, fillPalette)
	}

	return fillMetric, fillPaletteName
}

func applySpiralNumericFill(
	nodes []spiral.SpiralNode,
	buckets []spiral.TimeBucket,
	p palette.ColourPalette,
) {
	values := make([]float64, len(buckets))
	for i, b := range buckets {
		values[i] = b.FillValue
	}

	bb := metric.ComputeBuckets(values, len(p.Colours))

	for i := range nodes {
		idx := bb.BucketIndex(values[i])
		nodes[i].FillColour = palette.MapNumericToColour(idx, bb.NumBuckets(), p)
	}
}

func applySpiralCategoricalFill(
	nodes []spiral.SpiralNode,
	buckets []spiral.TimeBucket,
	p palette.ColourPalette,
) {
	types := collectBucketCategories(buckets, func(b *spiral.TimeBucket) string { return b.FillLabel })
	mapper := palette.NewCategoricalMapper(types, p)

	for i := range nodes {
		if buckets[i].FillLabel != "" {
			nodes[i].FillColour = mapper.Map(buckets[i].FillLabel)
		}
	}
}

// applyBorder applies border colours to spiral nodes based on the configured border metric.
func (c *SpiralCmd) applyBorder(
	nodes []spiral.SpiralNode,
	buckets []spiral.TimeBucket,
	cfg *config.Spiral,
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
		applySpiralNumericBorder(nodes, buckets, borderPalette)
	} else {
		applySpiralCategoricalBorder(nodes, buckets, borderPalette)
	}

	return border, borderPaletteName
}

func applySpiralNumericBorder(
	nodes []spiral.SpiralNode,
	buckets []spiral.TimeBucket,
	p palette.ColourPalette,
) {
	values := make([]float64, len(buckets))
	for i, b := range buckets {
		values[i] = b.BorderValue
	}

	bb := metric.ComputeBuckets(values, len(p.Colours))

	for i := range nodes {
		idx := bb.BucketIndex(values[i])
		c := palette.MapNumericToColour(idx, bb.NumBuckets(), p)
		nodes[i].BorderColour = &c
	}
}

func applySpiralCategoricalBorder(
	nodes []spiral.SpiralNode,
	buckets []spiral.TimeBucket,
	p palette.ColourPalette,
) {
	types := collectBucketCategories(buckets, func(b *spiral.TimeBucket) string { return b.BorderLabel })
	mapper := palette.NewCategoricalMapper(types, p)

	for i := range nodes {
		if buckets[i].BorderLabel != "" {
			c := mapper.Map(buckets[i].BorderLabel)
			nodes[i].BorderColour = &c
		}
	}
}

// collectBucketCategories gathers distinct non-empty category labels from time buckets.
func collectBucketCategories(
	buckets []spiral.TimeBucket,
	labelFn func(*spiral.TimeBucket) string,
) []string {
	seen := map[string]bool{}

	for i := range buckets {
		lbl := labelFn(&buckets[i])
		if lbl != "" {
			seen[lbl] = true
		}
	}

	types := make([]string, 0, len(seen))
	for t := range seen {
		types = append(types, t)
	}

	return types
}
