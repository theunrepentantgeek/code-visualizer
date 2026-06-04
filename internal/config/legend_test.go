package config

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestLegend_PositionStr_NilReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	var l *Legend
	g.Expect(l.PositionStr()).To(Equal(""))
}

func TestLegend_PositionStr_ReturnsPosition(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	pos := "top-right"
	l := &Legend{Position: &pos}
	g.Expect(l.PositionStr()).To(Equal("top-right"))
}

func TestLegend_PositionStr_VisibleFalseReturnsNone(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	hidden := false
	pos := "top-right"
	l := &Legend{Visible: &hidden, Position: &pos}
	g.Expect(l.PositionStr()).To(Equal("none"))
}

func TestLegend_PositionStr_VisibleTrueReturnsPosition(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	visible := true
	pos := "bottom-center"
	l := &Legend{Visible: &visible, Position: &pos}
	g.Expect(l.PositionStr()).To(Equal("bottom-center"))
}

func TestLegend_OrientationStr_NilReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	var l *Legend
	g.Expect(l.OrientationStr()).To(Equal(""))
}

func TestLegend_OrientationStr_ReturnsOrientation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	orient := "horizontal"
	l := &Legend{Orientation: &orient}
	g.Expect(l.OrientationStr()).To(Equal("horizontal"))
}
