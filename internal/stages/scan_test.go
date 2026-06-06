package stages_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestResolveDimensions_AppliesDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &stages.CommonState{RootConfig: &config.Config{}}

	g.Expect(stages.ResolveDimensions(s)).To(Succeed())
	g.Expect(s.Width).To(Equal(1920))
	g.Expect(s.Height).To(Equal(1080))
}

func TestResolveDimensions_NilRootConfig_UsesDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &stages.CommonState{RootConfig: nil}

	g.Expect(stages.ResolveDimensions(s)).To(Succeed())
	g.Expect(s.Width).To(Equal(1920))
	g.Expect(s.Height).To(Equal(1080))
}

func TestResolveDimensions_UsesConfigValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	width, height := 800, 600
	s := &stages.CommonState{
		RootConfig: &config.Config{ImageSize: &config.ImageSize{Width: &width, Height: &height}},
	}

	g.Expect(stages.ResolveDimensions(s)).To(Succeed())
	g.Expect(s.Width).To(Equal(800))
	g.Expect(s.Height).To(Equal(600))
}

// Smoke test: ScanFilesystem against a tiny tempdir.
func TestScanFilesystem_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "x.txt"), []byte("hi"), 0o600)).To(Succeed())

	s := &stages.CommonState{
		TargetPath: dir,
		Flags:      &stages.Flags{},
	}

	g.Expect(stages.ScanFilesystem(s)).To(Succeed())
	g.Expect(s.Root).NotTo(BeNil())
}
