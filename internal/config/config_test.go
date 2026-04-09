package config

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// New tests

func TestNew_ReturnsNonNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	cfg := New()

	// Assert
	g.Expect(cfg).NotTo(BeNil())
}

func TestNew_TreemapDefaultsSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	cfg := New()

	// Assert
	g.Expect(cfg.Treemap).NotTo(BeNil())
	g.Expect(cfg.Treemap.Width).NotTo(BeNil())
	g.Expect(*cfg.Treemap.Width).To(Equal(1920))
	g.Expect(cfg.Treemap.Height).NotTo(BeNil())
	g.Expect(*cfg.Treemap.Height).To(Equal(1080))
}

func TestNew_OptionalFieldsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	cfg := New()

	// Assert
	g.Expect(cfg.Treemap.Fill).To(BeNil())
	g.Expect(cfg.Treemap.FillPalette).To(BeNil())
	g.Expect(cfg.Treemap.Border).To(BeNil())
	g.Expect(cfg.Treemap.BorderPalette).To(BeNil())
}

// Load tests

func TestLoad_UnknownExtension_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	g.Expect(os.WriteFile(path, []byte("[treemap]\nwidth = 800\n"), 0o600)).To(Succeed())

	cfg := New()

	// Act
	err := Load(path, cfg)

	// Assert
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unsupported config file extension"))
}

func TestLoad_MissingFile_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	cfg := New()

	// Act
	err := Load("/nonexistent/path/config.yaml", cfg)

	// Assert
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to read config file"))
}

func TestLoad_YAMLPartialConfig_OverridesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := "treemap:\n  width: 800\n"
	g.Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())

	cfg := New()

	// Act
	err := Load(path, cfg)

	// Assert
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(*cfg.Treemap.Width).To(Equal(800))
	g.Expect(*cfg.Treemap.Height).To(Equal(1080)) // default preserved
}

func TestLoad_YMLExtension_ParsesCorrectly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := "treemap:\n  height: 720\n"
	g.Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())

	cfg := New()

	// Act
	err := Load(path, cfg)

	// Assert
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(*cfg.Treemap.Height).To(Equal(720))
	g.Expect(*cfg.Treemap.Width).To(Equal(1920)) // default preserved
}

func TestLoad_JSONConfig_OverridesFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{"treemap":{"fill":"file-type","fillPalette":"categorization"}}`
	g.Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())

	cfg := New()

	// Act
	err := Load(path, cfg)

	// Assert
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cfg.Treemap.Fill).NotTo(BeNil())
	g.Expect(*cfg.Treemap.Fill).To(Equal("file-type"))
	g.Expect(cfg.Treemap.FillPalette).NotTo(BeNil())
	g.Expect(*cfg.Treemap.FillPalette).To(Equal("categorization"))
}

func TestLoad_InvalidYAML_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	g.Expect(os.WriteFile(path, []byte(":\t: bad yaml"), 0o600)).To(Succeed())

	cfg := New()

	// Act
	err := Load(path, cfg)

	// Assert
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to parse YAML"))
}

func TestLoad_InvalidJSON_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	g.Expect(os.WriteFile(path, []byte("{not valid json"), 0o600)).To(Succeed())

	cfg := New()

	// Act
	err := Load(path, cfg)

	// Assert
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to parse JSON"))
}

// Save tests

func TestSave_UnknownExtension_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	err := Save("/tmp/config.toml", New())

	// Assert
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unsupported config file extension"))
}

func TestSave_YAML_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	cfg := New()

	// Act
	err := Save(path, cfg)

	// Assert
	g.Expect(err).NotTo(HaveOccurred())
	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("width: 1920"))
	g.Expect(string(data)).To(ContainSubstring("height: 1080"))
}

func TestSave_JSON_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	cfg := New()

	// Act
	err := Save(path, cfg)

	// Assert
	g.Expect(err).NotTo(HaveOccurred())
	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring(`"width": 1920`))
	g.Expect(string(data)).To(ContainSubstring(`"height": 1080`))
}

func TestSave_ThenLoad_RoundTrips(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	fill := "file-type"
	original := New()
	original.Treemap.Fill = &fill

	// Act: save then load into fresh config
	g.Expect(Save(path, original)).To(Succeed())

	loaded := New()
	g.Expect(Load(path, loaded)).To(Succeed())

	// Assert
	g.Expect(*loaded.Treemap.Width).To(Equal(1920))
	g.Expect(*loaded.Treemap.Height).To(Equal(1080))
	g.Expect(loaded.Treemap.Fill).NotTo(BeNil())
	g.Expect(*loaded.Treemap.Fill).To(Equal("file-type"))
}

func TestSave_OmitsNilFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	cfg := New() // Fill, Border etc. are nil

	// Act
	g.Expect(Save(path, cfg)).To(Succeed())

	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())

	// Assert: nil pointer fields should not appear in output
	g.Expect(string(data)).NotTo(ContainSubstring("fill"))
	g.Expect(string(data)).NotTo(ContainSubstring("border"))
}
