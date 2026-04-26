package config

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/filter"
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

func TestNew_DefaultsSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	cfg := New()

	// Assert
	g.Expect(cfg.Width).NotTo(BeNil())
	g.Expect(*cfg.Width).To(Equal(1920))
	g.Expect(cfg.Height).NotTo(BeNil())
	g.Expect(*cfg.Height).To(Equal(1080))
}

func TestNew_TreemapDefaultsSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	cfg := New()

	// Assert
	g.Expect(cfg.Treemap).NotTo(BeNil())
}

func TestNew_OptionalFieldsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	cfg := New()

	// Assert
	g.Expect(cfg.Treemap.Fill).To(BeNil())
	g.Expect(cfg.Treemap.Border).To(BeNil())
}

// Load tests

func TestLoad_UnknownExtension_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	g.Expect(os.WriteFile(path, []byte("[treemap]\nfill = \"file-type\"\n"), 0o600)).To(Succeed())

	cfg := New()

	// Act
	err := cfg.Load(path)

	// Assert
	g.Expect(err).To(MatchError(ContainSubstring("unsupported config file extension")))
}

func TestLoad_MissingFile_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	cfg := New()

	// Act
	err := cfg.Load("/nonexistent/path/config.yaml")

	// Assert
	g.Expect(err).To(MatchError(ContainSubstring("failed to read config file")))
}

func TestLoad_YAMLPartialConfig_OverridesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := "width: 800\n"
	g.Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())

	cfg := New()

	// Act
	err := cfg.Load(path)

	// Assert
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(*cfg.Width).To(Equal(800))
	g.Expect(*cfg.Height).To(Equal(1080)) // default preserved
}

func TestLoad_YMLExtension_ParsesCorrectly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := "height: 720\n"
	g.Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())

	cfg := New()

	// Act
	err := cfg.Load(path)

	// Assert
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(*cfg.Height).To(Equal(720))
	g.Expect(*cfg.Width).To(Equal(1920)) // default preserved
}

func TestLoad_JSONConfig_OverridesFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{"treemap":{"fill":"file-type,categorization"}}`
	g.Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())

	cfg := New()

	// Act
	err := cfg.Load(path)

	// Assert
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cfg.Treemap.Fill).NotTo(BeNil())
	g.Expect(cfg.Treemap.Fill.Metric).To(Equal("file-type"))
	g.Expect(cfg.Treemap.Fill.Palette).To(Equal("categorization"))
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
	err := cfg.Load(path)

	// Assert
	g.Expect(err).To(MatchError(ContainSubstring("failed to parse YAML")))
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
	err := cfg.Load(path)

	// Assert
	g.Expect(err).To(MatchError(ContainSubstring("failed to parse JSON")))
}

// Save tests

func TestSave_UnknownExtension_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	err := New().Save("/tmp/config.toml")

	// Assert
	g.Expect(err).To(MatchError(ContainSubstring("unsupported config file extension")))
}

func TestSave_YAML_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	cfg := New()

	// Act
	err := cfg.Save(path)

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
	err := cfg.Save(path)

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

	original := New()
	original.Treemap.Fill = &MetricSpec{Metric: "file-type"}

	// Act: save then load into fresh config
	g.Expect(original.Save(path)).To(Succeed())

	loaded := New()
	g.Expect(loaded.Load(path)).To(Succeed())

	// Assert
	g.Expect(*loaded.Width).To(Equal(1920))
	g.Expect(*loaded.Height).To(Equal(1080))
	g.Expect(loaded.Treemap.Fill).NotTo(BeNil())
	g.Expect(loaded.Treemap.Fill.Metric).To(Equal("file-type"))
}

func TestSave_OmitsNilFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	cfg := New() // Fill, Border etc. are nil

	// Act
	g.Expect(cfg.Save(path)).To(Succeed())

	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())

	// Assert: nil pointer fields should not appear in output
	g.Expect(string(data)).NotTo(ContainSubstring("fill"))
	g.Expect(string(data)).NotTo(ContainSubstring("border"))
}

func TestNew_DefaultFileFilter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := New()

	g.Expect(cfg.FileFilter).To(HaveLen(1))
	g.Expect(cfg.FileFilter[0].Pattern).To(Equal(".*"))
	g.Expect(cfg.FileFilter[0].Mode).To(Equal(filter.Exclude))
}

func TestLoad_YAMLFileFilter_ReplacesDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `fileFilter:
  - pattern: ".github/**"
    mode: include
  - pattern: ".*"
    mode: exclude
  - pattern: "**/*.log"
    mode: exclude
