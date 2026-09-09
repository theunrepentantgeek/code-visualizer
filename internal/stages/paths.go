package stages

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
)

// ValidatePathsHelper validates the target directory and output file paths.
// Returns *TargetPathError or *OutputPathError on failure.
func ValidatePathsHelper(targetPath, output string) error {
	if err := validateOutputPath(output); err != nil {
		return err
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &TargetPathError{Msg: "target path does not exist: " + targetPath}
		}

		return &TargetPathError{Msg: fmt.Sprintf("cannot access target path: %s", err)}
	}

	if !info.IsDir() {
		return &TargetPathError{Msg: "target path is not a directory: " + targetPath}
	}

	return nil
}

func validateOutputPath(output string) error {
	if _, err := canvas.FormatFromPath(output); err != nil {
		return &OutputPathError{Msg: err.Error()}
	}

	outDir := filepath.Dir(output)
	if outDir == "." {
		return nil
	}

	info, err := os.Stat(outDir)
	if err != nil {
		return &OutputPathError{Msg: "output directory does not exist: " + outDir}
	}

	if !info.IsDir() {
		return &OutputPathError{Msg: "output parent is not a directory: " + outDir}
	}

	return nil
}

// ValidatePaths validates c.TargetPath and c.Output.
func ValidatePaths(c *CommonState) error {
	var err error
	if c.Flags != nil && strings.TrimSpace(c.Flags.HistoryRange.Until) != "" {
		err = validateOutputPath(c.Output)
	} else {
		err = ValidatePathsHelper(c.TargetPath, c.Output)
	}

	if err != nil {
		return eris.Wrap(err, "invalid paths")
	}

	return nil
}
