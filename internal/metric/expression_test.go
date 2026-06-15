package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseExpression_BareMetricName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr, err := ParseExpression("file-size")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(expr.Filter).To(Equal(FilterName("")))
	g.Expect(expr.Base).To(Equal(Name("file-size")))
	g.Expect(expr.Aggregation).To(Equal(AggregationName("")))
}

func TestParseExpression_MetricWithAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr, err := ParseExpression("file-size.sum")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(expr.Filter).To(Equal(FilterName("")))
	g.Expect(expr.Base).To(Equal(Name("file-size")))
	g.Expect(expr.Aggregation).To(Equal(AggregationName("sum")))
}

func TestParseExpression_TwoSegmentsPreferAggregationWhenKnown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr, err := ParseExpression("public.count")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(expr.Filter).To(Equal(FilterName("")))
	g.Expect(expr.Base).To(Equal(Name("public")))
	g.Expect(expr.Aggregation).To(Equal(AggregationName("count")))
}

func TestParseExpression_FullExpression(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr, err := ParseExpression("public.types.count")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(expr.Filter).To(Equal(FilterName("public")))
	g.Expect(expr.Base).To(Equal(Name("types")))
	g.Expect(expr.Aggregation).To(Equal(AggregationName("count")))
}

func TestParseExpression_TwoSegmentsNonAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "public.types" — last segment "types" is not a known aggregation
	// so this is filter="public", base="types", aggregation=""
	expr, err := ParseExpression("public.types")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(expr.Filter).To(Equal(FilterName("public")))
	g.Expect(expr.Base).To(Equal(Name("types")))
	g.Expect(expr.Aggregation).To(Equal(AggregationName("")))
}

func TestParseExpression_EmptyStringReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseExpression("")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("empty"))
}

func TestParseExpression_TooManySegmentsReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseExpression("a.b.c.d")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("too many segments"))
}

func TestParseExpression_InvalidCharactersReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseExpression("file size")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid"))
}

func TestParseExpression_ThreeSegmentsUnknownAggregationReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseExpression("public.types.total")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("aggregation"))
}

func TestParseExpression_EmptySegmentsReturnError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	for _, input := range []string{".a", "a.", "a..b"} {
		_, err := ParseExpression(input)
		g.Expect(err).To(HaveOccurred(), input)
		g.Expect(err.Error()).To(ContainSubstring("invalid"), input)
	}
}

func TestMetricExpression_String_BareMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr := MetricExpression{Base: "file-size"}
	g.Expect(expr.String()).To(Equal("file-size"))
}

func TestMetricExpression_String_WithAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr := MetricExpression{Base: "file-size", Aggregation: "sum"}
	g.Expect(expr.String()).To(Equal("file-size.sum"))
}

func TestMetricExpression_String_Full(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr := MetricExpression{Filter: "public", Base: "types", Aggregation: "count"}
	g.Expect(expr.String()).To(Equal("public.types.count"))
}

func TestMetricExpression_ResultName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr := MetricExpression{Filter: "public", Base: "types", Aggregation: "count"}
	g.Expect(expr.ResultName()).To(Equal(Name("public.types.count")))
}
