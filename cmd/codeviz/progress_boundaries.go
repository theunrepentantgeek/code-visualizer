package main

import (
	"context"
	"errors"
	"io"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

const (
	phasePreparing = "Preparing"
	phaseRendering = "Rendering"
	phaseWriting   = "Writing output"
)

var (
	errInactiveProgressStage = errors.New("inactive progress stage")
	errProgressBoundariesNil = errors.New("progress boundaries are not initialized")
)

type determinateReporter interface {
	StartDeterminateStage(name string, work progress.WorkKind, total int64) (progress.Stage, error)
}

type progressBoundaries struct {
	ctx      context.Context
	reporter progress.Reporter
	state    *pipeline.State
	active   progress.Stage
}

func runBoundary(
	boundaries *progressBoundaries,
	name string,
	kind progress.StageKind,
	work progress.WorkKind,
	run func(),
) error {
	if err := boundaries.Start(name, kind, work); err != nil {
		return err
	}

	run()

	return boundaries.End()
}

func runDeterminateBoundary(
	boundaries *progressBoundaries,
	name string,
	work progress.WorkKind,
	total int64,
	run func(),
) error {
	if err := boundaries.StartDeterminate(name, work, total); err != nil {
		return err
	}

	run()

	return boundaries.End()
}

//nolint:revive // enabled expresses whether this optional pipeline stage exists.
func runGitMetricsBoundary(
	boundaries *progressBoundaries,
	enabled bool,
	total int64,
	run func(),
) error {
	if !enabled {
		return nil
	}

	return runDeterminateBoundary(
		boundaries,
		"Loading Git metrics",
		progress.WorkCommits,
		total,
		run,
	)
}

//nolint:revive // needsGit determines whether the optional Git stage is counted.
func ordinaryStageCount(needsGit bool) int {
	if needsGit {
		return 6
	}

	return 5
}

func newProgressBoundaries(
	flags *Flags,
	state *pipeline.State,
	title string,
	stageCount int,
) (*progressBoundaries, error) {
	if state.Err() != nil {
		return nil, eris.Wrap(state.Err(), "prepare workflow progress")
	}

	ctx := context.Background()
	if flags != nil && flags.Context != nil {
		ctx = flags.Context
	}

	reporter := progress.Reporter(nil)
	if flags != nil {
		reporter = flags.Reporter
	}

	if reporter == nil {
		var err error

		reporter, err = progress.New(progress.Config{
			Mode:   progress.ModePlain,
			Writer: io.Discard,
			Quiet:  true,
		})
		if err != nil {
			return nil, eris.Wrap(err, "initialize quiet progress reporter")
		}
	}

	pipeline.Set[context.Context](state, ctx)

	if err := reporter.Begin(title, stageCount); err != nil {
		return nil, eris.Wrap(err, "begin workflow progress")
	}

	return &progressBoundaries{
		ctx:      ctx,
		reporter: reporter,
		state:    state,
	}, nil
}

func (b *progressBoundaries) Start(
	name string,
	kind progress.StageKind,
	work progress.WorkKind,
) error {
	if b == nil {
		return errProgressBoundariesNil
	}

	if b.active != nil {
		return errors.New("progress stage is already active")
	}

	if err := b.ctx.Err(); err != nil {
		return eris.Wrap(err, "workflow cancelled before phase")
	}

	stage, err := b.reporter.StartStage(name, kind, work)
	if err != nil {
		return eris.Wrapf(err, "start progress phase %q", name)
	}

	b.active = stage
	pipeline.Set[progress.Sink](b.state, stage)

	return nil
}

func (b *progressBoundaries) StartDeterminate(
	name string,
	work progress.WorkKind,
	total int64,
) error {
	if b == nil {
		return errProgressBoundariesNil
	}

	if b.active != nil {
		return errors.New("progress stage is already active")
	}

	if err := b.ctx.Err(); err != nil {
		return eris.Wrap(err, "workflow cancelled before phase")
	}

	reporter, ok := b.reporter.(determinateReporter)
	if !ok {
		if err := b.Start(name, progress.StageLive, work); err != nil {
			return err
		}

		if err := b.active.SetTotal(total); err != nil {
			return errors.Join(err, b.End())
		}

		return nil
	}

	stage, err := reporter.StartDeterminateStage(name, work, total)
	if err != nil {
		return eris.Wrapf(err, "start progress phase %q", name)
	}

	b.active = stage
	pipeline.Set[progress.Sink](b.state, stage)

	return nil
}

func (b *progressBoundaries) End() error {
	if b == nil {
		return errProgressBoundariesNil
	}

	if b.active == nil {
		return errors.New("progress stage is not active")
	}

	stage := b.active

	processingErr := b.state.Err()
	if processingErr == nil {
		processingErr = b.ctx.Err()
	}

	var outcomeErr error

	switch {
	case processingErr == nil:
		outcomeErr = stage.Complete()
	case errors.Is(processingErr, context.Canceled), errors.Is(processingErr, context.DeadlineExceeded):
		outcomeErr = stage.Cancel(processingErr)
	default:
		outcomeErr = stage.Fail(processingErr)
	}

	b.active = nil
	pipeline.Set[progress.Sink](b.state, inactiveProgressSink{})

	return errors.Join(processingErr, outcomeErr)
}

func (b *progressBoundaries) Finish() error {
	if b == nil {
		return errProgressBoundariesNil
	}

	return eris.Wrap(b.reporter.Finish(), "finish workflow progress")
}

type inactiveProgressSink struct{}

func (inactiveProgressSink) WorkKind() progress.WorkKind { return progress.WorkNone }
func (inactiveProgressSink) SetTotal(int64) error        { return errInactiveProgressStage }
func (inactiveProgressSink) SetProgress(int64) error     { return errInactiveProgressStage }
func (inactiveProgressSink) SetStatus(string) error      { return errInactiveProgressStage }

func determineGitMetricTotal(state *pipeline.State) int64 {
	var total int64

	pipeline.ApplyFuncXY(state, func(common *stages.CommonState, ctx context.Context) error {
		var err error

		total, err = stages.GitMetricTotal(ctx, common)

		return eris.Wrap(err, "determine Git metric total")
	})

	return total
}
