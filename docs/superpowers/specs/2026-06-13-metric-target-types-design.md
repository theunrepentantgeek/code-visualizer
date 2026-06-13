# Metric Target Types

**Issue:** #405  
**Date:** 2026-06-13  
**Status:** Approved

## Problem

The list of metrics is growing, and it's no longer obvious which metrics apply to files versus directories. We need a classification system that:

1. Labels each metric with what it targets (file or directory).
2. Separates metrics by target in the registry so same-name metrics can coexist for different targets.
3. Provides actionable error messages when a metric is requested for the wrong target type.

## Design

### New type: `metric.Target`

Defined in `internal/metric/metric.go`:

```go
type Target int

const (
    File      Target = iota // metric applies to individual files
    Directory               // metric applies to directories (aggregates)
)
```

A `String()` method returns `"file"` or `"directory"`.

### Provider Interface addition

`provider.Interface` gains a `Target() metric.Target` method:

```go
type Interface interface {
    Name() metric.Name
    Kind() metric.Kind
    Target() metric.Target  // NEW
    Description() string
    Dependencies() []metric.Name
    DefaultPalette() palette.PaletteName
    Loader
}
```

`MetricDescriptor` gains a corresponding `Target metric.Target` field, populated by `Descriptor()`.

### Registry restructure

The backing store changes from `map[metric.Name]Interface` to `map[metric.Target]map[metric.Name]Interface`:

```go
type registry struct {
    mu        sync.RWMutex
    providers map[metric.Target]map[metric.Name]Interface
}
```

**Registration:** `register(p)` uses `p.Target()` to select the inner map. Panics if the (target, name) pair already exists.

**Lookup:** `get(name, target)` checks the target's inner map.

**Hint generation:** On a miss, iterates other target maps. If the name exists for another target, returns a hint in the error.

### Public API changes

All functions gain a `target metric.Target` parameter:

| Before | After |
|--------|-------|
| `Get(name)` | `Get(name, target)` |
| `GetDescriptor(name)` | `GetDescriptor(name, target)` |
| `All()` | `All(target)` |
| `AllDescriptors()` | `AllDescriptors(target)` |
| `Names()` | `Names(target)` |

New function:

```go
func FindWithHint(name metric.Name, target metric.Target) (Interface, error)
```

Returns the provider on success. On failure, returns an error that includes a hint if the metric exists for a different target. Example:

```
unknown file metric "dir-count"; metric "dir-count" exists for target "directory"
```

### Provider updates

All existing providers (`filesystem`, `git`, `golang`) implement `Target()` returning `metric.File`. This is the only target in use today; `metric.Directory` is reserved for future directory-level metrics.

The `providerDef` structs in `git` and `golang` packages gain a `target metric.Target` field (defaulting to `metric.File`).

### Caller updates

Every call site for `provider.Get`, `provider.GetDescriptor`, `provider.All`, `provider.AllDescriptors`, and `provider.Names` is updated to pass the appropriate target. Since all current metrics target files, all current callers pass `metric.File`.

Validation in `MetricSpec.Validate` and `TreemapCmd.validateConfig` (and equivalents) use `FindWithHint` for better error messages.

### Help metrics command

The `help metrics` table gains a "Target" column displaying "file" or "directory". Groups continue to be organized by provider package (filesystem/git/go/other).

### Testing

1. **Registry unit tests:** Register same name with different targets, verify lookup returns correct provider for each target, verify hint error on wrong target, verify panic on duplicate (target, name).
2. **Existing tests:** All pass unchanged since every provider returns `metric.File` and callers pass `metric.File`.
3. **Help metrics test:** Verifies the new "Target" column appears.
4. **Integration:** `validateConfig` tests verify hint message when wrong target is used.

## Out of scope

- Defining actual directory-targeted metrics (future work).
- Aggregation logic for directory metrics.
- Changes to the model layer (`MetricContainer` already supports both `File` and `Directory`).
