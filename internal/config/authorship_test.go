package config

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestDefaultAuthorshipConfig_AllFieldsNonNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := DefaultAuthorshipConfig()

	g.Expect(cfg).NotTo(BeNil())
	g.Expect(cfg.ActivityWindowDays).NotTo(BeNil())
	g.Expect(cfg.RecentWindowDays).NotTo(BeNil())
	g.Expect(cfg.EarlyWindowFraction).NotTo(BeNil())
	g.Expect(cfg.SignificantShareThreshold).NotTo(BeNil())
	g.Expect(cfg.BusFactorThreshold).NotTo(BeNil())
	g.Expect(cfg.IdentityTopK).NotTo(BeNil())
	g.Expect(cfg.HonorMailmap).NotTo(BeNil())
}

func TestDefaultAuthorshipConfig_SpecValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := DefaultAuthorshipConfig()

	g.Expect(*cfg.ActivityWindowDays).To(Equal(180))
	g.Expect(*cfg.RecentWindowDays).To(Equal(180))
	g.Expect(*cfg.EarlyWindowFraction).To(BeNumerically("~", 0.25, 1e-9))
	g.Expect(*cfg.SignificantShareThreshold).To(BeNumerically("~", 0.10, 1e-9))
	g.Expect(*cfg.BusFactorThreshold).To(BeNumerically("~", 0.50, 1e-9))
	g.Expect(*cfg.IdentityTopK).To(Equal(11))
	g.Expect(*cfg.HonorMailmap).To(BeTrue())
}

func TestNew_AuthorshipDefaultsSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := New()

	g.Expect(cfg.Authorship).NotTo(BeNil())
	g.Expect(cfg.Authorship.ActivityWindowDays).NotTo(BeNil())
	g.Expect(*cfg.Authorship.ActivityWindowDays).To(Equal(180))
	g.Expect(*cfg.Authorship.IdentityTopK).To(Equal(11))
}

func TestDefaultAuthorshipConfig_IndependentInstances(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := DefaultAuthorshipConfig()
	b := DefaultAuthorshipConfig()

	// Mutating one must not affect the other (values are copies, not shared pointers).
	*a.ActivityWindowDays = 999

	g.Expect(*b.ActivityWindowDays).To(Equal(180))
}
