# File Inclusion/Exclusion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add glob-based file inclusion/exclusion filtering during scanning, controlled by CLI flags and config files.

**Architecture:** A new `internal/filter/` package provides `Rule` types and `IsIncluded()` matching using `doublestar`. Config holds default rules (`.*` exclude). Scanner accepts rules and filters during the walk. CLI exposes `--filter` flags with `!` prefix for excludes.

**Tech Stack:** Go 1.26.1, `github.com/bmatcuk/doublestar/v4`, Kong (CLI), Gomega (test assertions)

**Spec:** `docs/superpowers/specs/2026-04-12-file-inclusion-exclusion-design.md`

---

### Task 1: Add doublestar dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add the doublestar module**

```bash
go get github.com/bmatcuk/doublestar/v4
```

- [ ] **Step 2: Tidy**

```bash
go mod tidy
```

- [ ] **Step 3: Verify**

```bash
grep doublestar go.mod
```

Expected: `github.com/bmatcuk/doublestar/v4 v4.x.x`

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add doublestar/v4 for glob matching"
```

---

### Task 2: Create `internal/filter/` — types and `IsIncluded`

**Files:**
- Create: `internal/filter/filter_test.go`
- Create: `internal/filter/filter.go`

- [ ] **Step 1: Write failing tests for `IsIncluded`**

Create `internal/filter/filter_test.go`:

```go
package filter

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestIsIncluded_NoRules_ReturnsTrue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(IsIncluded("anything.go", nil)).To(BeTrue())
}

func TestIsIncluded_SingleExclude_MatchesEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: ".*", Mode: Exclude}}

	g.Expect(IsIncluded(".git", rules)).To(BeFalse())
	g.Expect(IsIncluded(".gitignore", rules)).To(BeFalse())
}

func TestIsIncluded_SingleExclude_NoMatch_ReturnsTrue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: ".*", Mode: Exclude}}

	g.Expect(IsIncluded("main.go", rules)).To(BeTrue())
	g.Expect(IsIncluded("src/main.go", rules)).To(BeTrue())
}

func TestIsIncluded_SingleInclude_MatchesEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: "*.go", Mode: Include}}

	g.Expect(IsIncluded("main.go", rules)).To(BeTrue())
}

func TestIsIncluded_FirstMatchWins_IncludeBeforeExclude(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{
		{Pattern: ".github", Mode: Include},
		{Pattern: ".github/**", Mode: Include},
		{Pattern: ".*", Mode: Exclude},
	}

	g.Expect(IsIncluded(".github", rules)).To(BeTrue())
	g.Expect(IsIncluded(".github/workflows/ci.yml", rules)).To(BeTrue())
	g.Expect(IsIncluded(".git", rules)).To(BeFalse())
}

func TestIsIncluded_FirstMatchWins_ExcludeBeforeInclude(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{
		{Pattern: ".*", Mode: Exclude},
		{Pattern: ".github/**", Mode: Include},
	}

	// .github matches .* first → excluded
	g.Expect(IsIncluded(".github", rules)).To(BeFalse())
}

func TestIsIncluded_DoublestarPattern(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: "**/*.log", Mode: Exclude}}

	g.Expect(IsIncluded("src/debug.log", rules)).To(BeFalse())
	g.Expect(IsIncluded("src/main.go", rules)).To(BeTrue())
	g.Expect(IsIncluded("debug.log", rules)).To(BeFalse())
}

