package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/render"
	"github.com/bevan/code-visualizer/internal/scan"
	"github.com/bevan/code-visualizer/internal/treemap"
)

type CLI struct {
	TargetPath string            `arg:"" help:"Path to directory to scan."`
	Output     string            `help:"Output PNG file path." short:"o" required:""`
	Size       metric.MetricName `help:"Metric for rectangle area (file-size, file-lines, file-age, file-freshness, author-count)." short:"s" required:"" enum:"file-size,file-lines,file-age,file-freshness,author-count"`
	Verbose    bool              `help:"Enable debug-level logging." short:"v"`
	Format     string            `help:"Diagnostic/error output format (text, json)." enum:"text,json" default:"text"`
	Width      int               `help:"Image width in pixels." default:"1920"`
	Height     int               `help:"Image height in pixels." default:"1080"`
}

func (c *CLI) Validate() error {
	// Check target path exists and is a directory
	info, err := os.Stat(c.TargetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target path does not exist: %s", c.TargetPath)
		}
		return fmt.Errorf("cannot access target path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target path is not a directory: %s", c.TargetPath)
	}

	// Check output parent directory exists
	outDir := filepath.Dir(c.Output)
	if outDir != "." {
		info, err = os.Stat(outDir)
		if err != nil {
			return fmt.Errorf("output directory does not exist: %s", outDir)
		}
		if !info.IsDir() {
			return fmt.Errorf("output parent is not a directory: %s", outDir)
		}
	}

	// Size metric must be numeric (already enforced by enum, but belt-and-suspenders)
	if !c.Size.IsNumeric() {
		return fmt.Errorf("size metric must be numeric, got %q", c.Size)
	}

	return nil
}

func (c *CLI) Run() error {
	setupLogger(c.Verbose)

	slog.Debug("scanning directory", "path", c.TargetPath)

	root, err := scan.Scan(c.TargetPath)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	files, dirs := countAll(root)

	slog.Debug("scan complete", "files", files, "directories", dirs)

	rects := treemap.Layout(root, c.Width, c.Height)

	slog.Debug("rendering", "width", c.Width, "height", c.Height, "output", c.Output)

	if err := render.RenderPNG(rects, c.Width, c.Height, c.Output); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	// Success output
	if c.Format == "json" {
		out := map[string]any{
			"files":          files,
			"directories":    dirs,
			"output":         c.Output,
			"width":          c.Width,
			"height":         c.Height,
			"size_metric":    string(c.Size),
			"fill_metric":    string(c.Size),
			"fill_palette":   nil,
			"border_metric":  nil,
			"border_palette": nil,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("Rendered treemap: %d files, %d directories → %s (%d×%d)\n",
		files, dirs, c.Output, c.Width, c.Height)

	return nil
}

func setupLogger(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

func countAll(node scan.DirectoryNode) (files int, dirs int) {
	files = len(node.Files)
	for _, d := range node.Dirs {
		dirs++
		f, d2 := countAll(d)
		files += f
		dirs += d2
	}
	return
}

func main() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("codeviz"),
		kong.Description("Generate treemap visualizations of file trees."),
		kong.UsageOnError(),
	)

	err := ctx.Run(&cli)
	if err != nil {
		switch {
		case isValidationError(err):
			exitWithError(cli.Format, err, 1)
		case isPathError(err):
			exitWithError(cli.Format, err, 2)
		case isOutputError(err):
			exitWithError(cli.Format, err, 4)
		default:
			exitWithError(cli.Format, err, 5)
		}
	}
}

func exitWithError(format string, err error, code int) {
	if format == "json" {
		out := map[string]any{
			"error": err.Error(),
			"code":  code,
		}
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(out)
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	}
	os.Exit(code)
}

func isValidationError(err error) bool {
	// Validation errors from Kong or our Validate()
	return false
}

func isPathError(err error) bool {
	return false
}

func isOutputError(err error) bool {
	return false
}