`
	g.Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())

	cfg := New()
	err := cfg.Load(path)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cfg.FileFilter).To(HaveLen(3))
	g.Expect(cfg.FileFilter[0].Pattern).To(Equal(".github/**"))
	g.Expect(cfg.FileFilter[0].Mode).To(Equal(filter.Include))
	g.Expect(cfg.FileFilter[1].Pattern).To(Equal(".*"))
	g.Expect(cfg.FileFilter[2].Pattern).To(Equal("**/*.log"))
}

func TestSave_Load_RoundTripsFileFilter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := New()
	original.FileFilter = []filter.Rule{
		{Pattern: ".*", Mode: filter.Exclude},
		{Pattern: ".github/**", Mode: filter.Include},
	}

	g.Expect(original.Save(path)).To(Succeed())

	loaded := New()
	g.Expect(loaded.Load(path)).To(Succeed())

	g.Expect(loaded.FileFilter).To(HaveLen(2))
	g.Expect(loaded.FileFilter[0].Pattern).To(Equal(".*"))
	g.Expect(loaded.FileFilter[0].Mode).To(Equal(filter.Exclude))
	g.Expect(loaded.FileFilter[1].Pattern).To(Equal(".github/**"))
	g.Expect(loaded.FileFilter[1].Mode).To(Equal(filter.Include))
}

// FindAutoConfig tests

func TestFindAutoConfig_NoFileExists_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	result, ok := FindAutoConfig("/tmp/nonexistent-output.png")

	g.Expect(ok).To(BeFalse())
	g.Expect(result).To(BeEmpty())
}

func TestFindAutoConfig_YMLExists_ReturnsYMLPath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "my-output.png")
	configPath := filepath.Join(dir, "my-output-config.yml")
	g.Expect(os.WriteFile(configPath, []byte("width: 800\n"), 0o600)).To(Succeed())

	result, ok := FindAutoConfig(outputPath)

	g.Expect(ok).To(BeTrue())
	g.Expect(result).To(Equal(configPath))
}

func TestFindAutoConfig_YAMLExists_ReturnsYAMLPath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "my-output.png")
	configPath := filepath.Join(dir, "my-output-config.yaml")
	g.Expect(os.WriteFile(configPath, []byte("width: 800\n"), 0o600)).To(Succeed())

	result, ok := FindAutoConfig(outputPath)

	g.Expect(ok).To(BeTrue())
	g.Expect(result).To(Equal(configPath))
}

func TestFindAutoConfig_JSONExists_ReturnsJSONPath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "my-output.png")
	configPath := filepath.Join(dir, "my-output-config.json")
	g.Expect(os.WriteFile(configPath, []byte(`{"width":800}`), 0o600)).To(Succeed())

	result, ok := FindAutoConfig(outputPath)

	g.Expect(ok).To(BeTrue())
	g.Expect(result).To(Equal(configPath))
}

func TestFindAutoConfig_YMLTakesPrecedenceOverYAML(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "my-output.png")
	ymlPath := filepath.Join(dir, "my-output-config.yml")
	yamlPath := filepath.Join(dir, "my-output-config.yaml")

	g.Expect(os.WriteFile(ymlPath, []byte("width: 800\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(yamlPath, []byte("width: 900\n"), 0o600)).To(Succeed())

	result, ok := FindAutoConfig(outputPath)

	g.Expect(ok).To(BeTrue())
	g.Expect(result).To(Equal(ymlPath))
}

func TestFindAutoConfig_SVGOutput_StillFindsConfig(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "my-output.svg")
	configPath := filepath.Join(dir, "my-output-config.yml")
	g.Expect(os.WriteFile(configPath, []byte("width: 800\n"), 0o600)).To(Succeed())

	result, ok := FindAutoConfig(outputPath)

	g.Expect(ok).To(BeTrue())
	g.Expect(result).To(Equal(configPath))
}

// TryAutoLoad tests

func TestTryAutoLoad_NoConfigFile_NoChange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "output.png")

	cfg := New()
	err := cfg.TryAutoLoad(outputPath)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cfg.Source).To(BeNil())
}

func TestTryAutoLoad_ConfigFilePresent_LoadsIt(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "output.png")
	configPath := filepath.Join(dir, "output-config.yml")
	g.Expect(os.WriteFile(configPath, []byte("width: 800\n"), 0o600)).To(Succeed())

	cfg := New()
	err := cfg.TryAutoLoad(outputPath)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cfg.Source).NotTo(BeNil())
	g.Expect(*cfg.Source).To(Equal(configPath))
	g.Expect(*cfg.Width).To(Equal(800))
}

func TestTryAutoLoad_AlreadyLoaded_SkipsAutoLoad(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "output.png")

	// Write an auto-config that would set width=800
	autoConfigPath := filepath.Join(dir, "output-config.yml")
	g.Expect(os.WriteFile(autoConfigPath, []byte("width: 800\n"), 0o600)).To(Succeed())

	// Manually load a different config first (sets Source)
	manualConfigPath := filepath.Join(dir, "manual.yml")
	g.Expect(os.WriteFile(manualConfigPath, []byte("width: 1600\n"), 0o600)).To(Succeed())

	cfg := New()
	g.Expect(cfg.Load(manualConfigPath)).To(Succeed())

	// TryAutoLoad should be a no-op because Source is already set
	err := cfg.TryAutoLoad(outputPath)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(*cfg.Width).To(Equal(1600)) // unchanged
}
