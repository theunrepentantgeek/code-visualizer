package scan

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

func TestFilterPolicyIncludesUsesRelativePath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root, err := filepath.Abs(filepath.Join("testdata", "with-dotfiles"))
	g.Expect(err).NotTo(HaveOccurred())

	policy := newFilterPolicy(root, []filter.Rule{{Pattern: ".*", Mode: filter.Exclude}})

	included, relPath, err := policy.includes(filepath.Join(root, ".hidden"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(relPath).To(Equal(".hidden"))
	g.Expect(included).To(BeFalse())
}
