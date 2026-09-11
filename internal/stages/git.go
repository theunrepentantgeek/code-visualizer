package stages

import (
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// CheckGitRequirementHelper verifies the target is inside a git repository
// when any requested metric needs git. No-op otherwise.
func CheckGitRequirementHelper(targetPath string, requested []metric.Name) error {
	name, needsGit := findGitMetric(requested)
	if !needsGit {
		return nil
	}

	return verifyGitRepo(targetPath, &GitRequiredError{Metric: name})
}

// CheckGitRepoHelper verifies the target path is inside a git repository.
// Used by visualizations (such as spiral) that always require git.
func CheckGitRepoHelper(targetPath string) error {
	return verifyGitRepo(targetPath, &GitRequiredError{Metric: "spiral"})
}

func checkGitRepoForFeature(targetPath, feature string) error {
	return verifyGitRepo(targetPath, &GitRequiredError{Feature: feature})
}

func verifyGitRepo(targetPath string, requiredError *GitRequiredError) error {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return eris.Wrap(err, "failed to resolve absolute path")
	}

	isGit, err := scan.IsGitRepo(absPath)
	if err != nil {
		return eris.Wrap(err, "git check failed")
	}

	if !isGit {
		requiredError.Target = targetPath

		return requiredError
	}

	return nil
}

func findGitMetric(requested []metric.Name) (metric.Name, bool) {
	for _, name := range requested {
		if git.IsGitMetric(name) {
			return name, true
		}
	}

	return "", false
}

// CheckGitRequirement wraps CheckGitRequirementHelper.
func CheckGitRequirement(c *CommonState) error {
	if c.RepoRoot != "" {
		return nil
	}

	return CheckGitRequirementHelper(c.TargetPath, c.Requested.BaseMetrics)
}