func TestIsIncluded_InvalidPattern_TreatedAsNoMatch(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rules := []Rule{{Pattern: "[invalid", Mode: Exclude}}

	// Invalid patterns never match, so default (include) applies
	g.Expect(IsIncluded("anything", rules)).To(BeTrue())
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd internal/filter && go test -v -run TestIsIncluded
```

Expected: compilation failure (package/functions don't exist yet)

- [ ] **Step 3: Implement `filter.go`**

Create `internal/filter/filter.go`:

```go
package filter

import (
	"github.com/bmatcuk/doublestar/v4"
)

// Mode indicates whether a rule includes or excludes matching paths.
type Mode int

const (
	// Include means matching paths are included.
	Include Mode = iota
	// Exclude means matching paths are excluded.
	Exclude
)

// Rule pairs a glob pattern with an include/exclude mode.
type Rule struct {
	Pattern string `yaml:"pattern" json:"pattern"`
	Mode    Mode   `yaml:"mode"   json:"mode"`
}

// IsIncluded evaluates relativePath against rules in order.
// The first matching rule wins. Returns true if the entry should be included.
// Default (no match) is include.
func IsIncluded(relativePath string, rules []Rule) bool {
	for _, r := range rules {
		matched, err := doublestar.Match(r.Pattern, relativePath)
		if err != nil {
			// Invalid pattern → skip this rule
			continue
		}

		if matched {
			return r.Mode == Include
		}
	}

	return true
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd internal/filter && go test -v -run TestIsIncluded
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/filter/
git commit -m "feat(filter): add Rule type and IsIncluded glob matching"
```

---

### Task 3: Add `Mode` text marshaling and `ParseFilterFlag`

**Files:**
- Modify: `internal/filter/filter_test.go`
- Modify: `internal/filter/filter.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/filter/filter_test.go`:

```go
func TestModeMarshaling_Include(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text, err := Include.MarshalText()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(text)).To(Equal("include"))

	var m Mode
	g.Expect(m.UnmarshalText([]byte("include"))).To(Succeed())
	g.Expect(m).To(Equal(Include))
}

func TestModeMarshaling_Exclude(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text, err := Exclude.MarshalText()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(text)).To(Equal("exclude"))

	var m Mode
	g.Expect(m.UnmarshalText([]byte("exclude"))).To(Succeed())
	g.Expect(m).To(Equal(Exclude))
}

func TestModeUnmarshaling_Invalid(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var m Mode
	g.Expect(m.UnmarshalText([]byte("bogus"))).To(HaveOccurred())
}

func TestParseFilterFlag_ExcludeWithBang(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rule, err := ParseFilterFlag("!.*")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rule.Pattern).To(Equal(".*"))
	g.Expect(rule.Mode).To(Equal(Exclude))
}

func TestParseFilterFlag_IncludeWithoutPrefix(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rule, err := ParseFilterFlag("*.go")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rule.Pattern).To(Equal("*.go"))
	g.Expect(rule.Mode).To(Equal(Include))
}

func TestParseFilterFlag_DoublestarPattern(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rule, err := ParseFilterFlag("!**/*.log")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rule.Pattern).To(Equal("**/*.log"))
	g.Expect(rule.Mode).To(Equal(Exclude))
}

func TestParseFilterFlag_InvalidGlob(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseFilterFlag("![invalid")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid glob pattern"))
}

func TestParseFilterFlag_EmptyString(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseFilterFlag("")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("empty filter"))
}

func TestParseFilterFlag_BangOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseFilterFlag("!")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("empty filter"))
}
```

- [ ] **Step 2: Run tests to verify new tests fail**

```bash
cd internal/filter && go test -v
```

Expected: compilation failure (MarshalText, ParseFilterFlag don't exist)

- [ ] **Step 3: Implement Mode marshaling and ParseFilterFlag**

Add to `internal/filter/filter.go`:

```go
import (
	"fmt"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rotisserie/eris"
)

// MarshalText implements encoding.TextMarshaler.
func (m Mode) MarshalText() ([]byte, error) {
	switch m {
	case Include:
		return []byte("include"), nil
	case Exclude:
		return []byte("exclude"), nil
	default:
		return nil, fmt.Errorf("unknown filter mode: %d", m)
	}
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *Mode) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "include":
		*m = Include
	case "exclude":
		*m = Exclude
	default:
		return fmt.Errorf("unknown filter mode: %q", string(text))
	}

	return nil
}

