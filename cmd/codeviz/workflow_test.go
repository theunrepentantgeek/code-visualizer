package main

import (
	"context"
	"errors"
	"io"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
)

type workflowReporter struct {
	events    []string
	failError error
	closeErr  error
}

func (r *workflowReporter) Begin(title string, count int) error {
	r.events = append(r.events, title)
	return nil
}
func (r *workflowReporter) StartStage(name string, _ progress.StageKind, work progress.WorkKind) (progress.Stage, error) {
	r.events = append(r.events, name)
	return &workflowStage{reporter: r, work: work}, nil
}
func (r *workflowReporter) Finish() error {
	r.events = append(r.events, "finish")
	return nil
}
func (*workflowReporter) DiagnosticWriter() io.Writer { return io.Discard }
func (r *workflowReporter) Close() error {
	r.events = append(r.events, "close")
	return r.closeErr
}

type workflowStage struct {
	reporter *workflowReporter
	work     progress.WorkKind
}

func (s *workflowStage) WorkKind() progress.WorkKind { return s.work }
func (*workflowStage) SetTotal(int64) error          { return nil }
func (*workflowStage) SetProgress(int64) error       { return nil }
func (*workflowStage) SetStatus(string) error        { return nil }
func (s *workflowStage) Complete() error {
	s.reporter.events = append(s.reporter.events, "complete")
	return nil
}
func (s *workflowStage) Fail(error) error {
	s.reporter.events = append(s.reporter.events, "fail")
	return s.reporter.failError
}
func (s *workflowStage) Cancel(error) error {
	s.reporter.events = append(s.reporter.events, "cancel")
	return nil
}

func TestRunWorkflowOrdersPhasesAndReplacesTypedSink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter := &workflowReporter{}
	state := pipeline.NewState()
	seen := []progress.WorkKind{}
	phases := []workflowPhase{
		{Name: "Prepare", Kind: progress.StageSummary, Run: func(state *pipeline.State) {
			pipeline.ApplyFuncX(state, func(sink progress.Sink) error {
				seen = append(seen, sink.WorkKind())
				return nil
			})
		}},
		{Name: "Acquire", Kind: progress.StageLive, Work: progress.WorkObservations, Run: func(state *pipeline.State) {
			pipeline.ApplyFuncX(state, func(sink progress.Sink) error {
				seen = append(seen, sink.WorkKind())
				return nil
			})
		}},
	}

	err := runWorkflow(context.Background(), reporter, state, "Build", phases)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(seen).To(Equal([]progress.WorkKind{progress.WorkNone, progress.WorkObservations}))
	g.Expect(reporter.events).To(Equal([]string{
		"Build", "Prepare", "complete", "Acquire", "complete", "finish", "close",
	}))
}

func TestRunWorkflowCancelsActivePhase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter := &workflowReporter{}
	ctx, cancel := context.WithCancel(context.Background())
	state := pipeline.NewState()

	err := runWorkflow(ctx, reporter, state, "Build", []workflowPhase{{
		Name: "Acquire",
		Kind: progress.StageLive,
		Run: func(*pipeline.State) {
			cancel()
		},
	}})

	g.Expect(err).To(MatchError(context.Canceled))
	g.Expect(reporter.events).To(Equal([]string{"Build", "Acquire", "cancel", "close"}))
}

func TestRunWorkflowJoinsProcessingAndReporterErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	processingErr := errors.New("processing failed")
	reporterErr := errors.New("reporter failed")
	closeErr := errors.New("close failed")
	reporter := &workflowReporter{failError: reporterErr, closeErr: closeErr}
	state := pipeline.NewState()

	err := runWorkflow(context.Background(), reporter, state, "Build", []workflowPhase{{
		Name: "Acquire",
		Run: func(state *pipeline.State) {
			pipeline.ApplyFuncX(state, func(context.Context) error { return processingErr })
		},
	}})

	g.Expect(err).To(MatchError(And(
		ContainSubstring("processing failed"),
		ContainSubstring("reporter failed"),
		ContainSubstring("close failed"),
	)))
	g.Expect(errors.Is(err, processingErr)).To(BeTrue())
}
