#!/usr/bin/env bash

set -eu

GIT_ROOT=$(git rev-parse --show-toplevel)
TOOL_DEST=$GIT_ROOT/tools
mkdir -p "$TOOL_DEST"

# Keep the lint cache outside the worktree so tests scanning the repo never see it change.
# Must match GOLANGCI_LINT_CACHE in Taskfile.yml.
export GOLANGCI_LINT_CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/codeviz/golangci-lint/$(printf %s "$GIT_ROOT" | cksum | cut -d' ' -f1)"

# This will be fast if everything is already installed
VERBOSE=true
$GIT_ROOT/.devcontainer/install-dependencies.sh --skip-installed

export PATH="$TOOL_DEST:$PATH"

if [[ ${1:-} == "-c" ]]; then
    "$SHELL" "$@"
    exit
fi

echo "Entering $SHELL with expanded PATH (use 'exit' to quit)."
echo "Try running 'task -l' to see possible commands."
$SHELL
