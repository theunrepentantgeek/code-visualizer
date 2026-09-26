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

func runCommandWorkflow(
	flags *Flags,
	state *pipeline.State,
	title string,
	phases []workflowPhase,
) error {
	ctx := flags.Context
	if ctx == nil {
		ctx = context.Background()
	}

	reporter := flags.Reporter
	if reporter == nil {
		var err error

		reporter, err = progress.New(progress.Config{
			Mode:   progress.ModePlain,
			Writer: io.Discard,
			Quiet:  true,
		})
		if err != nil {
			return eris.Wrap(err, "initialize quiet progress reporter")
		}
	}

	return runWorkflow(ctx, reporter, state, title, phases)
}

type workflowPhase struct {
	Name string
	Kind progress.StageKind
	Work progress.WorkKind
	Run  func(*pipeline.State)
}

//nolint:revive // Lifecycle ordering and error precedence are intentionally centralized.
func runWorkflow(
	ctx context.Context,
	reporter progress.Reporter,
	state *pipeline.State,
	title string,
	phases []workflowPhase,
) (resultErr error) {
	defer func() {
		resultErr = errors.Join(resultErr, reporter.Close())
	}()

	pipeline.Set[context.Context](state, ctx)

	if err := reporter.Begin(title, len(phases)); err != nil {
		return eris.Wrap(err, "begin workflow progress")
	}

	for _, phase := range phases {
		if err := ctx.Err(); err != nil {
			return eris.Wrap(err, "workflow cancelled before phase")
		}

		stage, err := reporter.StartStage(phase.Name, phase.Kind, phase.Work)
		if err != nil {
			return eris.Wrapf(err, "start progress phase %q", phase.Name)
		}

		pipeline.Set[progress.Sink](state, stage)
		phase.Run(state)

		processingErr := state.Err()
		if processingErr == nil {
			processingErr = ctx.Err()
		}

		if processingErr != nil {
			var outcomeErr error
			if errors.Is(processingErr, context.Canceled) ||
				errors.Is(processingErr, context.DeadlineExceeded) {
				outcomeErr = stage.Cancel(processingErr)
			} else {
				outcomeErr = stage.Fail(processingErr)
			}

			return errors.Join(processingErr, outcomeErr)
		}

		if err := stage.Complete(); err != nil {
			return eris.Wrapf(err, "complete progress phase %q", phase.Name)
		}
	}

	return eris.Wrap(reporter.Finish(), "finish workflow progress")
}
