package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"go.yaml.in/yaml/v3"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func configWithSelectionMetrics(t *testing.T, yamlSnippet string) *config.Config {
	t.Helper()

	cfg := config.New()

	err := yaml.Unmarshal([]byte(yamlSnippet), cfg)
	if err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	return cfg
}

func TestRegisterSelectionMetrics_RegistersConfiguredMetrics(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	cfg := configWithSelectionMetrics(t, `
selectionMetrics:
  sm-test-purpose:
    - category: test
      filename: "*_test.go"
    - category: source
      filename: "*"
`)

	s := &stages.CommonState{
		Flags: &stages.Flags{Config: cfg},
	}

	err := stages.RegisterSelectionMetrics(s)
	g.Expect(err).NotTo(HaveOccurred())

	desc, ok := provider.GetBase(metric.Name("sm-test-purpose"))
	g.Expect(ok).To(BeTrue())
	g.Expect(desc.Kind).To(Equal(metric.Classification))
}

func TestRegisterSelectionMetrics_IdempotentOnRepeat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	cfg := configWithSelectionMetrics(t, `
selectionMetrics:
  sm-idempotent-metric:
    - category: all
      filename: "*"
`)

	s := &stages.CommonState{
		Flags: &stages.Flags{Config: cfg},
	}

	// Calling twice must not panic (duplicate registration is silently skipped).
	g.Expect(stages.RegisterSelectionMetrics(s)).To(Succeed())
	g.Expect(stages.RegisterSelectionMetrics(s)).To(Succeed())
}

func TestRegisterSelectionMetrics_NoopWhenNoMetrics(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	beforeCount := len(provider.BaseNames())

	s := &stages.CommonState{
		Flags: &stages.Flags{Config: config.New()},
	}

	g.Expect(stages.RegisterSelectionMetrics(s)).To(Succeed())
	// No new providers should have been registered.
	g.Expect(provider.BaseNames()).To(HaveLen(beforeCount))
}

func TestRegisterSelectionMetrics_NoopWhenNilFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	s := &stages.CommonState{}

	g.Expect(stages.RegisterSelectionMetrics(s)).To(Succeed())
}

func TestRegisterSelectionMetrics_InvalidPattern_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	cfg := configWithSelectionMetrics(t, `
selectionMetrics:
  bad-metric:
    - category: broken
      filename: ""
`)

	s := &stages.CommonState{
		Flags: &stages.Flags{Config: cfg},
	}

	err := stages.RegisterSelectionMetrics(s)
	g.Expect(err).To(HaveOccurred())
	//nolint:nilaway,nolintlint // guarded by HaveOccurred above
	g.Expect(err.Error()).To(ContainSubstring("invalid selection metric configuration"))
}
