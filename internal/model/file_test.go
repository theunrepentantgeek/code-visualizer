package model

import (
	"sync"
	"testing"

	. "github.com/onsi/gomega"
)

func TestFileSetAndGetQuantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}

	f.SetQuantity("file-size", 1024)

	v, ok := f.Quantity("file-size")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(1024))
}

func TestFileSetAndGetMeasure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}

	f.SetMeasure("complexity", 3.14)

	v, ok := f.Measure("complexity")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(3.14))
}

func TestFileSetAndGetClassification(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}

	f.SetClassification("file-type", "go")

	v, ok := f.Classification("file-type")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal("go"))
}

func TestFileGetUnsetMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}

	_, ok := f.Quantity("unset")
	g.Expect(ok).To(BeFalse())

	_, ok = f.Measure("unset")
	g.Expect(ok).To(BeFalse())

	_, ok = f.Classification("unset")
	g.Expect(ok).To(BeFalse())
}

func TestFileConcurrentAccess(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}

	var wg sync.WaitGroup

	wg.Go(func() {
		for i := range 100 {
			f.SetQuantity("size", i)
		}
	})

	wg.Go(func() {
		for range 100 {
			f.SetClassification("type", "go")
		}
	})

	wg.Wait()

	v, ok := f.Quantity("size")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(99))

	s, ok := f.Classification("type")
	g.Expect(ok).To(BeTrue())
	g.Expect(s).To(Equal("go"))
}
