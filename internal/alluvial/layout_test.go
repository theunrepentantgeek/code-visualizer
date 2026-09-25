package alluvial_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/alluvial"
)

func TestLayoutData_ScalesBandsInProportionToTheirMetricValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := alluvial.LayoutData(alluvial.Data{
		Columns: []alluvial.Column{
			{Reference: "before", Values: []alluvial.Value{{Path: "api", Width: 10}, {Path: "docs", Width: 30}}},
			{Reference: "after", Values: []alluvial.Value{{Path: "api", Width: 20}, {Path: "docs", Width: 20}}},
		},
		Transitions: []alluvial.Transition{
			{FromReference: "before", ToReference: "after", Path: "api", FromWidth: 10, ToWidth: 20},
			{FromReference: "before", ToReference: "after", Path: "docs", FromWidth: 30, ToWidth: 20},
		},
		FillValues: map[string]float64{"api": 7, "docs": 9},
	}, 200, 100)

	g.Expect(layout.Columns).To(HaveLen(2))
	g.Expect(layout.Columns[0].Bands).To(HaveLen(2))
	g.Expect(layout.Columns[0].Bands[0].Path).To(Equal("api"))
	g.Expect(layout.Columns[0].Bands[1].Path).To(Equal("docs"))
	g.Expect(layout.Columns[0].Bands[0].FillValue).To(Equal(float64(7)))
	g.Expect(layout.Columns[0].Bands[1].Bottom - layout.Columns[0].Bands[1].Top).To(
		BeNumerically("~", 3*(layout.Columns[0].Bands[0].Bottom-layout.Columns[0].Bands[0].Top), 0.001),
	)
	g.Expect(layout.Flows).To(HaveLen(2))
}

func TestLayoutData_UsesDestinationSnapshotFillValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := alluvial.LayoutData(alluvial.Data{
		Columns: []alluvial.Column{
			{Reference: "before", Values: []alluvial.Value{{Path: "api", Width: 10}}},
			{Reference: "after", Values: []alluvial.Value{{Path: "api", Width: 10}}},
		},
		Transitions: []alluvial.Transition{
			{FromReference: "before", ToReference: "after", Path: "api", FromWidth: 10, ToWidth: 10},
		},
		FillValuesByReference: map[string]map[string]float64{
			"before": {},
			"after":  {"api": 2},
		},
	}, 200, 100)

	g.Expect(layout.Columns[0].Bands[0].HasFillValue).To(BeFalse())
	g.Expect(layout.Columns[0].Bands[0].FillValue).To(Equal(float64(2)))
	g.Expect(layout.Columns[1].Bands[0].FillValue).To(Equal(float64(2)))
	g.Expect(layout.Columns[1].Bands[0].HasFillValue).To(BeTrue())
	g.Expect(layout.Flows[0].FillValue).To(Equal(float64(2)))
	g.Expect(layout.Flows[0].HasFillValue).To(BeTrue())
}

func TestLayoutData_UsesSharedScaleAcrossColumns(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := alluvial.LayoutData(alluvial.Data{
		Columns: []alluvial.Column{
			{Reference: "smaller", Values: []alluvial.Value{{Path: "api", Width: 10}}},
			{Reference: "larger", Values: []alluvial.Value{{Path: "api", Width: 10}, {Path: "docs", Width: 30}}},
		},
	}, 200, 100)

	smaller := layout.Columns[0].Bands[0]
	larger := layout.Columns[1].Bands[0]

	g.Expect(smaller.Bottom - smaller.Top).To(
		BeNumerically("~", larger.Bottom-larger.Top, 0.001),
	)

	placeholder := layout.Columns[0].Bands[1]
	g.Expect(placeholder.Top - smaller.Bottom).To(BeNumerically("==", 6))
}

func TestLayoutData_TapersIntroducedAndRemovedPaths(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := alluvial.LayoutData(alluvial.Data{
		Columns: []alluvial.Column{
			{
				Reference: "before",
				Values:    []alluvial.Value{{Path: "continuing", Width: 10}, {Path: "removed", Width: 5}},
			},
			{
				Reference: "after",
				Values:    []alluvial.Value{{Path: "continuing", Width: 20}, {Path: "introduced", Width: 5}},
			},
		},
		Transitions: []alluvial.Transition{
			{FromReference: "before", ToReference: "after", Path: "continuing", FromWidth: 10, ToWidth: 20},
			{FromReference: "before", ToReference: "after", Path: "introduced", ToWidth: 5},
			{FromReference: "before", ToReference: "after", Path: "removed", FromWidth: 5},
		},
	}, 200, 100)

	introduced := flowByPath(layout.Flows, "introduced")
	removed := flowByPath(layout.Flows, "removed")
	introducedOrigin := bandByPath(layout.Columns[0].Bands, "introduced")

	g.Expect(introducedOrigin.Top).To(Equal(introducedOrigin.Bottom))
	g.Expect(introduced.FromTop).To(Equal(introducedOrigin.Top))
	g.Expect(introduced.FromTop).To(BeNumerically("==", introduced.FromBottom))
	g.Expect(introduced.ToBottom).To(BeNumerically(">", introduced.ToTop))
	g.Expect(removed.FromBottom).To(BeNumerically(">", removed.FromTop))
	g.Expect(removed.ToTop).To(BeNumerically("==", removed.ToBottom))
}

func TestLayoutData_SpacesZeroWidthPlaceholderFromAdjacentBands(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := alluvial.LayoutData(alluvial.Data{
		Columns: []alluvial.Column{
			{
				Reference: "before",
				Values: []alluvial.Value{
					{Path: "alpha", Width: 10},
					{Path: "gamma", Width: 10},
				},
			},
			{
				Reference: "after",
				Values: []alluvial.Value{
					{Path: "alpha", Width: 10},
					{Path: "beta", Width: 10},
					{Path: "gamma", Width: 10},
				},
			},
		},
	}, 400, 300)

	alpha := bandByPath(layout.Columns[0].Bands, "alpha")
	beta := bandByPath(layout.Columns[0].Bands, "beta")
	gamma := bandByPath(layout.Columns[0].Bands, "gamma")

	g.Expect(beta.Top - alpha.Bottom).To(BeNumerically("==", 6))
	g.Expect(gamma.Top - beta.Bottom).To(BeNumerically("==", 6))
}

func TestLayoutData_SkipsZeroWidthValuesAndTransitions(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := alluvial.LayoutData(alluvial.Data{
		Columns: []alluvial.Column{
			{Reference: "before", Values: []alluvial.Value{{Path: "empty", Width: 0}}},
			{Reference: "after", Values: []alluvial.Value{{Path: "empty", Width: 0}}},
		},
		Transitions: []alluvial.Transition{
			{FromReference: "before", ToReference: "after", Path: "empty"},
		},
	}, 200, 100)

	g.Expect(layout.Columns[0].Bands).To(BeEmpty())
	g.Expect(layout.Columns[1].Bands).To(BeEmpty())
	g.Expect(layout.Flows).To(BeEmpty())
}

func flowByPath(flows []alluvial.Flow, path string) alluvial.Flow {
	for _, flow := range flows {
		if flow.Path == path {
			return flow
		}
	}

	return alluvial.Flow{}
}

func bandByPath(bands []alluvial.Band, path string) alluvial.Band {
	for _, band := range bands {
		if band.Path == path {
			return band
		}
	}

	return alluvial.Band{}
}
