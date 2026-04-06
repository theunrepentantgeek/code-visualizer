# Data Model: Exclude Binary Files for Line-Count Metric

**Feature**: 004-exclude-binary-lines  
**Date**: 2026-04-06

## Entities

### FileNode (existing — no changes)

| Field       | Type             | Description                                                |
|-------------|------------------|------------------------------------------------------------|
| Path        | string           | Absolute file path                                         |
| Name        | string           | Filename only                                              |
| Extension   | string           | File extension (without dot)                               |
| Size        | int64            | File size in bytes                                         |
| LineCount   | int              | Number of lines (0 for binary files)                       |
| FileType    | string           | Extension or "no-extension"                                |
| Age         | *time.Duration   | Time since first commit (nil if no git)                    |
| Freshness   | *time.Duration   | Time since most recent commit (nil if no git)              |
| AuthorCount | *int             | Number of distinct committers (nil if no git)              |
| IsBinary    | bool             | True if file is binary (set by git or content heuristic)   |

No new fields required. The `IsBinary` flag already exists and is populated by the scanning pipeline.

### DirectoryNode (existing — no changes)

| Field | Type              | Description                        |
|-------|-------------------|------------------------------------|
| Path  | string            | Absolute directory path            |
| Name  | string            | Directory name                     |
| Files | []FileNode        | Files in this directory            |
| Dirs  | []DirectoryNode   | Child directories                  |

No new fields required.

### noFilesAfterFilterError (new)

| Field | Type   | Description                                             |
|-------|--------|---------------------------------------------------------|
| msg   | string | Human-readable error message                            |

A new error type in `cmd/codeviz/main.go` that maps to exit code 6. Raised when `FilterBinaryFiles` returns a tree with zero files.

## Relationships

```
DirectoryNode
  ├── Files: []FileNode  (1:N, owned)
  └── Dirs: []DirectoryNode  (1:N, owned, recursive)
```

The filter operates on the `DirectoryNode` tree:
- Removes `FileNode` entries where `IsBinary == true`
- Recursively prunes `DirectoryNode` entries that become empty (no files, no non-empty subdirectories)
- Returns a new tree; does not mutate the original

## State Transitions

None — this is a stateless CLI. The filter is a pure transformation of the tree.

## Validation Rules

- `FilterBinaryFiles` only called when size metric is `file-lines`
- After filtering, if total file count is 0, raise `noFilesAfterFilterError` (exit code 6)
- Binary classification (`IsBinary`) is read-only during filtering — never modified by the filter
