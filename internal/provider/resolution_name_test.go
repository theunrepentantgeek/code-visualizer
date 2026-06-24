package provider_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
)

func TestResolveName_BareFileMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	filesystem.Register()

	resolved, err := provider.ResolveName("file-size", metric.LevelFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.ResultKind).To(Equal(metric.Quantity))
}

func TestResolveName_AggregationExpression(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	golang.Register()

	resolved, err := provider.ResolveName("declarations.count", metric.LevelFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.ResultKind).To(Equal(metric.Quantity))
}

func TestResolveName_UnknownMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := provider.ResolveName("not-a-real-metric", metric.LevelFile)
	g.Expect(err).To(MatchError(ContainSubstring("unknown base metric")))
}
