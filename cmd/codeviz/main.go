package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"

	"github.com/alecthomas/kong"
	"github.com/lmittmann/tint"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

type CLI struct {
	Quiet    bool          `help:"Suppress progress; show only warnings and errors." short:"q" xor:"verbosity"`
	Verbose  bool          `help:"Show detailed scanning and metric progress." short:"v" xor:"verbosity"`
	Debug    bool          `help:"Show per-directory scan progress (implies verbose output)." xor:"verbosity"`
	Config   string        `help:"Path to configuration file (.yaml, .yml, or .json)." name:"config" optional:""`
	Progress progress.Mode `help:"Progress output mode." default:"auto" enum:"auto,tty,plain"`
	NoColor  bool          `help:"Disable coloured output." name:"no-color"`

	//nolint:revive,nolintlint // Long help text is more important than minimizing line length, and annotations can't be wrapped
	ExportConfig string `help:"Write effective configuration to file (.yaml, .yml, or .json)." name:"export-config" optional:""`
	ExportData   string `help:"Write computed metrics to file (.json or .yaml/.yml)." name:"export-data" optional:""`

	TreeMap    TreemapCmd    `cmd:"" name:"tree-map"    help:"Generate a tree-map visualization."`
	RadialTree RadialCmd     `cmd:"" name:"radial-tree" help:"Generate a radial tree visualization."`
	DonutTree  DonutTreeCmd  `cmd:"" name:"donut-tree"  help:"Generate a hierarchical donut visualization."`
	BubbleTree BubbletreeCmd `cmd:"" name:"bubble-tree" help:"Generate a bubble tree visualization."`
	Spiral     SpiralCmd     `cmd:""                    help:"Generate a spiral timeline visualization."`
	Scatter    ScatterCmd    `cmd:""                    help:"Generate a scatter plot visualization."`
	Alluvial   AlluvialCmd   `cmd:""                    help:"Generate an alluvial release visualization."`
	Render     RenderCmd     `cmd:""                    help:"Render a preset visualization."`
	Help       HelpCmd       `cmd:""                    help:"Show this help message."`
}

// Flags bundles cross-cutting concerns that are passed to every command's Run method.
type Flags struct {
	Quiet        bool
	Verbose      bool
	Debug        bool
	ExportConfig string
	ExportData   string
	Config       *config.Config
	Context      context.Context
	Reporter     progress.Reporter
	Stdout       io.Writer
	configPath   string // path passed to --config, empty if not explicitly provided
}

type application struct {
	args       []string
	stdout     io.Writer
	stderr     io.Writer
	context    context.Context
	isTerminal func(io.Writer) bool
	lookupEnv  func(string) (string, bool)
}

// HasExplicitConfig reports whether --config was explicitly provided on the command line.
func (f *Flags) HasExplicitConfig() bool {
	return f.configPath != ""
}

func (f *Flags) stdoutWriter() io.Writer {
	if f.Stdout != nil {
		return f.Stdout
	}

	return os.Stdout
}

// toStagesFlags converts the cmd-local Flags struct into the stages-package form.
func toStagesFlags(f *Flags) *stages.Flags {
	return &stages.Flags{
		Quiet:        f.Quiet,
		Verbose:      f.Verbose,
		Debug:        f.Debug,
		ExportConfig: f.ExportConfig,
		ExportData:   f.ExportData,
		Config:       f.Config,
		ChangedOnly:  f.Config.ChangedOnlyEnabled(),
	}
}

func stagesFlagsForCommand(flags *Flags, fromValue, untilValue string) *stages.Flags {
	parsedFlags := toStagesFlags(flags)
	parsedFlags.HistoryRange = git.HistoryRange{
		From:  fromValue,
		Until: untilValue,
	}

	return parsedFlags
}

//nolint:revive // Boolean values map directly from independent CLI flags.
func setupLogger(writer io.Writer, debug, noColor bool) {
	level := slog.LevelWarn
	if debug {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(tint.NewTextHandler(writer, &tint.Options{
		Level:   level,
		NoColor: noColor,
	})))
}

func main() {
	filesystem.Register()
	git.Register()
	golang.Register()

	runContext, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := runApplication(application{
		args:       os.Args[1:],
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		context:    runContext,
		isTerminal: nil,
		lookupEnv:  os.LookupEnv,
	})

	stop()

	os.Exit(code)
}

//nolint:cyclop,funlen,revive // Bootstrap failures map directly to documented process exit codes.
func runApplication(app application) int {
	// Install tint early so bootstrap errors are formatted consistently.
	setupLogger(app.stderr, false, false)

	cli := CLI{}

	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		kong.Description("Generate visualizations of file trees."),
		filterMapperOption(),
		kong.Writers(app.stdout, app.stderr),
	)
	if err != nil {
		slog.Error("failed to initialize CLI", "error", err)

		return 5
	}

	ctx, err := parser.Parse(app.args)
	if err != nil {
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) && parseErr.Context != nil {
			_ = parseErr.Context.PrintUsage(false)
		}

		slog.Error("failed to parse arguments", "err", err)

		return 1
	}

	reporter, err := progress.New(progress.Config{
		Mode:       cli.Progress,
		Writer:     app.stderr,
		IsTerminal: app.isTerminal,
		LookupEnv:  app.lookupEnv,
		NoColor:    cli.NoColor,
		Quiet:      cli.Quiet,
		Verbose:    cli.Verbose || cli.Debug,
	})
	if err != nil {
		slog.Error("failed to initialize progress output", "error", err)

		return 5
	}

	noColor := cli.NoColor || environmentDisablesColor(app.lookupEnv)
	setupLogger(reporter.DiagnosticWriter(), cli.Debug, noColor)

	cfg := config.New()

	if cli.Config != "" {
		if loadErr := cfg.Load(cli.Config); loadErr != nil {
			slog.Error(loadErr.Error())

			return 5
		}
	}

	flags := &Flags{
		Quiet:        cli.Quiet,
		Verbose:      cli.Verbose,
		Debug:        cli.Debug,
		ExportConfig: cli.ExportConfig,
		ExportData:   cli.ExportData,
		Config:       cfg,
		Context:      app.context,
		Reporter:     reporter,
		Stdout:       app.stdout,
		configPath:   cli.Config,
	}

	err = ctx.Run(flags)
	if err != nil {
		code := classifyError(err)
		slog.Error("command failed", "err", err)

		return code
	}

	return 0
}

func environmentDisablesColor(lookupEnv func(string) (string, bool)) bool {
	for name, disabledValue := range map[string]string{
		"NO_COLOR":    "",
		"TERM":        "dumb",
		"FORCE_COLOR": "0",
	} {
		value, exists := lookupEnv(name)
		if exists && (disabledValue == "" || value == disabledValue) {
			return true
		}
	}

	return false
}

func classifyError(err error) int {
	var (
		gitErr     *stages.GitRequiredError
		targetErr  *stages.TargetPathError
		outputErr  *stages.OutputPathError
		noFilesErr *stages.NoFilesAfterFilterError
	)

	switch {
	case errors.Is(err, context.Canceled):
		return 130
	case errors.As(err, &targetErr):
		return 2
	case errors.As(err, &gitErr):
		return 3
	case errors.As(err, &outputErr):
		return 4
	case errors.As(err, &noFilesErr):
		return 6
	default:
		return 5
	}
}
