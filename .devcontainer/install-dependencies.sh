#!/bin/bash

# -e immediate exit on error
# -u treat unset variables as an error
# -o pipefail fail a pipeline if any command fails
set -euo pipefail

# This may be run in two modes:
#
# - When being used to set up a devcontainer.
#   In this mode the code is not checked out yet,
#   and we can install the tools globally.
#
# - When being used to install tools locally.
#   In this mode the code is already checked out,
#   and we do not want to pollute the user's system.
#
# To distinguish between these modes we will
# have the devcontainer script pass the argument
# `devcontainer`
#
# Other available arguments
#
# -v --verbose          : Generate more logging
# -s --skip-installed   : Skip anything that's already installed
#

VERBOSE=false
SKIP=false
DEVCONTAINER=false

while [[ $# -gt 0 ]]; do
  case $1 in
    -v | --verbose)
      VERBOSE=true
      shift
      ;;
    -s | --skip-installed)
      SKIP=true
      shift
      ;;
    -* | --*)
      echo "Unknown option $1"
      exit 1
      ;;
    devcontainer)
      DEVCONTAINER=true
      shift
      ;;
    *)
      echo "Unknown parameter $1"
      exit 1
      ;;
  esac
done

write-verbose() {
    if [ "$VERBOSE" = true ]; then
      echo "[VER] $1"
    fi
}

write-info() {
      echo "[INF] $1"
}

write-error() {
    echo "[ERR] $1"
}

# Configure behaviour for devcontainer mode or not

if [ "$DEVCONTAINER" = true ]; then
    TOOL_DEST=/usr/local/bin
else
    TOOL_DEST=$(git rev-parse --show-toplevel)/tools
    mkdir -p "$TOOL_DEST"
fi

SCRIPT_DIR=$(dirname "$(realpath "$0")")

# Ensure we have the right version of GO

if ! command -v go > /dev/null 2>&1; then
    write-error "Go must be installed manually; see https://golang.org/doc/install"
    exit 1
fi

GOVER=$(go version)
write-info "Go version: ${GOVER[*]}"

# Define os and arch
os=$(go env GOOS)
arch=$(go env GOARCH)

write-verbose "Installing tools to $TOOL_DEST"

# Install Go tools
TMPDIR=$(mktemp -d)
clean() {
    # Macos wants different flag order
    if [[ ${os} == "darwin" ]]; then
        chmod -R +w "$TMPDIR"
    else
        chmod +w -R "$TMPDIR"
    fi
    rm -rf "$TMPDIR"
}
trap clean EXIT

export GOBIN=$TOOL_DEST
export GOPATH=$TMPDIR
export GOCACHE=$TMPDIR/cache
export GO111MODULE=on

write-verbose "Installing Go tools..."

# go tools for vscode are preinstalled by base image (see first comment in Dockerfile)

# should-install() is a helper function for deciding whether
# a given installation is necessary
should-install() {
    if [ "$SKIP" == true ] && [ -f "$1" ]; then
        # We can skip installation
        return 1
    fi

    # Installation is needed
    return 0
}

# go-install() is a helper function to trigger `go install`
go-install() {
    write-verbose "Checking for $GOBIN/$1"
    if should-install "$GOBIN/$1"; then
        write-info "Installing $1"
        shift # Discard the command name so we can pass the remaining arguments to GO
        go install $@
    fi
}

# Stricter GO formatting
go-install gofumpt mvdan.cc/gofumpt@latest

# Install golangci-lint
write-verbose "Checking for $TOOL_DEST/golangci-lint"
if should-install "$TOOL_DEST/golangci-lint"; then
    write-info "Installing golangci-lint"
    curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b "$TOOL_DEST" v2.12.1 2>&1
fi

if should-install "$TOOL_DEST/golangci-lint-custom"; then
    write-info "Building golangci-lint custom"
    TOOL_DEST=$TOOL_DEST envsubst < "$SCRIPT_DIR/.custom-gcl.template.yml" > .custom-gcl.yml
    $TOOL_DEST/golangci-lint custom -v
    rm .custom-gcl.yml
fi

# Install Task
write-verbose "Checking for $TOOL_DEST/task"
if should-install "$TOOL_DEST/task"; then
    write-info "Installing go-task"
    curl -sL "https://github.com/go-task/task/releases/download/v3.49.1/task_${os}_${arch}.tar.gz" | tar xz -C "$TOOL_DEST" task
fi

if [ "$VERBOSE" == true ]; then
    echo "Installed tools: $(ls "$TOOL_DEST")"
fi

if [ "$DEVCONTAINER" == true ]; then
    # Git Permissions
    # Workaround for issue where /workspaces has different owner because checkout happens outside the container
    git config --global --add safe.directory /workspaces
fi
