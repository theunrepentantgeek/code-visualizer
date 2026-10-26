// Package donuttree implements deterministic geometry for donut tree visualizations.
package donuttree

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

const donutRingWidthRatio = 0.9

// Layout builds a concentric directory-sector layout. The root is represented
// by the central anchor, while its directories begin in the first annular ring.
func Layout(root *model.Directory, canvasSize int, sizeMetric metric.Name) LayoutResult {
	center := float64(canvasSize) / 2
	result := LayoutResult{
		Center: geometry.NewPoint(center, center),
	}

	if root == nil {
		return result
	}

	result.RootName = root.Name
	maxDepth := maxDirectoryDepth(root)

	if maxDepth == 0 {
		result.AnchorRadius = math.Max(center, 0)

		return result
	}

	ringWidth := math.Max(center, 0) / float64(maxDepth+1)
	result.AnchorRadius = ringWidth
	contributions := directoryContributions(root, sizeMetric)
	result.Children = layoutChildren(
		root.Dirs, 1, -math.Pi/2, 2*math.Pi, ringWidth, sizeMetric, contributions,
	)

	return result
}

func layoutChildren(
	dirs []*model.Directory,
	depth int,
	startAngle, sweepAngle, ringWidth float64,
	sizeMetric metric.Name,
	contributions map[*model.Directory]metricContribution,
) []DonutNode {
	if len(dirs) == 0 {
		return nil
	}

	values, sumPositive := allocationValues(dirs, sizeMetric)
	minimum := math.Min(math.Pi/180, sweepAngle/float64(len(dirs)))
	remaining := sweepAngle - minimum*float64(len(dirs))
	innerRadius := float64(depth) * ringWidth
	outerRadius := innerRadius + ringWidth*donutRingWidthRatio
	children := make([]DonutNode, 0, len(dirs))
	childStart := startAngle

	for i, dir := range dirs {
		childSweep := childSweep(values[i], sumPositive, minimum, remaining, sweepAngle, len(dirs))

		child := DonutNode{
			Directory:   dir,
			Depth:       depth,
			StartAngle:  childStart,
			SweepAngle:  childSweep,
			InnerRadius: innerRadius,
			OuterRadius: outerRadius,
		}
		if dir != nil {
			nestedStart := child.StartAngle
			nestedSweep := child.SweepAngle * nestedSectorShare(dir, sizeMetric, contributions)
			nestedStart += (child.SweepAngle - nestedSweep) / 2

			child.Children = layoutChildren(
				dir.Dirs,
				depth+1,
				nestedStart,
				nestedSweep,
				ringWidth,
				sizeMetric,
				contributions,
			)
		}

		children = append(children, child)
		childStart = child.EndAngle()
	}

	return children
}

type metricContribution struct {
	value                float64
	count                int
	distinctValues       map[string]struct{}
	directDistinctValues map[string]struct{}
	distinctMode         bool
}

type metricContributionConfig struct {
	expression   metric.MetricExpression
	descriptor   provider.BaseMetricDescriptor
	key          metric.Name
	aggregation  metric.AggregationName
	useCounts    bool
	distinctMode bool
}

func (c *metricContribution) add(other metricContribution) {
	c.value += other.value

	c.count += other.count
	if len(other.distinctValues) > 0 {
		if c.distinctValues == nil {
			c.distinctValues = make(map[string]struct{}, len(other.distinctValues))
		}

		for value := range other.distinctValues {
			c.distinctValues[value] = struct{}{}
		}
	}
}

func (c *metricContribution) share(nested metricContribution) float64 {
	if c.distinctMode {
		if len(c.distinctValues) == 0 {
			return 0
		}

		nestedCount := 0

		for value := range nested.distinctValues {
			if _, direct := c.directDistinctValues[value]; !direct {
				nestedCount++
			}
		}

		return math.Min(float64(nestedCount)/float64(len(c.distinctValues)), 1)
	}

	if c.value > 0 {
		return math.Min(nested.value/c.value, 1)
	}

	if c.count > 0 {
		return math.Min(float64(nested.count)/float64(c.count), 1)
	}

	return 0
}

