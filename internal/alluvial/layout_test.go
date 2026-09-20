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
	}, 200, 100)

	g.Expect(layout.Columns).To(HaveLen(2))
	g.Expect(layout.Columns[0].Bands).To(HaveLen(2))
	g.Expect(layout.Columns[0].Bands[0].Path).To(Equal("api"))
	g.Expect(layout.Columns[0].Bands[1].Path).To(Equal("docs"))
	g.Expect(layout.Columns[0].Bands[1].Bottom - layout.Columns[0].Bands[1].Top).To(
		BeNumerically("~", 3*(layout.Columns[0].Bands[0].Bottom-layout.Columns[0].Bands[0].Top), 0.001),
	)
	g.Expect(layout.Flows).To(HaveLen(2))
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

	g.Expect(introduced.FromTop).To(BeNumerically("==", introduced.FromBottom))
	g.Expect(introduced.ToBottom).To(BeNumerically(">", introduced.ToTop))
	g.Expect(removed.FromBottom).To(BeNumerically(">", removed.FromTop))
	g.Expect(removed.ToTop).To(BeNumerically("==", removed.ToBottom))
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