// ParseFilterFlag parses a CLI filter string into a Rule.
// A leading ! marks an exclusion; anything else is an inclusion.
func ParseFilterFlag(s string) (Rule, error) {
	if s == "" {
		return Rule{}, eris.New("empty filter string")
	}

	mode := Include
	pattern := s

	if strings.HasPrefix(s, "!") {
		mode = Exclude
		pattern = s[1:]
	}

	if pattern == "" {
		return Rule{}, eris.New("empty filter pattern after prefix")
	}

	// Validate the glob pattern
	if _, err := doublestar.Match(pattern, ""); err != nil {
		return Rule{}, eris.Wrapf(err, "invalid glob pattern %q", pattern)
	}

	return Rule{Pattern: pattern, Mode: mode}, nil
}
```

Note: the import block at the top of `filter.go` needs to be updated to include `fmt`, `strings`, and `eris`. The full import block should be:

```go
import (
	"fmt"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rotisserie/eris"
)
```

- [ ] **Step 4: Run all filter tests**

```bash
cd internal/filter && go test -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/filter/
git commit -m "feat(filter): add Mode text marshaling and ParseFilterFlag"
```

---

### Task 4: Add `FileFilter` to Config

**Files:**
- Modify: `internal/config/config_test.go`
- Modify: `internal/config/config.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/config/config_test.go`:

```go
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
```

The test file also needs the `filter` import added:

```go
import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/filter"
)
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd internal/config && go test -v -run "TestNew_DefaultFileFilter|TestLoad_YAMLFileFilter|TestSave_Load_RoundTrips"
```

Expected: compilation failure (FileFilter doesn't exist on Config)

- [ ] **Step 3: Add FileFilter to Config**

In `internal/config/config.go`, add the import and field:

Add to imports:
```go
"github.com/bevan/code-visualizer/internal/filter"
```

Update the `Config` struct:
```go
type Config struct {
	Width      *int          `yaml:"width,omitempty"      json:"width,omitempty"`
	Height     *int          `yaml:"height,omitempty"     json:"height,omitempty"`
	Treemap    *Treemap      `yaml:"treemap,omitempty"    json:"treemap,omitempty"`
	FileFilter []filter.Rule `yaml:"fileFilter,omitempty" json:"fileFilter,omitempty"`
}
```

Update `New()` to include default rules:
```go
func New() *Config {
	width := 1920
	height := 1080

	return &Config{
		Width:   &width,
		Height:  &height,
		Treemap: &Treemap{},
		FileFilter: []filter.Rule{
			{Pattern: ".*", Mode: filter.Exclude},
		},
	}
}
```

- [ ] **Step 4: Run config tests**

```bash
cd internal/config && go test -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/ internal/filter/
git commit -m "feat(config): add FileFilter with default dotfile exclusion"
```

---

### Task 5: Integrate filtering into Scanner

**Files:**
- Modify: `internal/scan/scanner_test.go`
- Modify: `internal/scan/scanner.go`
- Create: `internal/scan/testdata/with-dotfiles/.hidden` (empty file)
- Create: `internal/scan/testdata/with-dotfiles/.config/settings.json` (content: `{}`)
- Create: `internal/scan/testdata/with-dotfiles/src/main.go` (content: `package main`)
- Create: `internal/scan/testdata/with-dotfiles/README.md` (content: `# Test`)

- [ ] **Step 1: Create testdata fixture**

```bash
mkdir -p internal/scan/testdata/with-dotfiles/.config
mkdir -p internal/scan/testdata/with-dotfiles/src

echo -n "" > internal/scan/testdata/with-dotfiles/.hidden
echo -n "{}" > "internal/scan/testdata/with-dotfiles/.config/settings.json"
echo -n "package main" > internal/scan/testdata/with-dotfiles/src/main.go
echo -n "# Test" > internal/scan/testdata/with-dotfiles/README.md
```

- [ ] **Step 2: Write failing tests**

Append to `internal/scan/scanner_test.go`:

