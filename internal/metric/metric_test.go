package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestKindConstants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Quantity).To(Equal(Kind(0)))
	g.Expect(Measure).To(Equal(Kind(1)))
	g.Expect(Classification).To(Equal(Kind(2)))
}

func TestMetricName_IsValid(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	valid := []MetricName{FileSize, FileLines, FileType, FileAge, FileFreshness, AuthorCount}
	for _, m := range valid {
		g.Expect(m.IsValid()).To(BeTrue(), "expected %q to be valid", m)
	}

	g.Expect(MetricName("unknown").IsValid()).To(BeFalse())
	g.Expect(MetricName("").IsValid()).To(BeFalse())
	g.Expect(MetricName("FILE-SIZE").IsValid()).To(BeFalse())
}

func TestMetricName_IsNumeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	numeric := []MetricName{FileSize, FileLines, FileAge, FileFreshness, AuthorCount}
	for _, m := range numeric {
		g.Expect(m.IsNumeric()).To(BeTrue(), "expected %q to be numeric", m)
	}

	g.Expect(FileType.IsNumeric()).To(BeFalse())
}

func TestMetricName_IsGitRequired(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	gitRequired := []MetricName{FileAge, FileFreshness, AuthorCount}
	for _, m := range gitRequired {
		g.Expect(m.IsGitRequired()).To(BeTrue(), "expected %q to be git-required", m)
	}

	nonGit := []MetricName{FileSize, FileLines, FileType}
	for _, m := range nonGit {
		g.Expect(m.IsGitRequired()).To(BeFalse(), "expected %q to NOT be git-required", m)
	}
}
