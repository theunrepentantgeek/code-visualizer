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
