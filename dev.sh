#!/usr/bin/env bash

set -eu

GIT_ROOT=$(git rev-parse --show-toplevel)
TOOL_DEST=$GIT_ROOT/tools
mkdir -p "$TOOL_DEST"

# This will be fast if everything is already installed
VERBOSE=true
$GIT_ROOT/.devcontainer/install-dependencies.sh --skip-installed

export PATH="$TOOL_DEST:$PATH"

echo "Entering $SHELL with expanded PATH (use 'exit' to quit)."
echo "Try running 'task -l' to see possible commands."
$SHELL
