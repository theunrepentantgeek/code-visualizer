# Abstractions — `internal/viz`

## Grain

**Purpose.**

- `Grain` names the granularity of the nodes a diagram shows: files and directories, or directories only ([grain.go#L4](grain.go#L4), [grain.go#L6](grain.go#L6)).

**Boundary and invariants.**

- It is a string enum whose values are exactly the user-facing config/CLI words `"file"` and `"directory"`, so config parsing compares against the constants rather than re-spelling them ([grain.go#L8](grain.go#L8), [../scatter/stages.go#L173](../scatter/stages.go#L173)).
- Grain choice determines which model level supplies metric values, so it is resolved before metric resolution ([../scatter/stages.go#L184](../scatter/stages.go#L184)).

**Related operations.**

- Visualization packages alias the type rather than redefining it ([../radialtree/node.go#L18](../radialtree/node.go#L18)).

**Proper-use patterns.**

- Branch on `viz.GrainDirectory` when deciding whether files are collected at all ([../scatter/data.go#L109](../scatter/data.go#L109)).
- Map an empty config string to `GrainFile` as the default during stage resolution ([../scatter/stages.go#L175](../scatter/stages.go#L175)).

**Anti-patterns.**

- Do not declare a private grain enum inside a visualization package; alias `viz.Grain` so CLI, config, and layout agree on the vocabulary ([../radialtree/node.go#L17](../radialtree/node.go#L17)).

**Source locations.**

- [grain.go#L4](grain.go#L4) — `Grain` and its constants.

## LabelMode

**Purpose.**

- `LabelMode` names which node labels a diagram renders: all, folders only, lap boundaries, or none ([label_mode.go#L5](label_mode.go#L5), [label_mode.go#L13](label_mode.go#L13)).

**Boundary and invariants.**

- Like `Grain`, the constant values are the user-facing config words, and config values are converted directly to the type ([label_mode.go#L9](label_mode.go#L9), [../bubbletree/stages.go#L36](../bubbletree/stages.go#L36)).
- The mode is a *layout* input, not a rendering afterthought: layout decides per node whether a label is shown ([../bubbletree/layout.go#L25](../bubbletree/layout.go#L25), [../bubbletree/node.go#L22](../bubbletree/node.go#L22)).

**Related operations.**

- Visualization packages alias the type and re-export only the constants they support ([../bubbletree/node.go#L8](../bubbletree/node.go#L8), [../radialtree/node.go#L8](../radialtree/node.go#L8)).

**Proper-use patterns.**

- Carry the resolved mode in the visualization's pipeline state and pass it into `Layout` ([../bubbletree/state.go#L19](../bubbletree/state.go#L19), [../bubbletree/layout.go#L25](../bubbletree/layout.go#L25)).

**Anti-patterns.**

- Do not re-export constants a visualization cannot honour; bubble tree and radial tree deliberately alias only `LabelAll`, `LabelFoldersOnly` and `LabelNone` ([../bubbletree/node.go#L11](../bubbletree/node.go#L11), [../radialtree/node.go#L11](../radialtree/node.go#L11)).

**Source locations.**

- [label_mode.go#L5](label_mode.go#L5) — `LabelMode` and its constants.
