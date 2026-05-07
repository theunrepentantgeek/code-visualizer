package provider

import (
"errors"
"sync"
"sync/atomic"
"testing"
"time"

. "github.com/onsi/gomega"

"github.com/bevan/code-visualizer/internal/metric"
"github.com/bevan/code-visualizer/internal/model"
)

// orderTracker records which providers ran and in what order.
type orderTracker struct {
mu    sync.Mutex
calls []metric.Name
}

func (o *orderTracker) record(name metric.Name) {
o.mu.Lock()
defer o.mu.Unlock()

o.calls = append(o.calls, name)
}

// mockLoader records load calls and optionally returns an error.
type mockLoader struct {
name    metric.Name
loadErr error
tracker *orderTracker
}

func (m *mockLoader) Load(_ *model.Directory) error {
if m.tracker != nil {
m.tracker.record(m.name)
}

return m.loadErr
}

func mockDesc(name metric.Name, deps ...metric.Name) MetricDescriptor {
return MetricDescriptor{
Name:         name,
Kind:         metric.Quantity,
Dependencies: deps,
}
}

func TestRunBasicExecution(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

reg := newRegistry()
tracker := &orderTracker{}
reg.register(mockDesc("m1"), &mockLoader{name: "m1", tracker: tracker})

err := runWithRegistry(reg, nil, []metric.Name{"m1"}, nil)
g.Expect(err).NotTo(HaveOccurred())
g.Expect(tracker.calls).To(Equal([]metric.Name{"m1"}))
}

func TestRunTransitiveDependencies(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

reg := newRegistry()
tracker := &orderTracker{}
reg.register(mockDesc("base"), &mockLoader{name: "base", tracker: tracker})
reg.register(mockDesc("derived", "base"), &mockLoader{name: "derived", tracker: tracker})

err := runWithRegistry(reg, nil, []metric.Name{"derived"}, nil)
g.Expect(err).NotTo(HaveOccurred())

// "base" must run before "derived"
g.Expect(tracker.calls).To(HaveLen(2))

baseIdx := -1
derivedIdx := -1

for i, n := range tracker.calls {
if n == "base" {
baseIdx = i
}

if n == "derived" {
derivedIdx = i
}
}

g.Expect(baseIdx).To(BeNumerically("<", derivedIdx))
}

func TestRunCycleDetection(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

reg := newRegistry()
reg.register(mockDesc("a", "b"), &stubLoader{})
reg.register(mockDesc("b", "a"), &stubLoader{})

err := runWithRegistry(reg, nil, []metric.Name{"a"}, nil)
g.Expect(err).To(HaveOccurred())
g.Expect(err).To(MatchError(ContainSubstring("circular dependency")))
}

func TestRunUnknownDependency(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

reg := newRegistry()
reg.register(mockDesc("a", "missing"), &stubLoader{})

err := runWithRegistry(reg, nil, []metric.Name{"a"}, nil)
g.Expect(err).To(HaveOccurred())
g.Expect(err).To(MatchError(ContainSubstring("unknown metric")))
}

func TestRunUnknownRequestedMetric(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

reg := newRegistry()

err := runWithRegistry(reg, nil, []metric.Name{"nonexistent"}, nil)
g.Expect(err).To(HaveOccurred())
g.Expect(err).To(MatchError(ContainSubstring("unknown metric")))
}

func TestRunUnknownMetricListsAvailable(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

reg := newRegistry()
reg.register(mockDesc("alpha"), &stubLoader{})
reg.register(mockDesc("beta"), &stubLoader{})

err := runWithRegistry(reg, nil, []metric.Name{"nonexistent"}, nil)
g.Expect(err).To(HaveOccurred())
g.Expect(err).To(MatchError(ContainSubstring("available metrics: alpha, beta")))
}

func TestRunErrorPropagation(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

reg := newRegistry()
reg.register(mockDesc("fail"), &mockLoader{name: "fail", loadErr: errors.New("load failed")})

err := runWithRegistry(reg, nil, []metric.Name{"fail"}, nil)
g.Expect(err).To(HaveOccurred())
g.Expect(err).To(MatchError(ContainSubstring("load failed")))
}

// concurrentLoader tracks concurrent Load calls via shared atomics.
type concurrentLoader struct {
counter       *atomic.Int32
maxConcurrent *atomic.Int32
}

func (c *concurrentLoader) Load(_ *model.Directory) error {
cur := c.counter.Add(1)

for {
old := c.maxConcurrent.Load()
if cur <= old || c.maxConcurrent.CompareAndSwap(old, cur) {
break
}
}

// Small sleep to keep goroutines alive long enough to overlap
time.Sleep(10 * time.Millisecond)
c.counter.Add(-1)

return nil
}

func TestRunParallelExecution(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

reg := newRegistry()

var (
counter       atomic.Int32
maxConcurrent atomic.Int32
)

// Register 3 independent providers — they should run concurrently
newCL := func() *concurrentLoader { return &concurrentLoader{counter: &counter, maxConcurrent: &maxConcurrent} }
reg.register(mockDesc("p1"), newCL())
reg.register(mockDesc("p2"), newCL())
reg.register(mockDesc("p3"), newCL())

err := runWithRegistry(reg, nil, []metric.Name{"p1", "p2", "p3"}, nil)
g.Expect(err).NotTo(HaveOccurred())
g.Expect(maxConcurrent.Load()).To(BeNumerically(">", 1), "expected concurrent execution")
}

func TestRunEmptyRequest(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

reg := newRegistry()

err := runWithRegistry(reg, nil, []metric.Name{}, nil)
g.Expect(err).NotTo(HaveOccurred())
}

func TestRunAutoExpandsDependencies(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

reg := newRegistry()
tracker := &orderTracker{}
reg.register(mockDesc("base"), &mockLoader{name: "base", tracker: tracker})
reg.register(mockDesc("mid", "base"), &mockLoader{name: "mid", tracker: tracker})
reg.register(mockDesc("top", "mid"), &mockLoader{name: "top", tracker: tracker})

// Only request "top" — "mid" and "base" should be auto-included
err := runWithRegistry(reg, nil, []metric.Name{"top"}, nil)
g.Expect(err).NotTo(HaveOccurred())
g.Expect(tracker.calls).To(HaveLen(3))
}
