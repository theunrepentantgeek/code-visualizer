package alluvial_test

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/alluvial"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

const (
	widthMetric        = metric.Name("test-width.sum")
	measureWidthMetric = metric.Name("test-measure-width.sum")
	fillMetric         = metric.Name("test-fill.sum")
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

func TestBuildData_IncludesTargetDirectFilesBesideChildDirectories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{
			Reference:   "before",
			Root:        testRoot(testDirectory("internal/child", 10)),
			DirectFiles: testDirectory("internal", 4),
		},
		{
			Reference:   "after",
			Root:        testRoot(testDirectory("internal/child", 12)),
			DirectFiles: testDirectory("internal", 7),
		},
	}, alluvial.Options{Metric: widthMetric})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.Columns).To(Equal([]alluvial.Column{
		{Reference: "before", Values: []alluvial.Value{
			{Path: "internal", Width: 4},
			{Path: "internal/child", Width: 10},
		}},
		{Reference: "after", Values: []alluvial.Value{
			{Path: "internal", Width: 7},
			{Path: "internal/child", Width: 12},
		}},
	}))
}

func TestBuildData_RendersLeafTargetAsDirectFilesBand(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{
			Reference:   "before",
			Root:        testRoot(),
			DirectFiles: testDirectory("internal/alluvial", 9),
		},
		{
			Reference:   "after",
			Root:        testRoot(),
			DirectFiles: testDirectory("internal/alluvial", 11),
		},
	}, alluvial.Options{Metric: widthMetric})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.Columns).To(Equal([]alluvial.Column{
		{Reference: "before", Values: []alluvial.Value{{Path: "internal/alluvial", Width: 9}}},
		{Reference: "after", Values: []alluvial.Value{{Path: "internal/alluvial", Width: 11}}},
	}))
}

func TestBuildData_RejectsSnapshotsWithoutAnyBands(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot()},
		{Reference: "after", Root: testRoot()},
	}, alluvial.Options{Metric: widthMetric})

	g.Expect(err).To(MatchError("alluvial target contains no files in any reference"))
}

func TestBuildData_HidesOnlyPathsConstantAcrossEveryReference(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot(
			testDirectory("alpha", 10),
			testDirectory("beta", 5),
			testDirectory("removed", 2),
		)},
		{Reference: "middle", Root: testRoot(
			testDirectory("alpha", 10),
			testDirectory("beta", 7),
			testDirectory("introduced", 3),
		)},
		{Reference: "after", Root: testRoot(
			testDirectory("alpha", 10),
			testDirectory("beta", 5),
			testDirectory("introduced", 3),
		)},
	}, alluvial.Options{
		Metric:        widthMetric,
		ConstantBands: config.ConstantBandsHide,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.Columns).To(Equal([]alluvial.Column{
		{Reference: "before", Values: []alluvial.Value{
			{Path: "beta", Width: 5},
			{Path: "removed", Width: 2},
		}},
		{Reference: "middle", Values: []alluvial.Value{
			{Path: "beta", Width: 7},
			{Path: "introduced", Width: 3},
		}},
		{Reference: "after", Values: []alluvial.Value{
			{Path: "beta", Width: 5},
			{Path: "introduced", Width: 3},
		}},
	}))
	g.Expect(data.FillValues).To(Equal(map[string]float64{
		"beta": 5, "introduced": 3, "removed": 0,
	}))
	g.Expect(data.MutedPaths).To(BeEmpty())
}

func TestBuildData_UsesExactWidthsWhenClassifyingConstantPaths(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot(testDirectoryWithMeasure("api", 10))},
		{Reference: "after", Root: testRoot(testDirectoryWithMeasure("api", math.Nextafter(10, 11)))},
	}, alluvial.Options{
		Metric:        measureWidthMetric,
		ConstantBands: config.ConstantBandsHide,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.Columns[0].Values).To(HaveLen(1))
	g.Expect(data.Columns[1].Values).To(HaveLen(1))
}