func nestedSectorShare(
	dir *model.Directory,
	sizeMetric metric.Name,
	contributions map[*model.Directory]metricContribution,
) float64 {
	nested := metricContribution{}
	for _, child := range dir.Dirs {
		nested.add(contributions[child])
	}

	contribution := contributions[dir]
	if contribution.count > 0 || contribution.distinctMode {
		return contribution.share(nested)
	}

	return legacyNestedShare(dir, sizeMetric)
}

// directoryContributions derives additive weights from positive source values,
// observation counts for .count, or unique source values for .distinct.
func directoryContributions(root *model.Directory, sizeMetric metric.Name) map[*model.Directory]metricContribution {
	contributions := make(map[*model.Directory]metricContribution)

	expr, err := metric.ParseExpression(string(sizeMetric))
	if err != nil {
		return contributions
	}

	desc, ok := provider.GetBase(expr.Base)
	if !ok {
		return contributions
	}

	config := metricContributionConfig{
		expression:  expr,
		descriptor:  desc,
		aggregation: expr.Aggregation,
	}
	if config.aggregation.IsZero() {
		config.aggregation, _ = desc.Kind.DefaultAggregation()
	}

	config.useCounts = config.aggregation == metric.AggCount
	config.distinctMode = desc.Kind == metric.Classification && config.aggregation == metric.AggDistinct
	config.key = metricSourceKey(expr, desc)

	collectDirectoryContributions(root, config, contributions)

	return contributions
}

func metricSourceKey(expr metric.MetricExpression, desc provider.BaseMetricDescriptor) metric.Name {
	if desc.Level != metric.LevelFile || expr.Filter.IsZero() {
		return expr.Base
	}

	return metric.MetricExpression{Filter: expr.Filter, Base: expr.Base}.ResultName()
}

func collectDirectoryContributions(
	dir *model.Directory,
	config metricContributionConfig,
	contributions map[*model.Directory]metricContribution,
) metricContribution {
	if dir == nil {
		return metricContribution{}
	}

	result := directMetricContribution(dir, config)
	for _, child := range dir.Dirs {
		result.add(collectDirectoryContributions(child, config, contributions))
	}

	contributions[dir] = result

	return result
}

func directMetricContribution(dir *model.Directory, config metricContributionConfig) metricContribution {
	result := metricContribution{distinctMode: config.distinctMode}

	switch config.descriptor.Level {
	case metric.LevelFile:
		addFileContributions(&result, dir, config)
	case metric.LevelDeclaration:
		addDeclarationContributions(&result, dir, config)
	case metric.LevelCommit:
		addCommitContributions(&result, dir, config)
	default:
		return result
	}

	return result
}

func addFileContributions(result *metricContribution, dir *model.Directory, config metricContributionConfig) {
	for _, file := range dir.Files {
		if file != nil {
			addMetricSource(result, file, file, config)
		}
	}
}

func addDeclarationContributions(result *metricContribution, dir *model.Directory, config metricContributionConfig) {
	for _, file := range dir.Files {
		if file == nil {
			continue
		}

		for _, declaration := range file.Declarations {
			if declaration != nil {
				addMetricSource(result, declaration, declaration, config)
			}
		}
	}
}

func addCommitContributions(result *metricContribution, dir *model.Directory, config metricContributionConfig) {
	for _, file := range dir.Files {
		if file == nil {
			continue
		}

		for _, commit := range file.Commits {
			if commit != nil {
				addMetricSource(result, commit, commit, config)
			}
		}
	}
}

func addMetricSource(
	contribution *metricContribution,
	source metricValueSource,
	node any,
	config metricContributionConfig,
) {
	if !metricSourcePassesFilter(node, config) {
		return
	}

	if addDeclarationCount(contribution, node, config) {
		return
	}

	if config.descriptor.Kind == metric.Classification {
		addClassificationContribution(contribution, source, config)

		return
	}

	addNumericContribution(contribution, source, config)
}

