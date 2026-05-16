package pipeline_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
)

type counter struct {
	n   int
	log []string
}

func incN(amount int) pipeline.Stage[*counter] {
	return func(c *counter) error {
		c.n += amount
		c.log = append(c.log, "inc")

		return nil
	}
}

func fail(msg string) pipeline.Stage[*counter] {
	return func(c *counter) error {
		c.log = append(c.log, "fail")

		return errors.New(msg)
	}
}

func TestRun_EmptyPipeline_ReturnsStateUnchanged(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &counter{n: 7}
	got, err := pipeline.Run(c)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(BeIdenticalTo(c))
	g.Expect(c.n).To(Equal(7))
	g.Expect(c.log).To(BeEmpty())
}

func TestRun_SingleStage_RunsOnce(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &counter{}
	_, err := pipeline.Run(c, incN(3))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(c.n).To(Equal(3))
	g.Expect(c.log).To(Equal([]string{"inc"}))
}

func TestRun_MultipleStages_RunInDeclarationOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &counter{}
	_, err := pipeline.Run(c, incN(1), incN(2), incN(4))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(c.n).To(Equal(7))
	g.Expect(c.log).To(Equal([]string{"inc", "inc", "inc"}))
}

func TestRun_ErrorHaltsExecution(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &counter{}
	_, err := pipeline.Run(c, incN(1), fail("boom"), incN(100))

	g.Expect(err).To(MatchError("boom"))
	g.Expect(c.n).To(Equal(1))
	g.Expect(c.log).To(Equal([]string{"inc", "fail"}))
}

func TestRun_PartialStateReturnedOnError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &counter{}
	got, err := pipeline.Run(c, incN(2), incN(3), fail("stop"))

	g.Expect(err).To(HaveOccurred())
	g.Expect(got).To(BeIdenticalTo(c))
	g.Expect(c.n).To(Equal(5))
}

func TestRun_ErrorReturnedUnwrapped(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sentinel := errors.New("sentinel")
	c := &counter{}
	_, err := pipeline.Run(c, func(*counter) error { return sentinel })

	g.Expect(err).To(BeIdenticalTo(sentinel))
}

func TestRun_NilStage_Panics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	defer func() {
		r := recover()
		g.Expect(r).NotTo(BeNil())
	}()

	c := &counter{}
	_, _ = pipeline.Run(c, nil)
}
