# Development Shell Agent Guidance Design

## Goal

Help future AI agents recover when a required command-line tool is unavailable in
their current shell.

## Change

Add a short `Development shell` subsection near `Building, Testing, and Linting`
in `.github/copilot-instructions.md`.

The guidance will tell AI agents to:

- run `./dev.sh` when a required command-line tool is missing;
- execute the original command inside the shell that `dev.sh` starts;
- expect the first invocation to install missing dependencies and take a few
  minutes; and
- expect later invocations to start quickly.

## Scope

This is a documentation-only change. It will not modify `dev.sh`, dependency
installation, build commands, or CI behavior.

## Verification

Review the rendered Markdown and confirm that the instructions accurately
describe the existing `dev.sh` behavior.