func TestBuildData_DoesNotClassifyMissingZeroWidthPathAsConstant(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot()},
		{Reference: "after", Root: testRoot(testDirectory("empty", 0))},
	}, alluvial.Options{
		Metric:        widthMetric,
		ConstantBands: config.ConstantBandsHide,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.Columns[1].Values).To(Equal([]alluvial.Value{{Path: "empty", Width: 0}}))
}

func TestBuildData_MutesConstantPathsWithoutChangingGeometry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot(testDirectory("api", 10), testDirectory("docs", 4))},
		{Reference: "after", Root: testRoot(testDirectory("api", 10), testDirectory("docs", 6))},
	}, alluvial.Options{
		Metric:        widthMetric,
		ConstantBands: config.ConstantBandsMute,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.Columns[0].Values).To(Equal([]alluvial.Value{
		{Path: "api", Width: 10},
		{Path: "docs", Width: 4},
	}))
	g.Expect(data.Transitions).To(HaveLen(2))
	g.Expect(data.MutedPaths).To(Equal(map[string]struct{}{"api": {}}))
	g.Expect(data.FillValues).To(Equal(map[string]float64{"docs": 6}))
}

func TestBuildData_MergesAdjacentConstantRunsSeparatedByChangedPaths(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot(
			testDirectory("alpha", 1),
			testDirectory("bravo", 2),
			testDirectory("charlie", 3),
			testDirectory("delta", 5),
			testDirectory("echo", 6),
		)},
		{Reference: "after", Root: testRoot(
			testDirectory("alpha", 1),
			testDirectory("bravo", 2),
			testDirectory("charlie", 4),
			testDirectory("delta", 5),
			testDirectory("echo", 6),
		)},
	}, alluvial.Options{
		Metric:        widthMetric,
		ConstantBands: config.ConstantBandsMerge,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.Columns).To(HaveLen(2))
	g.Expect(data.Columns[0].Values).To(HaveLen(3))
	firstRun := data.Columns[0].Values[0]
	secondRun := data.Columns[0].Values[2]

	g.Expect(firstRun.Path).NotTo(BeEmpty())
	g.Expect(firstRun.Path).NotTo(Equal(secondRun.Path))
	g.Expect(firstRun.Width).To(Equal(float64(3)))
	g.Expect(data.Columns[0].Values[1]).To(Equal(alluvial.Value{Path: "charlie", Width: 3}))
	g.Expect(secondRun.Width).To(Equal(float64(11)))
	g.Expect(data.Columns[1].Values).To(Equal([]alluvial.Value{
		{Path: firstRun.Path, Width: 3},
		{Path: "charlie", Width: 4},
		{Path: secondRun.Path, Width: 11},
	}))
	g.Expect(data.MutedPaths).To(Equal(map[string]struct{}{
		firstRun.Path:  {},
		secondRun.Path: {},
	}))
	g.Expect(data.Transitions).To(HaveLen(3))
	g.Expect(data.FillValues).To(Equal(map[string]float64{"charlie": 4}))
}

func TestBuildData_HideAllowsEveryBandToBeRemoved(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot(testDirectory("api", 10))},
		{Reference: "after", Root: testRoot(testDirectory("api", 10))},
	}, alluvial.Options{
		Metric:        widthMetric,
		ConstantBands: config.ConstantBandsHide,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.Columns).To(Equal([]alluvial.Column{
		{Reference: "before", Values: []alluvial.Value{}},
		{Reference: "after", Values: []alluvial.Value{}},
	}))
	g.Expect(data.Transitions).To(BeEmpty())
	g.Expect(data.FillValues).To(BeEmpty())
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
	}, alluvial.Options{Metric: widthMetric, FillMetric: fillMetric, FillTemporal: metric.TemporalDelta})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.FillValues).To(Equal(map[string]float64{
		"api":       3,
		"docs":      5,
		"legacy":    -4,
		"temporary": 0,
	}))
	g.Expect(data.FillValuesByReference).To(BeNil())
}

