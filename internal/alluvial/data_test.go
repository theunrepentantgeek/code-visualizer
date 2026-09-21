package alluvial_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/alluvial"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

const (
	widthMetric = metric.Name("test-width.sum")
	fillMetric  = metric.Name("test-fill.sum")
)

func TestMain(m *testing.M) {
	filesystem.Register()

	m.Run()
}

func TestBuildData_PreservesReferenceOrderAndAlignsSnapshotWidths(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "release-3", Root: testRoot(
			testDirectory("api", 10),
			testDirectory("docs", 5),
		)},
		{Reference: "release-1", Root: testRoot(
			testDirectory("api", 13),
			testDirectory("legacy", 7),
		)},
		{Reference: "release-2", Root: testRoot(
			testDirectory("api", 11),
			testDirectory("docs", 9),
		)},
	}, alluvial.Options{Metric: widthMetric})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.Columns).To(Equal([]alluvial.Column{
		{
			Reference: "release-3",
			Values: []alluvial.Value{
				{Path: "api", Width: 10},
				{Path: "docs", Width: 5},
			},
		},
		{
			Reference: "release-1",
			Values: []alluvial.Value{
				{Path: "api", Width: 13},
				{Path: "legacy", Width: 7},
			},
		},
		{
			Reference: "release-2",
			Values: []alluvial.Value{
				{Path: "api", Width: 11},
				{Path: "docs", Width: 9},
			},
		},
	}))
	g.Expect(data.Transitions).To(Equal([]alluvial.Transition{
		{FromReference: "release-3", ToReference: "release-1", Path: "api", FromWidth: 10, ToWidth: 13},
		{FromReference: "release-3", ToReference: "release-1", Path: "docs", FromWidth: 5, ToWidth: 0},
		{FromReference: "release-3", ToReference: "release-1", Path: "legacy", FromWidth: 0, ToWidth: 7},
		{FromReference: "release-1", ToReference: "release-2", Path: "api", FromWidth: 13, ToWidth: 11},
		{FromReference: "release-1", ToReference: "release-2", Path: "docs", FromWidth: 0, ToWidth: 9},
		{FromReference: "release-1", ToReference: "release-2", Path: "legacy", FromWidth: 7, ToWidth: 0},
	}))
}

func TestBuildData_ExpandsOnlySelectedDirectoryDirectChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot(
			testDirectory(
				"api", 100,
				testDirectory("api/internal", 30, testDirectory("api/internal/private", 10)),
				testDirectory("api/public", 70),
			),
			testDirectory("docs", 4),
		)},
		{Reference: "after", Root: testRoot(
			testDirectory(
				"api", 120,
				testDirectory("api/internal", 40, testDirectory("api/internal/private", 15)),
				testDirectory("api/public", 80),
			),
			// This tree models the result of the existing file filters: no
			// excluded directory is synthesized by the alluvial data stage.
			testDirectory("docs", 6),
		)},
	}, alluvial.Options{Metric: widthMetric, Expand: []string{"api"}})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.Columns[0].Values).To(Equal([]alluvial.Value{
		{Path: "api/internal", Width: 30},
		{Path: "api/public", Width: 70},
		{Path: "docs", Width: 4},
	}))
	g.Expect(data.Columns[1].Values).To(Equal([]alluvial.Value{
		{Path: "api/internal", Width: 40},
		{Path: "api/public", Width: 80},
		{Path: "docs", Width: 6},
	}))
}

func TestBuildData_UsesLastSnapshotFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot(testDirectoryWithFill("api", 10, 2))},
		{Reference: "after", Root: testRoot(
			testDirectoryWithFill("api", 12, 7),
			testDirectoryWithFill("docs", 5, 3),
		)},
	}, alluvial.Options{Metric: widthMetric, FillMetric: fillMetric})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.FillValues).To(Equal(map[string]float64{"api": 7, "docs": 3}))
}

func TestBuildData_DeltaFillUsesChangeFromFirstToLastSnapshot(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot(
			testDirectoryWithFill("api", 10, 8),
			testDirectoryWithFill("legacy", 4, 4),
		)},
		{Reference: "middle", Root: testRoot(testDirectoryWithFill("temporary", 6, 100))},
		{Reference: "after", Root: testRoot(
			testDirectoryWithFill("api", 12, 11),
			testDirectoryWithFill("docs", 5, 5),
		)},
	}, alluvial.Options{Metric: widthMetric, FillMetric: fillMetric, FillDelta: true})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.FillValues).To(Equal(map[string]float64{
		"api":       3,
		"docs":      5,
		"legacy":    -4,
		"temporary": 0,
	}))
}

func TestBuildDataStage_UsesConfiguredMetricAndExpansion(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	metricName := string(widthMetric)
	state := &alluvial.State{
		Snapshots: []alluvial.Snapshot{
			{Reference: "one", Root: testRoot(testDirectory("api", 12, testDirectory("api/public", 12)))},
			{Reference: "two", Root: testRoot(testDirectory("api", 18, testDirectory("api/public", 18)))},
		},
	}

	g.Expect(alluvial.BuildDataStage(state, &config.Alluvial{
		Metric: &metricName,
		Expand: []string{"api"},
	})).To(Succeed())
	g.Expect(state.Data.Columns[0].Values).To(Equal([]alluvial.Value{
		{Path: "api/public", Width: 12},
	}))
}

func TestResolveMetrics_AggregatesBareMetricForDirectoryWidths(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	metricName := "file-lines"
	common := &stages.CommonState{}

	g.Expect(alluvial.ResolveMetrics(common, &alluvial.State{}, &config.Alluvial{
		Metric: &metricName,
	})).To(Succeed())
	g.Expect(common.Requested.Expressions).To(HaveLen(1))
	g.Expect(common.Requested.Expressions[0].ResultName).To(Equal(metric.Name("file-lines.sum")))
}

func TestResolveMetrics_ResolvesDeltaFillWithoutRequestingModifier(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	metricName := "file-size"
	state := &alluvial.State{}
	common := &stages.CommonState{}

	g.Expect(alluvial.ResolveMetrics(common, state, &config.Alluvial{
		Metric: &metricName,
		Fill:   &config.MetricSpec{Metric: "file-lines.delta", Palette: "temperature"},
	})).To(Succeed())
	g.Expect(state.Fill.Metric).To(Equal(metric.Name("file-lines.sum")))
	g.Expect(state.FillLabel).To(Equal(metric.Name("file-lines.delta")))
	g.Expect(state.FillSpecified).To(BeTrue())
	g.Expect(state.FillDelta).To(BeTrue())
	g.Expect(common.Requested.Expressions).To(HaveLen(2))
}

func testRoot(dirs ...*model.Directory) *model.Directory {
	return &model.Directory{Dirs: dirs}
}

func testDirectory(repoPath string, width int64, children ...*model.Directory) *model.Directory {
	directory := &model.Directory{RepoPath: repoPath, Dirs: children}
	directory.SetQuantity(widthMetric, width)

	return directory
}

func testDirectoryWithFill(repoPath string, width, fill int64) *model.Directory {
	directory := testDirectory(repoPath, width)
	directory.SetQuantity(fillMetric, fill)

	return directory
}
