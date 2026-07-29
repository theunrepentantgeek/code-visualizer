# Sample regeneration tasks

## Purpose

Allow a single visualization's checked-in samples to be regenerated without
rendering every other visualization, while preserving `task samples` as the
command for a complete refresh.

## Task structure

`Taskfile.yml` will define five directly runnable tasks:

- `samples-tree-map`
- `samples-bubble-tree`
- `samples-radial-tree`
- `samples-spiral`
- `samples-scatter`

Each task depends on `build` and renders the matching sample configuration to
both `code-visualizer.png` and `code-visualizer.svg` in its own directory. Each
uses the existing stable footer and `bin/codeviz` path so output remains
byte-stable and consistent with the current task.

## Orchestration

`samples` will become an orchestration task that invokes the five visualization
tasks in the existing matrix order: tree-map, bubble-tree, radial-tree, spiral,
then scatter. Task will execute these dependencies in sequence, stopping on a
failure and retaining the direct usability of every subtask.

## Documentation and validation

The samples README will document both full regeneration and one
visualization-specific example. Validation will run a single subtask and the
orchestration task, confirming that each command renders only the intended
outputs or all five task groups respectively.
