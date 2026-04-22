package scan

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func TestIsGitRepo_InsideGitRepo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// The current working directory is the root of the code-visualizer git repo.
	wd, err := os.Getwd()
	g.Expect(err).NotTo(HaveOccurred())

	ok, err := IsGitRepo(wd)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ok).To(BeTrue())
}

func TestIsGitRepo_OutsideGitRepo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	ok, err := IsGitRepo(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ok).To(BeFalse())
}
