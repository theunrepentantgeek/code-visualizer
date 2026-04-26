package config

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"

	"go.yaml.in/yaml/v3"
)

// UnmarshalText tests

func TestMetricSpec_UnmarshalText_MetricOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms MetricSpec

	err := ms.UnmarshalText([]byte("file-size"))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ms.Metric).To(Equal("file-size"))
	g.Expect(ms.Palette).To(BeEmpty())
}

func TestMetricSpec_UnmarshalText_MetricAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ms MetricSpec

	err := ms.UnmarshalText([]byte("file-type,categorization"))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ms.Metric).To(Equal("file-type"))
	g.Expect(ms.Palette).To(Equal("categorization"))
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
	g.Expect(ms.Metric).To(Equal("file-lines"))
	g.Expect(ms.Palette).To(Equal("foliage"))
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

func TestMetricSpec_UnmarshalText_ExtraCommas_TreatedAsPartOfPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// SplitN(s, ",", 2) means only the first comma separates metric from palette.
	// Extra commas remain part of the palette name (and will be caught by validation later).
	var ms MetricSpec

	err := ms.UnmarshalText([]byte("file-type,a,b"))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ms.Metric).To(Equal("file-type"))
	g.Expect(ms.Palette).To(Equal("a,b"))
}

// String tests

func TestMetricSpec_String_MetricOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := MetricSpec{Metric: "file-size"}
	g.Expect(ms.String()).To(Equal("file-size"))
}

func TestMetricSpec_String_MetricAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ms := MetricSpec{Metric: "file-type", Palette: "categorization"}
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
	g.Expect(MetricSpec{Metric: "file-size"}.IsZero()).To(BeFalse())
}

// YAML round-trip tests

func TestMetricSpec_YAML_RoundTrip_MetricOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	original := MetricSpec{Metric: "file-lines"}

	data, err := yaml.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("file-lines"))

	var loaded MetricSpec
	g.Expect(yaml.Unmarshal(data, &loaded)).To(Succeed())
	g.Expect(loaded).To(Equal(original))
}

func TestMetricSpec_YAML_RoundTrip_MetricAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	original := MetricSpec{Metric: "file-type", Palette: "categorization"}

	data, err := yaml.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("file-type,categorization"))

	var loaded MetricSpec
	g.Expect(yaml.Unmarshal(data, &loaded)).To(Succeed())
	g.Expect(loaded).To(Equal(original))
}

// JSON round-trip tests

func TestMetricSpec_JSON_RoundTrip_MetricOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	original := MetricSpec{Metric: "file-lines"}

	data, err := json.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(Equal(`"file-lines"`))

	var loaded MetricSpec
	g.Expect(json.Unmarshal(data, &loaded)).To(Succeed())
	g.Expect(loaded).To(Equal(original))
}

func TestMetricSpec_JSON_RoundTrip_MetricAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	original := MetricSpec{Metric: "file-type", Palette: "categorization"}

	data, err := json.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(Equal(`"file-type,categorization"`))

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

	original := wrapper{Fill: &MetricSpec{Metric: "file-type", Palette: "categorization"}}

	data, err := yaml.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())

	var loaded wrapper
	g.Expect(yaml.Unmarshal(data, &loaded)).To(Succeed())
	g.Expect(loaded.Fill).NotTo(BeNil())
	g.Expect(*loaded.Fill).To(Equal(*original.Fill))
}
