package config

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestTitle_ShowTitle_FalseByDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ti := &Title{}
	g.Expect(ti.ShowTitle()).To(BeFalse())
}

func TestTitle_ShowTitle_FalseWhenHidden(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	hidden := true
	text := "My Repo"
	ti := &Title{
		Hidden: &hidden,
		Text:   &text,
	}

	g.Expect(ti.ShowTitle()).To(BeFalse())
}

func TestTitle_ShowTitle_FalseWhenTextEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	empty := ""
	ti := &Title{Text: &empty}
	g.Expect(ti.ShowTitle()).To(BeFalse())
}

func TestTitle_ShowTitle_TrueWhenTextSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text := "My Repository"
	ti := &Title{Text: &text}
	g.Expect(ti.ShowTitle()).To(BeTrue())
}

func TestTitle_ShowTitle_NilTitle_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ti *Title
	g.Expect(ti.ShowTitle()).To(BeFalse())
}

func TestConfig_OverrideTitleText_SetsText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := New()
	cfg.OverrideTitleText("My Project")

	g.Expect(cfg.Title).NotTo(BeNil())
	g.Expect(*cfg.Title.Text).To(Equal("My Project"))
}

func TestConfig_OverrideTitleText_Empty_LeavesNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := New()
	cfg.OverrideTitleText("")

	g.Expect(cfg.Title).To(BeNil())
}