```go
func TestScanWithRules_ExcludesDotfiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-dotfiles")

	rules := []filter.Rule{
		{Pattern: ".*", Mode: filter.Exclude},
	}

	root, err := Scan(dir, rules)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	// .hidden and .config/ should be excluded
	// Only src/main.go and README.md should remain
	allFiles := collectFileNames(root)
	g.Expect(allFiles).To(ConsistOf("main.go", "README.md"))

	allDirs := collectDirNames(root)
	g.Expect(allDirs).To(ConsistOf("src"))
}

func TestScanWithRules_ExcludedDirNotDescended(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-dotfiles")

	rules := []filter.Rule{
		{Pattern: ".*", Mode: filter.Exclude},
	}

	root, err := Scan(dir, rules)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	// .config/ should not appear in the tree at all
	allDirs := collectDirNames(root)
	g.Expect(allDirs).NotTo(ContainElement(".config"))
}

func TestScanWithRules_NoRules_IncludesAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-dotfiles")

	root, err := Scan(dir, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	allFiles := collectFileNames(root)
	g.Expect(allFiles).To(ContainElement("main.go"))
	g.Expect(allFiles).To(ContainElement("README.md"))
	g.Expect(allFiles).To(ContainElement(".hidden"))
	g.Expect(allFiles).To(ContainElement("settings.json"))
}

func TestScanWithRules_IncludeOverridesExclude(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-dotfiles")

	rules := []filter.Rule{
		{Pattern: ".config", Mode: filter.Include},
		{Pattern: ".config/**", Mode: filter.Include},
		{Pattern: ".*", Mode: filter.Exclude},
	}

	root, err := Scan(dir, rules)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	allFiles := collectFileNames(root)
	g.Expect(allFiles).To(ContainElement("settings.json"))
	g.Expect(allFiles).To(ContainElement("main.go"))
	g.Expect(allFiles).To(ContainElement("README.md"))
	g.Expect(allFiles).NotTo(ContainElement(".hidden"))
}

func TestScanWithRules_PrunesEmptyDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-dotfiles")

	// Exclude all .go and .json files — src/ and .config/ should be pruned
	rules := []filter.Rule{
		{Pattern: "**/*.go", Mode: filter.Exclude},
		{Pattern: "**/*.json", Mode: filter.Exclude},
	}

	root, err := Scan(dir, rules)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	allDirs := collectDirNames(root)
	g.Expect(allDirs).NotTo(ContainElement("src"))
	g.Expect(allDirs).NotTo(ContainElement(".config"))
}

// Helper: collect all file names recursively
func collectFileNames(dir *model.Directory) []string {
	var names []string
	for _, f := range dir.Files {
		names = append(names, f.Name)
	}

	for _, d := range dir.Dirs {
		names = append(names, collectFileNames(d)...)
	}

	return names
}

// Helper: collect all directory names recursively (excludes root)
func collectDirNames(dir *model.Directory) []string {
	var names []string
	for _, d := range dir.Dirs {
		names = append(names, d.Name)
		names = append(names, collectDirNames(d)...)
	}

	return names
}
```

Add the filter import to the test file:

```go
"github.com/bevan/code-visualizer/internal/filter"
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
cd internal/scan && go test -v -run "TestScanWithRules"
```

Expected: compilation failure (Scan signature doesn't accept rules)

- [ ] **Step 4: Update scanner to accept and apply rules**

In `internal/scan/scanner.go`:

Update the import block to add:
```go
"github.com/bevan/code-visualizer/internal/filter"
```

Update `Scan()` signature:
```go
func Scan(path string, rules []filter.Rule) (*model.Directory, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, eris.Wrap(err, "failed to resolve absolute path")
	}

	root, err := scanDir(absPath, absPath, rules)
	if err != nil {
		return nil, err
	}

	if countFiles(root) == 0 {
		return nil, errors.New("no files found in directory")
	}

	return root, nil
}
```

Update `scanDir()` to accept and pass through the root path and rules:
```go
func scanDir(dirPath string, rootPath string, rules []filter.Rule) (*model.Directory, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to read directory %s", dirPath)
	}

	node := &model.Directory{
		Path: dirPath,
		Name: filepath.Base(dirPath),
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())

		if err := processEntry(node, entry, entryPath, rootPath, rules); err != nil {
			return nil, err
		}
	}

	return node, nil
}
```

Update `processEntry()`:
```go
func processEntry(node *model.Directory, entry os.DirEntry, entryPath string, rootPath string, rules []filter.Rule) error {
	info, err := os.Stat(entryPath)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("skipping file: permission denied", "path", entryPath)

			return nil
		}

		slog.Warn("skipping file", "path", entryPath, "error", err)

		return nil
	}

	// Compute relative path for filter matching
	relPath, err := filepath.Rel(rootPath, entryPath)
	if err != nil {
		return eris.Wrapf(err, "failed to compute relative path for %s", entryPath)
	}

	if !filter.IsIncluded(relPath, rules) {
		slog.Debug("excluding by filter rule", "path", relPath)

		return nil
	}

	if info.IsDir() {
		return processDir(node, entry, entryPath, rootPath, rules)
	}

	if info.Mode().IsRegular() || isSymlink(entry) {
		processFile(node, entry, info, entryPath)
	}

	return nil
}
```

Update `processDir()`:
```go
func processDir(node *model.Directory, entry os.DirEntry, entryPath string, rootPath string, rules []filter.Rule) error {
	if isSymlink(entry) {
		slog.Debug("skipping directory symlink", "path", entryPath)

		return nil
	}

	child, err := scanDir(entryPath, rootPath, rules)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("skipping directory: permission denied", "path", entryPath)

			return nil
		}

		return err
	}

	// Prune empty directories
	if len(child.Files) > 0 || len(child.Dirs) > 0 {
		node.Dirs = append(node.Dirs, child)
	}

	return nil
}
```

- [ ] **Step 5: Fix existing scanner tests**

All existing `Scan()` calls need to pass `nil` as the rules parameter. Update every call in `internal/scan/scanner_test.go`:

- `Scan(dir)` → `Scan(dir, nil)`

This applies to: `TestScanFlat`, `TestScanNested`, `TestScanEmptyDir`, `TestScanFollowsFileSymlinks`, `TestScanSkipsDirSymlinks`, `TestScanFileExtension`, `TestScanSetsFileType`, and any platform-specific tests in `scanner_unix_test.go`.

- [ ] **Step 6: Run all scanner tests**

```bash
cd internal/scan && go test -v
```

Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/scan/
git commit -m "feat(scan): integrate filter rules into directory scanning"
```

---

### Task 6: Update CLI to pass filter rules to scanner

**Files:**
- Modify: `cmd/codeviz/treemap_cmd.go`

- [ ] **Step 1: Add `--filter` flag and filter processing to `TreemapCmd`**

In `cmd/codeviz/treemap_cmd.go`, add the import:
```go
"github.com/bevan/code-visualizer/internal/filter"
```

Add the field to `TreemapCmd`:
```go
Filter []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)." short:"f"` //nolint:revive // kong struct tags require long lines
```

Add validation in `Validate()`, before the final `return nil`:
```go
for _, f := range c.Filter {
	if _, err := filter.ParseFilterFlag(f); err != nil {
		return eris.Wrapf(err, "invalid filter %q", f)
	}
}
```

Add a helper method:
```go
func (c *TreemapCmd) buildFilterRules(cfg *config.Config) []filter.Rule {
	rules := make([]filter.Rule, len(cfg.FileFilter))
	copy(rules, cfg.FileFilter)

	for _, f := range c.Filter {
		// Already validated in Validate()
		rule, _ := filter.ParseFilterFlag(f)
		rules = append(rules, rule)
	}

	return rules
}
```

Update `Run()` to build and pass rules to scan. Replace the `scan.Scan(c.TargetPath)` call:
```go
filterRules := c.buildFilterRules(flags.Config)

