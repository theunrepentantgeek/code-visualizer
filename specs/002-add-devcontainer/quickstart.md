# Quickstart: Devcontainer

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) or a compatible container runtime
- [VS Code](https://code.visualstudio.com/) with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

## Opening the Devcontainer

1. Clone the repository:
   ```bash
   git clone https://github.com/theunrepentantgeek/code-visualizer.git
   cd code-visualizer
   ```

2. Open in VS Code:
   ```bash
   code .
   ```

3. When prompted "Reopen in Container", click **Reopen in Container**.
   - Alternatively, open the Command Palette (`Ctrl+Shift+P`) and select **Dev Containers: Reopen in Container**.

4. Wait for the container to build (first time only — subsequent opens use the cached image).

5. Verify the environment:
   ```bash
   task ci
   ```
   This runs build, test, and lint — all should pass.

## What's Included

The devcontainer provides:
- **Go 1.26** — matching the project's `go.mod`
- **golangci-lint** — for linting (`task lint`)
- **gofumpt** — for formatting (`task fmt`)
- **go-task** — for running Taskfile commands
- **Pre-downloaded Go modules** — first build doesn't need network access

## Working Without the Devcontainer

The devcontainer is optional. The project works with standard Go tooling:

```bash
# Install dependencies manually
go install mvdan.cc/gofumpt@latest
# Install golangci-lint: https://golangci-lint.run/welcome/install/
# Install task: https://taskfile.dev/installation/

# Then use normally
task ci
```
