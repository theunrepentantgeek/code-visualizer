package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
)

type boundaryReporter struct {
	events        []string
	stageWorks    []progress.WorkKind
	stageTotals   []int64
	terminalErr   error
	beginErr      error
	startStageErr error
	finishErr     error
}

func (r *boundaryReporter) Begin(title string, stageCount int) error {
	r.events = append(r.events, fmt.Sprintf("begin:%s:%d", title, stageCount))

	return r.beginErr
}

func (r *boundaryReporter) StartStage(
	name string,
	_ progress.StageKind,
	work progress.WorkKind,
) (progress.Stage, error) {
	r.events = append(r.events, "start:"+name)
	r.stageWorks = append(r.stageWorks, work)

	return &boundaryStage{reporter: r, work: work, name: name}, r.startStageErr
}

func (r *boundaryReporter) Finish() error {
	r.events = append(r.events, "finish")

	return r.finishErr
}

func (*boundaryReporter) DiagnosticWriter() io.Writer { return io.Discard }
func (*boundaryReporter) Close() error                { return nil }

type boundaryStage struct {
	reporter *boundaryReporter
	work     progress.WorkKind
	name     string
	finished bool
}

func (s *boundaryStage) WorkKind() progress.WorkKind { return s.work }
func (s *boundaryStage) SetTotal(total int64) error {
	if err := s.checkActive(); err != nil {
		return err
	}

	s.reporter.stageTotals = append(s.reporter.stageTotals, total)

	return nil
}
func (s *boundaryStage) SetProgress(int64) error { return s.checkActive() }
func (s *boundaryStage) SetStatus(string) error  { return s.checkActive() }

func (s *boundaryStage) Complete() error {
	return s.finish("complete")
}

func (s *boundaryStage) Fail(error) error {
	return s.finish("fail")
}

func (s *boundaryStage) Cancel(error) error {
	return s.finish("cancel")
}

func (s *boundaryStage) finish(event string) error {
	if err := s.checkActive(); err != nil {
		return err
	}

	s.reporter.events = append(s.reporter.events, event+":"+s.name)
	s.finished = true

	return s.reporter.terminalErr
}

func (s *boundaryStage) checkActive() error {
	if s.finished {
		return errors.New("boundary stage is already finished")
	}

	return nil
}

func TestProgressBoundaries_StartsAndCompletesStages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter := &boundaryReporter{}
	state := pipeline.NewState()

	boundaries, err := newProgressBoundaries(&Flags{
		Context:  context.Background(),
		Reporter: reporter,
	}, state, "Build", 2)
	g.Expect(err).NotTo(HaveOccurred())

	err = boundaries.Start("Prepare", progress.StageSummary, progress.WorkNone)
	g.Expect(err).NotTo(HaveOccurred())

	seen := []progress.WorkKind{}

	pipeline.ApplyFuncX(state, func(sink progress.Sink) error {
		seen = append(seen, sink.WorkKind())

		return nil
	})

	g.Expect(boundaries.End()).To(Succeed())
	g.Expect(boundaries.Start("Acquire", progress.StageLive, progress.WorkObservations)).To(Succeed())

	pipeline.ApplyFuncX(state, func(sink progress.Sink) error {
		seen = append(seen, sink.WorkKind())

		return nil
	})

	g.Expect(boundaries.End()).To(Succeed())
	g.Expect(boundaries.Finish()).To(Succeed())
	g.Expect(seen).To(Equal([]progress.WorkKind{progress.WorkNone, progress.WorkObservations}))
	g.Expect(reporter.events).To(Equal([]string{
		"begin:Build:2",
		"start:Prepare",
		"complete:Prepare",
		"start:Acquire",
		"complete:Acquire",
		"finish",
	}))
}

func TestRunDeterminateBoundarySetsTotalBeforeWork(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter := &boundaryReporter{}
	state := pipeline.NewState()

	boundaries, err := newProgressBoundaries(&Flags{Reporter: reporter}, state, "Build", 1)
	g.Expect(err).NotTo(HaveOccurred())

	totalWasSet := false
	err = runDeterminateBoundary(
		boundaries,
		"Loading filesystem metrics",
		progress.WorkObservations,
		12,
		func() {
			totalWasSet = len(reporter.stageTotals) == 1 && reporter.stageTotals[0] == 12
		},
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(totalWasSet).To(BeTrue())
}

func TestProgressBoundaries_EndsFailedStageAfterPipelineError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter := &boundaryReporter{}
	state := pipeline.NewState()

	boundaries, err := newProgressBoundaries(&Flags{Reporter: reporter}, state, "Build", 1)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(boundaries.Start("Acquire", progress.StageLive, progress.WorkObservations)).To(Succeed())

	processingErr := errors.New("processing failed")

	pipeline.ApplyFuncX(state, func(progress.Sink) error {
		return processingErr
	})

	err = boundaries.End()

	g.Expect(err).To(MatchError(ContainSubstring("processing failed")))
	g.Expect(errors.Is(err, processingErr)).To(BeTrue())
	g.Expect(reporter.events).To(Equal([]string{
		"begin:Build:1",
		"start:Acquire",
		"fail:Acquire",
	}))
}

