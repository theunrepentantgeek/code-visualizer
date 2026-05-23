package golang

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestIsGoMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(IsGoMetric(TypeCount)).To(BeTrue())
	g.Expect(IsGoMetric(PublicTypeCount)).To(BeTrue())
	g.Expect(IsGoMetric(CommentRatio)).To(BeTrue())
	g.Expect(IsGoMetric(CyclomaticComplexityMean)).To(BeTrue())
	g.Expect(IsGoMetric(FunctionLengthMax)).To(BeTrue())
	g.Expect(IsGoMetric(InternalImportCount)).To(BeTrue())
	g.Expect(IsGoMetric("file-size")).To(BeFalse())
	g.Expect(IsGoMetric("file-lines")).To(BeFalse())
	g.Expect(IsGoMetric("unknown-metric")).To(BeFalse())
}

func TestAllGoMetricCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(allMetrics).To(HaveLen(35))
}
