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

	s := &fakeState{common: stages.CommonState{RootConfig: &config.Config{}}}

	g.Expect(stages.ResolveDimensions[*fakeState](s)).To(Succeed())
	g.Expect(s.Common().Width).To(Equal(1920))
	g.Expect(s.Common().Height).To(Equal(1080))
}

func TestResolveDimensions_UsesConfigValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	g.Expect(os.WriteFile(path, []byte("imageSize:\n  width: 800\n  height: 600\n"), 0o600)).To(Succeed())

	cfg := config.New()
	g.Expect(cfg.Load(path)).To(Succeed())

	s := &fakeState{common: stages.CommonState{RootConfig: cfg}}

	g.Expect(stages.ResolveDimensions[*fakeState](s)).To(Succeed())
	g.Expect(s.Common().Width).To(Equal(800))
	g.Expect(s.Common().Height).To(Equal(600))
}

// Smoke test: ScanFilesystem against a tiny tempdir.
func TestScanFilesystem_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "x.txt"), []byte("hi"), 0o600)).To(Succeed())

	s := &fakeState{common: stages.CommonState{
		TargetPath: dir,
		Flags:      &stages.Flags{},
	}}

	g.Expect(stages.ScanFilesystem[*fakeState](s)).To(Succeed())
	g.Expect(s.Common().Root).NotTo(BeNil())
}