func TestProgressBoundaries_CancelsWhenContextEndsAtBoundary(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter := &boundaryReporter{}
	ctx, cancel := context.WithCancel(context.Background())
	state := pipeline.NewState()

	boundaries, err := newProgressBoundaries(&Flags{Context: ctx, Reporter: reporter}, state, "Build", 1)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(boundaries.Start("Acquire", progress.StageLive, progress.WorkObservations)).To(Succeed())

	cancel()

	err = boundaries.End()

	g.Expect(err).To(MatchError(context.Canceled))
	g.Expect(reporter.events).To(Equal([]string{
		"begin:Build:1",
		"start:Acquire",
		"cancel:Acquire",
	}))
}

func TestProgressBoundaries_DoesNotStartNextStageAfterCancellation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter := &boundaryReporter{}
	ctx, cancel := context.WithCancel(context.Background())
	state := pipeline.NewState()

	boundaries, err := newProgressBoundaries(&Flags{Context: ctx, Reporter: reporter}, state, "Build", 2)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(boundaries.Start("Prepare", progress.StageSummary, progress.WorkNone)).To(Succeed())
	g.Expect(boundaries.End()).To(Succeed())

	cancel()

	err = boundaries.Start("Acquire", progress.StageLive, progress.WorkObservations)

	g.Expect(err).To(MatchError(ContainSubstring("workflow cancelled before phase")))
	g.Expect(errors.Is(err, context.Canceled)).To(BeTrue())
	g.Expect(reporter.events).To(Equal([]string{
		"begin:Build:2",
		"start:Prepare",
		"complete:Prepare",
	}))
}

func TestProgressBoundaries_ReplacesFinishedSinkWithInactiveSink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter := &boundaryReporter{}
	state := pipeline.NewState()

	boundaries, err := newProgressBoundaries(&Flags{Reporter: reporter}, state, "Build", 1)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(boundaries.Start("Acquire", progress.StageLive, progress.WorkObservations)).To(Succeed())

	retainedStage, applyErr := pipelineValue[progress.Sink](state)
	g.Expect(applyErr).NotTo(HaveOccurred())
	g.Expect(boundaries.End()).To(Succeed())

	g.Expect(retainedStage.SetStatus("late")).To(MatchError("boundary stage is already finished"))

	var (
		sinkWork progress.WorkKind
		sinkErr  error
	)

	pipeline.ApplyFuncX(state, func(sink progress.Sink) error {
		sinkWork = sink.WorkKind()
		sinkErr = sink.SetStatus("late")

		return nil
	})

	g.Expect(sinkWork).To(Equal(progress.WorkNone))
	g.Expect(sinkErr).To(MatchError("inactive progress stage"))
}

func TestProgressBoundaries_JoinsProcessingAndTerminalErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	processingErr := errors.New("processing failed")
	terminalErr := errors.New("reporter failed")
	reporter := &boundaryReporter{terminalErr: terminalErr}
	state := pipeline.NewState()

	boundaries, err := newProgressBoundaries(&Flags{Reporter: reporter}, state, "Build", 1)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(boundaries.Start("Acquire", progress.StageLive, progress.WorkObservations)).To(Succeed())

	pipeline.ApplyFuncX(state, func(progress.Sink) error {
		return processingErr
	})

	err = boundaries.End()

	g.Expect(err).To(MatchError(And(
		ContainSubstring("processing failed"),
		ContainSubstring("reporter failed"),
	)))
	g.Expect(errors.Is(err, processingErr)).To(BeTrue())
	g.Expect(errors.Is(err, terminalErr)).To(BeTrue())
}

func TestProgressBoundaries_RejectsNilReceiver(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var boundaries *progressBoundaries

	g.Expect(boundaries.Start("Acquire", progress.StageLive, progress.WorkObservations)).
		To(MatchError("progress boundaries are not initialized"))
	g.Expect(boundaries.End()).To(MatchError("progress boundaries are not initialized"))
	g.Expect(boundaries.Finish()).To(MatchError("progress boundaries are not initialized"))
}

func pipelineValue[T any](state *pipeline.State) (T, error) {
	var (
		value T
		err   error
	)

	pipeline.ApplyFuncX(state, func(v T) error {
		value = v

		return nil
	})

	if state.Err() != nil {
		err = eris.Wrap(state.Err(), "read pipeline value")
	}

	return value, err
}
