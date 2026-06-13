package provider

import (
	"sync"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

var findWithHintTestMu sync.Mutex

func TestFindWithHint_Found(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	findWithHintTestMu.Lock()
	defer findWithHintTestMu.Unlock()

	oldReg := globalRegistry
	defer func() { globalRegistry = oldReg }()

	globalRegistry = newRegistry()
	globalRegistry.register(&stubProvider{name: "file-size", kind: metric.Quantity, target: metric.File})

	p, err := FindWithHint("file-size", metric.File)
	g.Expect(err).ToNot(HaveOccurred())

	if p == nil {
		t.Fatal("expected provider")
	}

	g.Expect(p.Name()).To(Equal(metric.Name("file-size")))
}

func TestFindWithHint_WrongTarget(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	findWithHintTestMu.Lock()
	defer findWithHintTestMu.Unlock()

	oldReg := globalRegistry
	defer func() { globalRegistry = oldReg }()

	globalRegistry = newRegistry()
	globalRegistry.register(&stubProvider{name: "dir-count", kind: metric.Quantity, target: metric.Directory})

	_, err := FindWithHint("dir-count", metric.File)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("exists for target"))
	g.Expect(err.Error()).To(ContainSubstring("directory"))
}

func TestFindWithHint_NotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	findWithHintTestMu.Lock()
	defer findWithHintTestMu.Unlock()

	oldReg := globalRegistry
	defer func() { globalRegistry = oldReg }()

	globalRegistry = newRegistry()

	_, err := FindWithHint("nonexistent", metric.File)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown file metric"))
	g.Expect(err.Error()).ToNot(ContainSubstring("exists for target"))
}