func metricSourcePassesFilter(node any, config metricContributionConfig) bool {
	filter := config.expression.Filter

	return filter.IsZero() || config.descriptor.PassesFilter(filter, node)
}

func addDeclarationCount(contribution *metricContribution, node any, config metricContributionConfig) bool {
	if config.descriptor.Level != metric.LevelDeclaration {
		return false
	}

	declaration, ok := node.(*model.Declaration)
	if !ok || !config.descriptor.MatchesDeclKind(declaration.Kind) {
		return true
	}

	if config.aggregation != metric.AggCount {
		return false
	}

	contribution.count++
	contribution.value++

	return true
}

func addNumericContribution(
	contribution *metricContribution,
	source metricValueSource,
	config metricContributionConfig,
) {
	value, ok := sourceValue(source, config.key, config.descriptor.Kind)
	if !ok {
		return
	}

	contribution.count++
	if config.useCounts {
		contribution.value++
	} else if value > 0 {
		contribution.value += value
	}
}

func addClassificationContribution(
	contribution *metricContribution,
	source metricValueSource,
	config metricContributionConfig,
) {
	value, ok := source.Classification(config.key)
	if !ok {
		return
	}

	contribution.count++
	if !config.distinctMode {
		contribution.value++

		return
	}

	if contribution.distinctValues == nil {
		contribution.distinctValues = make(map[string]struct{})
	}

	if contribution.directDistinctValues == nil {
		contribution.directDistinctValues = make(map[string]struct{})
	}

	contribution.distinctValues[value] = struct{}{}
	contribution.directDistinctValues[value] = struct{}{}
}

type metricValueSource interface {
	Quantity(name metric.Name) (int64, bool)
	Measure(name metric.Name) (float64, bool)
	Classification(name metric.Name) (string, bool)
}

func sourceValue(source metricValueSource, name metric.Name, kind metric.Kind) (float64, bool) {
	switch kind {
	case metric.Quantity:
		value, ok := source.Quantity(name)

		return float64(value), ok
	case metric.Measure:
		return source.Measure(name)
	default:
		return 0, false
	}
}

func legacyNestedShare(dir *model.Directory, sizeMetric metric.Name) float64 {
	parentValue := directoryMetricValue(dir, sizeMetric)

	_, nestedValue := allocationValues(dir.Dirs, sizeMetric)
	if parentValue > 0 && nestedValue > 0 && nestedValue < parentValue {
		return nestedValue / parentValue
	}

	return 1
}

func allocationValues(dirs []*model.Directory, sizeMetric metric.Name) ([]float64, float64) {
	values := make([]float64, len(dirs))

	var sumPositive float64

	for i, dir := range dirs {
		value := directoryMetricValue(dir, sizeMetric)

		if value > 0 {
			values[i] = value
			sumPositive += value
		}
	}

	return values, sumPositive
}

func childSweep(
	value, sumPositive, minimum, remaining, parentSweep float64,
	childCount int,
) float64 {
	if sumPositive == 0 {
		return parentSweep / float64(childCount)
	}

	if value > 0 {
		return minimum + remaining*value/sumPositive
	}

	return minimum
}

// directoryMetricValue returns a directory's allocation metric, preferring
// quantities to measures. Missing and non-positive values do not allocate area.
func directoryMetricValue(dir *model.Directory, sizeMetric metric.Name) float64 {
	if dir == nil {
		return 0
	}

	if quantity, ok := dir.Quantity(sizeMetric); ok {
		return float64(quantity)
	}

	if measure, ok := dir.Measure(sizeMetric); ok {
		return measure
	}

	return 0
}

func maxDirectoryDepth(dir *model.Directory) int {
	if dir == nil {
		return 0
	}

	maxDepth := 0

	for _, child := range dir.Dirs {
		depth := 1 + maxDirectoryDepth(child)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth
}