func TestBuildData_StepDeltaFillUsesDestinationSnapshot(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot(testDirectoryWithFill("api", 10, 8))},
		{Reference: "middle", Root: testRoot(testDirectoryWithFill("api", 10, 10))},
		{Reference: "after", Root: testRoot(testDirectoryWithFill("api", 10, 7))},
	}, alluvial.Options{
		Metric: widthMetric, FillMetric: fillMetric, FillTemporal: metric.TemporalStepDelta,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.FillValuesByReference["before"]).To(BeEmpty())
	g.Expect(data.FillValuesByReference["middle"]).To(Equal(map[string]float64{"api": 2}))
	g.Expect(data.FillValuesByReference["after"]).To(Equal(map[string]float64{"api": -3}))
}

func TestData_FillValuesForInkIncludesEveryStepDelta(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data, err := alluvial.BuildData([]alluvial.Snapshot{
		{Reference: "before", Root: testRoot(testDirectoryWithFill("api", 10, 0))},
		{Reference: "middle", Root: testRoot(testDirectoryWithFill("api", 10, 100))},
		{Reference: "after", Root: testRoot(testDirectoryWithFill("api", 10, 101))},
	}, alluvial.Options{
		Metric: widthMetric, FillMetric: fillMetric, FillTemporal: metric.TemporalStepDelta,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data.FillValuesForInk()).To(ConsistOf(float64(100), float64(1)))
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

func TestBuildDataStage_UsesConfiguredConstantBandMode(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	metricName := string(widthMetric)
	mode := config.ConstantBandsHide
	state := &alluvial.State{
		Snapshots: []alluvial.Snapshot{
			{Reference: "one", Root: testRoot(testDirectory("api", 12), testDirectory("docs", 4))},
			{Reference: "two", Root: testRoot(testDirectory("api", 12), testDirectory("docs", 6))},
		},
	}

	g.Expect(alluvial.BuildDataStage(state, &config.Alluvial{
		Metric:        &metricName,
		ConstantBands: &mode,
	})).To(Succeed())
	g.Expect(state.Data.Columns[0].Values).To(Equal([]alluvial.Value{
		{Path: "docs", Width: 4},
	}))
	g.Expect(state.Data.Columns[1].Values).To(Equal([]alluvial.Value{
		{Path: "docs", Width: 6},
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
	g.Expect(state.Fill.Encoding.Metric).To(Equal(metric.Name("file-lines.sum")))
	g.Expect(state.Fill.Label).To(Equal(metric.Name("file-lines.delta")))
	g.Expect(state.Fill.Explicit).To(BeTrue())
	g.Expect(state.Fill.Temporal).To(Equal(metric.TemporalDelta))
	g.Expect(common.Requested.Expressions).To(HaveLen(2))
}

func TestResolveMetrics_PaletteOnlyFillUsesWidthMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	metricName := "file-size"
	state := &alluvial.State{}
	common := &stages.CommonState{}

	g.Expect(alluvial.ResolveMetrics(common, state, &config.Alluvial{
		Metric: &metricName,
		Fill:   &config.MetricSpec{Palette: "temperature"},
	})).To(Succeed())
	g.Expect(state.Fill.Encoding.Metric).To(Equal(metric.Name("file-size.sum")))
	g.Expect(state.Fill.Encoding.Palette).To(Equal(palette.Temperature))
	g.Expect(state.Fill.Label).To(Equal(metric.Name("file-size.sum")))
	g.Expect(state.Fill.Explicit).To(BeFalse())
	g.Expect(state.Fill.LabelMetric()).To(BeEmpty())
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

func testDirectoryWithMeasure(repoPath string, width float64) *model.Directory {
	directory := &model.Directory{RepoPath: repoPath}
	directory.SetMeasure(measureWidthMetric, width)

	return directory
}
