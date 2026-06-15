package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	gitprovider "github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	golangprovider "github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
)

func registerBaseMetricsOnly(t *testing.T) {
	t.Helper()

	provider.ResetBaseRegistryForTesting()
	provider.ResetRegistryForTesting()

	filesystem.RegisterBase()
	gitprovider.RegisterBase()
	golangprovider.RegisterBase()

	t.Cleanup(func() {
		provider.ResetBaseRegistryForTesting()
		provider.ResetRegistryForTesting()

		filesystem.Register()
		gitprovider.Register()
		golangprovider.Register()
	})
}

func TestTreemapCmd_ValidateConfig_UsesBaseRegistry(t *testing.T) {
	g := NewGomegaWithT(t)
	registerBaseMetricsOnly(t)

	cfg := config.New()
	cfg.Treemap.Size = new("file-size")

	cmd := &TreemapCmd{}
	err := cmd.validateConfig(cfg.Treemap)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestScatterCmd_ValidateConfig_UsesBaseRegistryForAxesAndSize(t *testing.T) {
	g := NewGomegaWithT(t)
	registerBaseMetricsOnly(t)

	cfg := config.New()
	cfg.Scatter.XAxis = new("file-type")
	cfg.Scatter.YAxis = new("file-lines")
	cfg.Scatter.Size = new("file-size")

	cmd := &ScatterCmd{}
	err := cmd.validateConfig(cfg.Scatter)
	g.Expect(err).NotTo(HaveOccurred())
}
