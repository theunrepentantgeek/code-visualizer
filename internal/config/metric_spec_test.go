package config

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"

	"go.yaml.in/yaml/v3"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

// TestMain registers filesystem providers so Validate tests can look up known metrics.
func TestMain(m *testing.M) {
	filesystem.Register()
	m.Run()
}

// UnmarshalText tests

func TestMetricSpec_UnmarshalText_MetricOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms MetricSpec

	err := ms.UnmarshalText([]byte("file-size"))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ms.Metric).To(Equal(metric.Name("file-size")))
	g.Expect(ms.Palette).To(BeEmpty())
}

func TestMetricSpec_UnmarshalText_MetricAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms MetricSpec

	err := ms.UnmarshalText([]byte("file-type,categorization"))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ms.Metric).To(Equal(metric.Name("file-type")))
	g.Expect(ms.Palette).To(Equal(palette.PaletteName("categorization")))
}

func TestMetricSpec_UnmarshalText_EmptyString(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms MetricSpec

	err := ms.UnmarshalText([]byte(""))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ms.IsZero()).To(BeTrue())
}

func TestMetricSpec_UnmarshalText_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms MetricSpec

	err := ms.UnmarshalText([]byte("  "))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ms.IsZero()).To(BeTrue())
}

func TestMetricSpec_UnmarshalText_TrimsWhitespace(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms MetricSpec

	err := ms.UnmarshalText([]byte(" file-lines , foliage "))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ms.Metric).To(Equal(metric.Name("file-lines")))
	g.Expect(ms.Palette).To(Equal(palette.PaletteName("foliage")))
}

func TestMetricSpec_UnmarshalText_EmptyMetric_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms MetricSpec

	err := ms.UnmarshalText([]byte(",categorization"))

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("metric name must not be empty")))
}

func TestMetricSpec_UnmarshalText_EmptyPalette_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms MetricSpec

	err := ms.UnmarshalText([]byte("file-type,"))

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("palette name must not be empty after comma")))
}

func TestMetricSpec_UnmarshalText_ExtraCommas_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// strings.Cut twice means a third comma-delimited part is rejected.
	var ms MetricSpec

	err := ms.UnmarshalText([]byte("file-type,a,b"))

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("unexpected extra content")))
}

// String tests

func TestMetricSpec_String_MetricOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := MetricSpec{Metric: metric.Name("file-size")}
	g.Expect(ms.String()).To(Equal("file-size"))
}

func TestMetricSpec_String_MetricAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := MetricSpec{Metric: metric.Name("file-type"), Palette: palette.PaletteName("categorization")}
	g.Expect(ms.String()).To(Equal("file-type,categorization"))
}

func TestMetricSpec_String_Zero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := MetricSpec{}
	g.Expect(ms.String()).To(BeEmpty())
}

// IsZero tests

func TestMetricSpec_IsZero_True(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(MetricSpec{}.IsZero()).To(BeTrue())
}

func TestMetricSpec_IsZero_False(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(MetricSpec{Metric: metric.Name("file-size")}.IsZero()).To(BeFalse())
}

// YAML round-trip tests

func TestMetricSpec_YAML_RoundTrip_MetricOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	original := MetricSpec{Metric: metric.Name("file-lines")}

	data, err := yaml.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("metric: file-lines"))
	g.Expect(string(data)).NotTo(ContainSubstring("palette"))

	var loaded MetricSpec
	g.Expect(yaml.Unmarshal(data, &loaded)).To(Succeed())
	g.Expect(loaded).To(Equal(original))
}

func TestMetricSpec_YAML_RoundTrip_MetricAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	original := MetricSpec{Metric: metric.Name("file-type"), Palette: palette.PaletteName("categorization")}

	data, err := yaml.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("metric: file-type"))
	g.Expect(string(data)).To(ContainSubstring("palette: categorization"))

	var loaded MetricSpec
	g.Expect(yaml.Unmarshal(data, &loaded)).To(Succeed())
	g.Expect(loaded).To(Equal(original))
}

// JSON round-trip tests

func TestMetricSpec_JSON_RoundTrip_MetricOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	original := MetricSpec{Metric: metric.Name("file-lines")}

	data, err := json.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(Equal(`{"metric":"file-lines"}`))

	var loaded MetricSpec
	g.Expect(json.Unmarshal(data, &loaded)).To(Succeed())
	g.Expect(loaded).To(Equal(original))
}

