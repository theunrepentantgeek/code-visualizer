package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
)

func TestTreemapCmd_ValidateConfig_UsesBaseRegistry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Treemap.Size = new("file-size")

	cmd := &TreemapCmd{}
	err := cmd.validateConfig(cfg.Treemap)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestScatterCmd_ValidateConfig_UsesBaseRegistryForAxesAndSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Scatter.XAxis = new("file-type")
	cfg.Scatter.YAxis = new("file-lines")
	cfg.Scatter.Size = new("file-size")

	cmd := &ScatterCmd{}
	err := cmd.validateConfig(cfg.Scatter)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestScatterCmd_ValidateConfig_AggregationSizeMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Scatter.XAxis = new("file-size")
	cfg.Scatter.YAxis = new("comment-ratio")
	cfg.Scatter.Size = new("declarations.count")

	cmd := &ScatterCmd{}
	err := cmd.validateConfig(cfg.Scatter)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestScatterCmd_ValidateConfig_FilteredAggregationSizeMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Scatter.XAxis = new("file-size")
	cfg.Scatter.YAxis = new("comment-ratio")
	cfg.Scatter.Size = new("public.declarations.count")

	cmd := &ScatterCmd{}
	err := cmd.validateConfig(cfg.Scatter)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestScatterCmd_ValidateConfig_AggregationAxisMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Scatter.XAxis = new("declarations.count")
	cfg.Scatter.YAxis = new("comment-ratio")
	cfg.Scatter.Size = new("file-size")

	cmd := &ScatterCmd{}
	err := cmd.validateConfig(cfg.Scatter)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestTreemapCmd_ValidateConfig_AggregationSizeMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Treemap.Size = new("declarations.count")

	cmd := &TreemapCmd{}
	err := cmd.validateConfig(cfg.Treemap)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestValidateNumericMetric_RejectsClassificationAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// file-type.mode resolves to a classification, which is not numeric.
	err := validateNumericMetric("size", "file-type.mode")
	g.Expect(err).To(MatchError(ContainSubstring("size metric must be numeric")))
}
