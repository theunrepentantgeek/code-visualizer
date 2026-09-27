package main

import (
	"context"
	"errors"
	"io"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
)

const (
	phasePreparing = "Preparing"
	phaseAcquiring = "Acquiring data"
	phaseRendering = "Rendering"
	phaseWriting   = "Writing output"
)

var errInactiveProgressStage = errors.New("inactive progress stage")

type progressBoundaries struct {
	ctx      context.Context
	reporter progress.Reporter
	state    *pipeline.State
	active   progress.Stage
}

func newProgressBoundaries(
	flags *Flags,
	state *pipeline.State,
	title string,
	stageCount int,
) (*progressBoundaries, error) {
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

func (b *progressBoundaries) End() error {
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
	return eris.Wrap(b.reporter.Finish(), "finish workflow progress")
}

type inactiveProgressSink struct{}

func (inactiveProgressSink) WorkKind() progress.WorkKind { return progress.WorkNone }
func (inactiveProgressSink) SetTotal(int64) error        { return errInactiveProgressStage }
func (inactiveProgressSink) SetProgress(int64) error     { return errInactiveProgressStage }
func (inactiveProgressSink) SetStatus(string) error      { return errInactiveProgressStage }