root, err := scan.Scan(c.TargetPath, filterRules)
```

- [ ] **Step 2: Build and verify**

```bash
go build ./cmd/codeviz/
```

Expected: compiles cleanly

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/codeviz/
git commit -m "feat(cli): add --filter flag for file inclusion/exclusion"
```

---

### Task 7: Update CLI tests

**Files:**
- Modify: `cmd/codeviz/main_test.go`

- [ ] **Step 1: Read existing CLI tests to understand patterns**

Read `cmd/codeviz/main_test.go` to understand the existing test structure before adding new tests.

- [ ] **Step 2: Write filter validation tests**

Add tests for filter validation (append to `cmd/codeviz/main_test.go` or create a dedicated test if the existing pattern suggests it):

```go
func TestTreemapCmd_Validate_InvalidFilterGlob(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &TreemapCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "file-size",
		Filter:     []string{"![invalid"},
	}

	err := cmd.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid filter"))
}
```

Note: The exact test structure depends on what's in `main_test.go`. Adapt to match the existing testing patterns.

- [ ] **Step 3: Run tests**

```bash
cd cmd/codeviz && go test -v
```

Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/codeviz/
git commit -m "test(cli): add filter validation tests"
```

---

### Task 8: Final verification

- [ ] **Step 1: Run full test suite with race detector**

```bash
go test -race ./...
```

Expected: all PASS, no data races

- [ ] **Step 2: Run linter**

```bash
task lint
```

Expected: 0 issues

- [ ] **Step 3: Build and smoke test**

```bash
task build
./bin/codeviz render treemap --filter '!.*' -s file-size -o /tmp/test-filtered.png .
```

Expected: produces a PNG without dotfiles/dotdirs

- [ ] **Step 4: Smoke test without filters (defaults apply)**

```bash
./bin/codeviz render treemap -s file-size -o /tmp/test-default.png .
```

Expected: produces a PNG; dotfiles excluded by default `.*` rule

- [ ] **Step 5: Commit any final fixes and push**

```bash
git push
```
