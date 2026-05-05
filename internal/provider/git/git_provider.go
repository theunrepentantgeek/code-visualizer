package git

import (
	"errors"
	"log/slog"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

// gitProvider is a data-driven implementation of provider.Interface for all
// git-based metric providers. The seven individually identical provider files
// are replaced by a single table in providerDefs.
type gitProvider struct {
	name           metric.Name
	kind           metric.Kind
	description    string
	defaultPalette palette.PaletteName
	process        func(*repoService, *model.File, string)
	onFile         func()
}

func (p *gitProvider) Name() metric.Name                   { return p.name }
func (p *gitProvider) Kind() metric.Kind                   { return p.kind }
func (p *gitProvider) Description() string                 { return p.description }
func (_ *gitProvider) Dependencies() []metric.Name         { return nil }
func (p *gitProvider) DefaultPalette() palette.PaletteName { return p.defaultPalette }
func (p *gitProvider) SetOnFileProcessed(fn func())        { p.onFile = fn }

func (p *gitProvider) Load(root *model.Directory) error {
	return walkGitFiles(root, string(p.name), p.onFile, p.process)
}

// providerDef holds the static fields for one gitProvider.
type providerDef struct {
	name           metric.Name
	kind           metric.Kind
	description    string
	defaultPalette palette.PaletteName
	process        func(*repoService, *model.File, string)
}

// providerDefs is the authoritative list of all git metric providers.
// Adding a new git metric requires only a new entry here.
var providerDefs = []providerDef{
	{
		name:           FileAge,
		kind:           metric.Quantity,
		description:    "Time since first commit (days); older files score higher.",
		defaultPalette: palette.Temperature,
		process:        quantityProcess(FileAge, (*repoService).fileAge),
	},
	{
		name:           FileFreshness,
		kind:           metric.Quantity,
		description:    "Time since most recent commit (days); recently changed files score higher.",
		defaultPalette: palette.Temperature,
		process:        quantityProcess(FileFreshness, (*repoService).fileFreshness),
	},
	{
		name:           AuthorCount,
		kind:           metric.Quantity,
		description:    "Number of distinct commit authors; files touched by many people score higher.",
		defaultPalette: palette.GoodBad,
		process:        quantityProcess(AuthorCount, (*repoService).authorCount),
	},
	{
		name:           CommitCount,
		kind:           metric.Quantity,
		description:    "Number of commits that modified the file; frequently changed files score higher.",
		defaultPalette: palette.Temperature,
		process:        quantityProcess(CommitCount, (*repoService).commitCount),
	},
	{
		name:           TotalLinesAdded,
		kind:           metric.Quantity,
		description:    "Lines added over all commits, excluding the initial commit; high-churn files score higher.",
		defaultPalette: palette.Temperature,
		process:        quantityProcess(TotalLinesAdded, (*repoService).totalLinesAdded),
	},
	{
		name:           TotalLinesRemoved,
		kind:           metric.Quantity,
		description:    "Accumulated lines removed over all commits; high churn files score higher.",
		defaultPalette: palette.Temperature,
		process:        quantityProcess(TotalLinesRemoved, (*repoService).totalLinesRemoved),
	},
	{
		name:           CommitDensity,
		kind:           metric.Measure,
		description:    "Commits per month of file lifetime; frequently changed files score higher.",
		defaultPalette: palette.Temperature,
		process:        measureProcess(CommitDensity, (*repoService).commitDensity),
	},
}

// newProvider creates a fresh gitProvider instance for the given metric name.
// Panics if name is not a recognised git metric (programmer error).
func newProvider(name metric.Name) *gitProvider {
	for i := range providerDefs {
		if providerDefs[i].name == name {
			d := &providerDefs[i]

			return &gitProvider{
				name:           d.name,
				kind:           d.kind,
				description:    d.description,
				defaultPalette: d.defaultPalette,
				process:        d.process,
			}
		}
	}

	panic("newProvider: unknown git metric name: " + string(name))
}

// quantityProcess returns a walkGitFiles callback that computes an int64
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

// measureProcess returns a walkGitFiles callback that computes a float64
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
