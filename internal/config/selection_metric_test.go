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
	purpose := metrics[0]
	g.Expect(purpose.Name).To(Equal("code-purpose"))
	g.Expect(purpose.Rules).To(HaveLen(2))
	g.Expect(purpose.Rules[0].Category).To(Equal("test"))
	g.Expect(purpose.Rules[0].Filename).To(Equal("*_test.go"))
	g.Expect(purpose.Rules[1].Category).To(Equal("source"))

	source := metrics[1]
	g.Expect(source.Name).To(Equal("code-source"))
	g.Expect(source.Rules).To(HaveLen(2))
	g.Expect(source.Rules[0].Category).To(Equal("gen"))
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

	first := metrics[0]
	second := metrics[1]
	third := metrics[2]
	g.Expect(first.Name).To(Equal("aaa"))
	g.Expect(second.Name).To(Equal("mmm"))
	g.Expect(third.Name).To(Equal("zzz"))
}
