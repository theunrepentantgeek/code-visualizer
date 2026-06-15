package git

import (
	"errors"
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// providerDef holds the static fields and processing callback for one git metric.
type providerDef struct {
	kind           metric.Kind
	description    string
	defaultPalette palette.PaletteName
	process        func(*repoService, *model.File, string)
}

// providerDefs is the authoritative map of all git metric providers.
// Adding a new git metric requires only a new entry here.
var providerDefs = map[metric.Name]providerDef{
	FileAge: {
		kind:           metric.Quantity,
		description:    "Time since first commit (days); older files score higher.",
		defaultPalette: palette.Temperature,
		process:        quantityProcess(FileAge, (*repoService).fileAge),
	},
	FileFreshness: {
		kind:           metric.Quantity,
		description:    "Time since most recent commit (days); recently changed files score higher.",
		defaultPalette: palette.Temperature,
		process:        quantityProcess(FileFreshness, (*repoService).fileFreshness),
	},
	AuthorCount: {
		kind:           metric.Quantity,
		description:    "Number of distinct commit authors; files touched by many people score higher.",
		defaultPalette: palette.GoodBad,
		process:        quantityProcess(AuthorCount, (*repoService).authorCount),
	},
	CommitCount: {
		kind:           metric.Quantity,
		description:    "Number of commits that modified the file; frequently changed files score higher.",
		defaultPalette: palette.Temperature,
		process:        quantityProcess(CommitCount, (*repoService).commitCount),
	},
	TotalLinesAdded: {
		kind:           metric.Quantity,
		description:    "Lines added over all commits, excluding the initial commit; high-churn files score higher.",
		defaultPalette: palette.Temperature,
		process:        quantityProcess(TotalLinesAdded, (*repoService).totalLinesAdded),
	},
	TotalLinesRemoved: {
		kind:           metric.Quantity,
		description:    "Accumulated lines removed over all commits; high churn files score higher.",
		defaultPalette: palette.Temperature,
		process:        quantityProcess(TotalLinesRemoved, (*repoService).totalLinesRemoved),
	},
	CommitDensity: {
		kind:           metric.Measure,
		description:    "Commits per month of file lifetime; frequently changed files score higher.",
		defaultPalette: palette.Temperature,
		process:        measureProcess(CommitDensity, (*repoService).commitDensity),
	},
}

// quantityProcess returns a providerDef process callback that computes an int64
// metric and stores it via SetQuantity.
func quantityProcess(
	name metric.Name,
	fn func(*repoService, string) (int64, error),
) func(*repoService, *model.File, string) {
	return func(s *repoService, f *model.File, relPath string) {
		val, err := fn(s, relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get "+string(name), "path", relPath, "error", err)
			}

			return
		}

		f.SetQuantity(name, val)
	}
}

// measureProcess returns a providerDef process callback that computes a float64
// metric and stores it via SetMeasure.
func measureProcess(
	name metric.Name,
	fn func(*repoService, string) (float64, error),
) func(*repoService, *model.File, string) {
	return func(s *repoService, f *model.File, relPath string) {
		val, err := fn(s, relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get "+string(name), "path", relPath, "error", err)
			}

			return
		}

		f.SetMeasure(name, val)
	}
}