func TestMetricSpec_JSON_RoundTrip_MetricAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	original := MetricSpec{Metric: metric.Name("file-type"), Palette: palette.PaletteName("categorization")}

	data, err := json.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(Equal(`{"metric":"file-type","palette":"categorization"}`))

	var loaded MetricSpec
	g.Expect(json.Unmarshal(data, &loaded)).To(Succeed())
	g.Expect(loaded).To(Equal(original))
}

// Pointer YAML test (config files use *MetricSpec)

func TestMetricSpec_YAML_Pointer_OmitsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	type wrapper struct {
		Fill *MetricSpec `yaml:"fill,omitempty"`
	}

	data, err := yaml.Marshal(wrapper{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(Equal("{}\n"))
}

func TestMetricSpec_YAML_Pointer_RoundTrips(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	type wrapper struct {
		Fill *MetricSpec `yaml:"fill,omitempty"`
	}

	original := wrapper{
		Fill: &MetricSpec{
			Metric:  metric.Name("file-type"),
			Palette: palette.PaletteName("categorization"),
		},
	}

	data, err := yaml.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("metric: file-type"))
	g.Expect(string(data)).To(ContainSubstring("palette: categorization"))

	var loaded wrapper
	g.Expect(yaml.Unmarshal(data, &loaded)).To(Succeed())
	g.Expect(loaded.Fill).NotTo(BeNil())
	g.Expect(*loaded.Fill).To(Equal(*original.Fill))
}

// Zero/omitempty tests

func TestMetricSpec_YAML_OmitsEmptyPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := MetricSpec{Metric: metric.Name("file-size")}

	data, err := yaml.Marshal(ms)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("metric: file-size"))
	g.Expect(string(data)).NotTo(ContainSubstring("palette"))
}

func TestMetricSpec_JSON_OmitsEmptyPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := MetricSpec{Metric: metric.Name("file-size")}

	data, err := json.Marshal(ms)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(Equal(`{"metric":"file-size"}`))
	g.Expect(string(data)).NotTo(ContainSubstring("palette"))
}

// YAML scalar fallback test (backward compatibility)

func TestMetricSpec_YAML_UnmarshalScalar_Fallback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms MetricSpec
	g.Expect(yaml.Unmarshal([]byte(`"file-type,categorization"`), &ms)).To(Succeed())
	g.Expect(ms.Metric).To(Equal(metric.Name("file-type")))
	g.Expect(ms.Palette).To(Equal(palette.PaletteName("categorization")))
}

// MarshalText tests

func TestMetricSpec_MarshalText_MetricOnly_ProducesMetricBytes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := MetricSpec{Metric: metric.Name("file-size")}
	b, err := ms.MarshalText()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(b)).To(Equal("file-size"))
}

func TestMetricSpec_MarshalText_MetricAndPalette_ProducesFullBytes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := MetricSpec{Metric: metric.Name("file-type"), Palette: palette.PaletteName("categorization")}
	b, err := ms.MarshalText()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(b)).To(Equal("file-type,categorization"))
}

func TestMetricSpec_MarshalText_Zero_ProducesEmptyBytes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b, err := MetricSpec{}.MarshalText()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(b).To(BeEmpty())
}

// Validate tests

func TestMetricSpec_Validate_NilReceiver_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms *MetricSpec
	g.Expect(ms.Validate("fill")).To(Succeed())
}

func TestMetricSpec_Validate_EmptySpec_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect((&MetricSpec{}).Validate("fill")).To(Succeed())
}

func TestMetricSpec_Validate_KnownMetric_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := &MetricSpec{Metric: "file-size"}
	g.Expect(ms.Validate("size")).To(Succeed())
}

func TestMetricSpec_Validate_UnknownMetric_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := &MetricSpec{Metric: "not-a-real-metric"}
	err := ms.Validate("fill")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring(`invalid fill metric "not-a-real-metric"`)))
}

func TestMetricSpec_Validate_KnownMetricAndValidPalette_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := &MetricSpec{Metric: "file-lines", Palette: palette.Temperature}
	g.Expect(ms.Validate("fill")).To(Succeed())
}

func TestMetricSpec_Validate_InvalidPalette_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := &MetricSpec{Metric: "file-size", Palette: "no-such-palette"}
	err := ms.Validate("fill")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring(`invalid fill palette "no-such-palette"`)))
}

func TestMetricSpec_Validate_EmptyMetricWithInvalidPalette_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Palette is checked even when Metric is empty.
	ms := &MetricSpec{Palette: "bogus"}
	err := ms.Validate("border")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring(`invalid border palette "bogus"`)))
}
