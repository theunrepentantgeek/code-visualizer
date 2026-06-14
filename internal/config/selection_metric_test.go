package config_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
)

func TestSelectionMetricsRoundTrip(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	input := `
selection-metrics:
  code-purpose:
    - category: test
      filename: "*_test.go"
    - category: source
      filename: "*"
  code-source:
    - category: gen
      filename: "*_gen.go"
    - category: authored
      filename: "*"
`

	cfg := config.New()
	err := yaml.Unmarshal([]byte(input), cfg)
	g.Expect(err).NotTo(HaveOccurred())

	metrics := cfg.SelectionMetricsList()
	g.Expect(metrics).To(HaveLen(2))

	// Sorted by name: code-purpose < code-source
	g.Expect(metrics[0].Name).To(Equal("code-purpose"))
	g.Expect(metrics[0].Rules).To(HaveLen(2))
	g.Expect(metrics[0].Rules[0].Category).To(Equal("test"))
	g.Expect(metrics[0].Rules[0].Filename).To(Equal("*_test.go"))
	g.Expect(metrics[0].Rules[1].Category).To(Equal("source"))

	g.Expect(metrics[1].Name).To(Equal("code-source"))
	g.Expect(metrics[1].Rules).To(HaveLen(2))
	g.Expect(metrics[1].Rules[0].Category).To(Equal("gen"))
}

func TestSelectionMetricsList_NilConfig(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var cfg *config.Config
	g.Expect(cfg.SelectionMetricsList()).To(BeNil())
}

func TestSelectionMetricsList_EmptyConfig(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cfg := config.New()
	g.Expect(cfg.SelectionMetricsList()).To(BeNil())
}

func TestSelectionMetricsList_StableSortOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	input := `
selection-metrics:
  zzz:
    - category: z
      filename: "*"
  aaa:
    - category: a
      filename: "*"
  mmm:
    - category: m
      filename: "*"
`

	cfg := config.New()
	err := yaml.Unmarshal([]byte(input), cfg)
	g.Expect(err).NotTo(HaveOccurred())

	metrics := cfg.SelectionMetricsList()
	g.Expect(metrics).To(HaveLen(3))
	g.Expect(metrics[0].Name).To(Equal("aaa"))
	g.Expect(metrics[1].Name).To(Equal("mmm"))
	g.Expect(metrics[2].Name).To(Equal("zzz"))
}
