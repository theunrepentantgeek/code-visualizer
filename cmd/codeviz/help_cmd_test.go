package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/alecthomas/kong"

	"github.com/bevan/code-visualizer/internal/metric"
)

func TestKindLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(kindLabel(metric.Quantity)).To(Equal("quantity"))
	g.Expect(kindLabel(metric.Measure)).To(Equal("measure"))
	g.Expect(kindLabel(metric.Classification)).To(Equal("category"))
	g.Expect(kindLabel(metric.Kind(99))).To(Equal("unknown"))
}

func TestGitMetricNames_ContainsKnownGitMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(gitMetricNames).To(HaveKey(metric.Name("file-age")))
	g.Expect(gitMetricNames).To(HaveKey(metric.Name("file-freshness")))
	g.Expect(gitMetricNames).To(HaveKey(metric.Name("author-count")))
}

func TestGitMetricNames_DoesNotContainFilesystemMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(gitMetricNames).NotTo(HaveKey(metric.Name("file-size")))
	g.Expect(gitMetricNames).NotTo(HaveKey(metric.Name("file-lines")))
	g.Expect(gitMetricNames).NotTo(HaveKey(metric.Name("file-type")))
}

func TestHelpMetrics_RunSucceeds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// filesystem providers are pre-registered by TestMain; test with those.
	cmd := HelpMetricsCmd{}
	err := cmd.Run(&Flags{})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestHelpPalettes_RunSucceeds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := HelpPalettesCmd{}
	err := cmd.Run(&Flags{})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestKongParsesHelpMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(&cli, kong.Name("codeviz"), kong.Exit(func(int) {}))
	g.Expect(err).NotTo(HaveOccurred())
	_, err = parser.Parse([]string{"help", "metrics"})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestKongParsesHelpPalettes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(&cli, kong.Name("codeviz"), kong.Exit(func(int) {}))
	g.Expect(err).NotTo(HaveOccurred())
	_, err = parser.Parse([]string{"help", "palettes"})
	g.Expect(err).NotTo(HaveOccurred())
}
