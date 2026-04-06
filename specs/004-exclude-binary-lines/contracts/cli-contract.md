# CLI Contract: Exclude Binary Files for Line-Count Metric

**Feature**: 004-exclude-binary-lines  
**Date**: 2026-04-06

## Behavioural Changes

### Binary File Exclusion (line-count sizing only)

When `--size file-lines` is specified, binary files are excluded from the visualization. No new CLI flags are introduced — the behaviour is automatic.

**Condition**: `--size file-lines`

**Effect**: Binary files (as detected by git metadata or content heuristics) are removed from the treemap. Directories that become empty after binary removal are also removed.

**No effect when**: `--size` is any value other than `file-lines` (`file-size`, `file-age`, `file-freshness`, `author-count`).

### Verbose Logging

When `--verbose` is enabled and `--size file-lines` is specified, each excluded binary file is logged to stderr at DEBUG level:

```
time=... level=DEBUG msg="excluding binary file" path=/abs/path/to/file.bin
```

A summary is also logged:

```
time=... level=DEBUG msg="binary file filter complete" excluded=N remaining=M
```

## New Exit Code

| Code | Meaning                                            |
|------|----------------------------------------------------|
| 6    | No files available after filtering binary files    |

This exit code is raised when:
- `--size file-lines` is specified, AND
- All files in the target directory tree are binary (none remain after filtering)

### JSON error output (when `--format json`)

```json
{
  "error": "no files available for visualization after excluding binary files",
  "code": 6
}
```

### Text error output (default)

```
error: no files available for visualization after excluding binary files
```

## Updated Exit Code Table

| Code | Meaning                                            |
|------|----------------------------------------------------|
| 0    | Success — PNG written to output path               |
| 1    | Invalid arguments or validation failure            |
| 2    | Target path does not exist or is not a directory   |
| 3    | Git-required metric used on non-git directory      |
| 4    | Output path error (parent missing, permission)     |
| 5    | Internal error during scan/render                  |
| 6    | No files available after filtering                 |

## Unchanged Behaviour

- All existing flags, defaults, and enum values remain unchanged.
- `--fill` and `--border` metric choices do not affect which files appear in the treemap.
- The `--fill` default (same as `--size`) still applies — if `--size file-lines` is used without `--fill`, the fill metric is also `file-lines`, but exclusion is driven by the size metric only.
