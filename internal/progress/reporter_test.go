package progress

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

type recordingRenderer struct {
	events []event
	writer io.Writer
	closed int
}

func (r *recordingRenderer) render(e event) error {
	r.events = append(r.events, e)

	return nil
}

func (*recordingRenderer) writeDiagnostic(string) error { return nil }

func (r *recordingRenderer) close() error {
	r.closed++

	return nil
}

func fixedClock(times ...time.Time) func() time.Time {
	index := 0

	return func() time.Time {
		value := times[min(index, len(times)-1)]
		index++

		return value
	}
}

func TestReporter_CompletesNormalLifecycleAndFinalProgress(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	start := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	renderer := &recordingRenderer{writer: io.Discard}
	reporter := newReporter(Config{
		Writer:  io.Discard,
		Verbose: true,
		Now:     fixedClock(start, start.Add(2*time.Second)),
	}, renderer)

	g.Expect(reporter.Begin("Processing project", 1)).To(Succeed())
	stage, err := reporter.StartStage("Acquiring data", StageLive, WorkObservations)
	g.Expect(err).NotTo(HaveOccurred())

	if stage == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(stage.WorkKind()).To(Equal(WorkObservations))
	g.Expect(stage.SetTotal(4)).To(Succeed())
	g.Expect(stage.SetProgress(3)).To(Succeed())
	g.Expect(stage.SetStatus("three files")).To(Succeed())
	g.Expect(stage.Complete()).To(Succeed())
	g.Expect(reporter.Finish()).To(Succeed())
	g.Expect(reporter.Close()).To(Succeed())
	g.Expect(reporter.Close()).To(Succeed())

	g.Expect(renderer.closed).To(Equal(1))
	g.Expect(renderer.events).To(HaveLen(8))
	g.Expect(renderer.events[5].kind).To(Equal(eventProgress))
	g.Expect(renderer.events[5].current).To(Equal(int64(4)))
	g.Expect(renderer.events[6].kind).To(Equal(eventStageSucceeded))
	g.Expect(renderer.events[6].elapsed).To(Equal(2 * time.Second))
	g.Expect(renderer.events[7].kind).To(Equal(eventFinished))
}

func TestReporter_RejectsInvalidLifecycleTransitions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter := newReporter(Config{Writer: io.Discard}, &recordingRenderer{writer: io.Discard})

	_, err := reporter.StartStage("early", StageSummary, WorkNone)
	g.Expect(err).To(MatchError(ContainSubstring("begin")))
	g.Expect(reporter.Begin("Processing", 2)).To(Succeed())
	g.Expect(reporter.Begin("Again", 2)).To(MatchError(ContainSubstring("already begun")))

	first, err := reporter.StartStage("first", StageLive, WorkObservations)
	g.Expect(err).NotTo(HaveOccurred())

	if first == nil {
		panic("StartStage returned a nil stage without an error")
	}

	_, err = reporter.StartStage("overlap", StageSummary, WorkNone)
	g.Expect(err).To(MatchError(ContainSubstring("active")))
	g.Expect(reporter.Finish()).To(MatchError(ContainSubstring("active")))

	g.Expect(first.SetTotal(0)).To(MatchError(ContainSubstring("positive")))
	g.Expect(first.SetTotal(10)).To(Succeed())
	g.Expect(first.SetTotal(10)).To(MatchError(ContainSubstring("already set")))
	g.Expect(first.SetProgress(-1)).To(MatchError(ContainSubstring("negative")))
	g.Expect(first.SetProgress(5)).To(Succeed())
	g.Expect(first.SetProgress(4)).To(MatchError(ContainSubstring("decrease")))
	g.Expect(first.SetProgress(11)).To(MatchError(ContainSubstring("total")))
	g.Expect(first.Complete()).To(Succeed())
	g.Expect(first.Complete()).To(MatchError(ContainSubstring("finished")))
	g.Expect(first.SetStatus("late")).To(MatchError(ContainSubstring("finished")))

	second, err := reporter.StartStage("second", StageSummary, WorkNone)
	g.Expect(err).NotTo(HaveOccurred())

	if second == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(second.SetTotal(1)).To(MatchError(ContainSubstring("summary")))
	g.Expect(second.Fail(errors.New("boom"))).To(Succeed())
	g.Expect(second.Cancel(context.Canceled)).To(MatchError(ContainSubstring("finished")))
	g.Expect(reporter.Finish()).To(Succeed())
	g.Expect(reporter.Finish()).To(MatchError(ContainSubstring("already finished")))
}

func TestReporter_RequiresEveryPlannedStageBeforeFinish(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter := newReporter(Config{Writer: io.Discard}, &recordingRenderer{writer: io.Discard})

	g.Expect(reporter.Begin("Processing", 2)).To(Succeed())
	stage, err := reporter.StartStage("first", StageSummary, WorkNone)
	g.Expect(err).NotTo(HaveOccurred())

	if stage == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(stage.Complete()).To(Succeed())
	g.Expect(reporter.Finish()).To(MatchError(ContainSubstring("1 of 2")))
}

func TestReporter_RendersFailureAndCancellation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	renderer := &recordingRenderer{writer: io.Discard}
	reporter := newReporter(Config{Writer: io.Discard}, renderer)

	g.Expect(reporter.Begin("Processing", 2)).To(Succeed())
	failed, err := reporter.StartStage("first", StageSummary, WorkNone)
	g.Expect(err).NotTo(HaveOccurred())

	if failed == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(failed.Fail(errors.New("boom"))).To(Succeed())

	cancelled, err := reporter.StartStage("second", StageSummary, WorkNone)
	g.Expect(err).NotTo(HaveOccurred())

	if cancelled == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(cancelled.Cancel(context.Canceled)).To(Succeed())

	kinds := make([]eventKind, 0, len(renderer.events))
	for _, e := range renderer.events {
		kinds = append(kinds, e.kind)
	}

	g.Expect(kinds).To(ContainElements(eventStageFailed, eventStageCancelled))
}

func TestReporter_QuietModeValidatesWithoutWriting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var output bytes.Buffer

	reporter, err := New(Config{Writer: &output, Quiet: true})
	g.Expect(err).NotTo(HaveOccurred())

	if reporter == nil {
		panic("progress.New returned a nil reporter without an error")
	}

	g.Expect(reporter.Begin("Processing", 1)).To(Succeed())
	stage, err := reporter.StartStage("work", StageLive, WorkObservations)
	g.Expect(err).NotTo(HaveOccurred())

	if stage == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(stage.SetTotal(1)).To(Succeed())
	g.Expect(stage.SetProgress(1)).To(Succeed())
	g.Expect(stage.Complete()).To(Succeed())
	g.Expect(reporter.Finish()).To(Succeed())
	g.Expect(reporter.Close()).To(Succeed())
	g.Expect(output.String()).To(BeEmpty())
}

func TestReporter_CloseDoesNotSynthesizeSuccess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	renderer := &recordingRenderer{writer: io.Discard}
	reporter := newReporter(Config{Writer: io.Discard}, renderer)

	g.Expect(reporter.Begin("Processing", 1)).To(Succeed())
	_, err := reporter.StartStage("work", StageLive, WorkObservations)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(reporter.Close()).To(Succeed())

	for _, e := range renderer.events {
		g.Expect(e.kind).NotTo(Equal(eventStageSucceeded))
		g.Expect(e.kind).NotTo(Equal(eventFinished))
	}
}
