package config

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestFooter_ShowFooter_FalseByDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &Footer{}
	g.Expect(f.ShowFooter()).To(BeFalse())
}

func TestFooter_ShowFooter_TrueWhenSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &Footer{
		Hidden: new(true),
	}

	g.Expect(f.ShowFooter()).To(BeFalse())
}

func TestFooter_ShowFooter_NilFooter_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var f *Footer
	g.Expect(f.ShowFooter()).To(BeFalse())
}

func TestConfig_OverrideFooterText_SetsText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := New()
	cfg.OverrideFooterText("custom text")

	g.Expect(cfg.Footer).NotTo(BeNil())
	g.Expect(*cfg.Footer.Text).To(Equal("custom text"))
}

func TestConfig_OverrideHideFooter_SetsHidden(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := New()
	cfg.OverrideHideFooter(true)

	g.Expect(cfg.Footer).NotTo(BeNil())
	g.Expect(cfg.Footer.ShowFooter()).To(BeFalse())
}
