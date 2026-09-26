package progress

import (
	"bytes"
	"io"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func TestDiagnosticWriter_BuffersPartialAndSplitsMultipleLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	var output bytes.Buffer
	reporter, err := New(Config{
		Mode:       ModePlain,
		Writer:     &output,
		IsTerminal: func(io.Writer) bool { return false },
		LookupEnv:  env(nil),
	})
	g.Expect(err).NotTo(HaveOccurred())
	writer := reporter.DiagnosticWriter()

	n, err := writer.Write([]byte("first"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(n).To(Equal(5))
	g.Expect(output.String()).To(BeEmpty())

	n, err = writer.Write([]byte(" line\nsecond\nthird"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(n).To(Equal(len(" line\nsecond\nthird")))
	g.Expect(output.String()).To(Equal("first line\nsecond\n"))

	g.Expect(reporter.Close()).To(Succeed())
	g.Expect(output.String()).To(Equal("first line\nsecond\nthird\n"))
}

func TestDiagnosticWriter_PausesWritesAndRedrawsActiveTTY(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter, backend := ttyTestReporter(t, true, false)

	g.Expect(reporter.Begin("Processing", 1)).To(Succeed())
	stage, err := reporter.StartStage("Work", StageLive, WorkCommits)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stage).NotTo(BeNil())

	_, err = reporter.DiagnosticWriter().Write([]byte("warning: retrying\n"))
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(backend.started).To(HaveLen(2))
	g.Expect(backend.started[0].stops).To(Equal(1))
	g.Expect(backend.started[1].kind).To(Equal("spinner"))
	g.Expect(strings.Join(backend.lines, "\n")).To(ContainSubstring("warning: retrying"))
}

func TestDiagnosticWriter_PlainModeRemainsAppendOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	var output bytes.Buffer
	reporter, err := New(Config{
		Mode:       ModePlain,
		Writer:     &output,
		IsTerminal: func(io.Writer) bool { return false },
		LookupEnv:  env(nil),
	})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(reporter.Begin("Processing", 1)).To(Succeed())
	stage, err := reporter.StartStage("Work", StageLive, WorkObservations)
	g.Expect(err).NotTo(HaveOccurred())
	_, err = reporter.DiagnosticWriter().Write([]byte("warning\n"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stage.Complete()).To(Succeed())

	g.Expect(output.String()).To(Equal(
		"Processing\n[1/1] Work: started\nwarning\n[1/1] Work: done (0.0s)\n",
	))
	g.Expect(output.String()).NotTo(ContainSubstring("\x1b"))
}
