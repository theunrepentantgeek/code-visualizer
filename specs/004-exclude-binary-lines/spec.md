# Feature Specification: Exclude Binary Files for Line-Count Metric

**Feature Branch**: `004-exclude-binary-lines`  
**Created**: 2026-04-06  
**Status**: Draft  
**Input**: User description: "Exclude binary files for text metrics as described in GH issue #4"  
**GitHub Issue**: [#4 — Skip binary files when using line count](https://github.com/theunrepentantgeek/code-visualizer/issues/4)

## Clarifications

### Session 2026-04-06

- Q: When the size metric is a git-based metric (file-age, file-freshness, or author-count), should binary files be included or excluded? → A: Include binary files for all size metrics except line count.
- Q: Which exit code should the tool use when line count is the size metric and no text files remain after excluding binaries? → A: New exit code 6 — a distinct "no files after filtering" code.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Binary Files Omitted from Line-Count Treemap (Priority: P1)

A developer visualizes a repository that contains a mix of source code and binary assets (images, compiled objects, fonts). They choose line count as the size metric. The resulting treemap shows only text-based source files — binary files are completely absent from the visualization so they do not distort the proportional representation of code.

**Why this priority**: This is the core ask of the feature. Binary files have no meaningful line count, and including them (as zero-area or minimum-area rectangles) adds visual noise without conveying useful information.

**Independent Test**: Point the tool at a directory containing both text files and binary files, using line count as the size metric. Verify that binary files do not appear anywhere in the output.

**Acceptance Scenarios**:

1. **Given** a directory containing text files and binary files, **When** the user generates a treemap with line count as the size metric, **Then** binary files are completely excluded from the visualization.
2. **Given** a directory where all files are binary, **When** the user generates a treemap with line count as the size metric, **Then** the tool reports that no files are available for visualization and exits with an appropriate error.
3. **Given** a directory containing binary files in nested subdirectories, **When** the user generates a treemap with line count as the size metric, **Then** directories that contained only binary files are also excluded from the visualization.

---

### User Story 2 — Binary Files Still Included for File-Size Treemap (Priority: P2)

A developer visualizes the same repository using file size (bytes) as the size metric. All files — including binaries — appear in the treemap because byte size is meaningful for every file type.

**Why this priority**: This confirms backward compatibility. The exclusion behaviour must only apply to line-count sizing; file-size mode must remain unchanged.

**Independent Test**: Point the tool at a directory containing both text files and binary files, using file size as the size metric. Verify that every file (text and binary) appears in the output.

**Acceptance Scenarios**:

1. **Given** a directory containing text files and binary files, **When** the user generates a treemap with file size as the size metric, **Then** all files — including binary files — appear in the visualization.
2. **Given** a binary file that is the largest file in a directory, **When** the user generates a treemap with file size as the size metric, **Then** that binary file occupies the largest rectangle, proportional to its byte size.

---

### User Story 3 — Binary Files Usable as Fill/Border Metric Source (Priority: P3)

A developer uses line count as the size metric but selects a different metric (file type, file age, etc.) for fill or border colouring. Binary files are still excluded from the treemap because they have no line count and therefore no rectangle to colour.

**Why this priority**: This clarifies the interaction between the size metric (which determines rectangle existence) and the fill/border metrics (which only colour existing rectangles). The exclusion is driven by the size metric, not the colour metrics.

**Independent Test**: Generate a treemap with line count as the size metric and file type as the fill metric. Verify that binary files remain excluded even though file type is defined for them.

**Acceptance Scenarios**:

1. **Given** a directory containing text and binary files, **When** the user generates a treemap with line count as size and file type as fill, **Then** binary files are excluded (no rectangle exists for them to be coloured).
2. **Given** a directory containing text and binary files, **When** the user generates a treemap with file size as size and file type as fill, **Then** binary files are included and coloured by their file type.

---

### Edge Cases

- What happens when a directory becomes empty after binary files are excluded? The directory should be omitted from the treemap entirely.
- What happens in a non-git repository where binary detection relies on content heuristics? The same exclusion behaviour applies — any file detected as binary (via the existing null-byte/long-line heuristic) is excluded from line-count treemaps.
- What happens when verbose mode is enabled and binary files are excluded? The tool should log which files were excluded and why, so users can verify the behaviour.
- What happens when every file in the target directory is binary? The tool should report an error indicating no files are available for visualization (same behaviour as an empty directory).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST exclude binary files from the treemap when the size metric is line count.
- **FR-002**: The system MUST include binary files in the treemap when the size metric is any metric other than line count (file size, file age, file freshness, author count).
- **FR-003**: The system MUST exclude directories that contain only binary files (and no text files) from the treemap when the size metric is line count.
- **FR-004**: The system MUST report an error and exit with exit code 6 (no files after filtering) when the size metric is line count and no text files are found after excluding binaries.
- **FR-005**: The system MUST log excluded binary files when verbose mode is enabled.
- **FR-006**: The system MUST use the existing binary-detection mechanisms (git attribute detection and content-based null-byte/long-line heuristic) to classify files — no new detection method is required.
- **FR-007**: The exclusion MUST apply based on the size metric choice only; fill and border metric selections do not affect which files appear in the treemap.

### Key Entities

- **File**: A scanned file with attributes including path, name, size (bytes), line count, and a binary classification flag.
- **Binary classification**: A boolean property of each file, determined by git metadata (in git repositories) or content heuristics (in non-git directories), indicating whether the file contains non-text content.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: When line count is the size metric, 100% of binary files are absent from the generated treemap.
- **SC-002**: When file size is the size metric, 100% of binary files remain present in the generated treemap.
- **SC-003**: A repository containing a mix of source code and binary assets produces a treemap (under line-count sizing) that is visually comparable in clarity to manually removing the binary files beforehand.
- **SC-004**: All existing tests continue to pass — no regression in file-size or other metric modes.
- **SC-005**: Directories that become empty after binary exclusion do not appear as empty rectangles in the visualization.

## Assumptions

- The existing binary-detection mechanisms (git-based and content-heuristic-based) are sufficient and accurate for this feature; no new detection logic needs to be introduced.
- Binary exclusion only affects the set of files included in the treemap; it does not change how binary detection itself works.
- The fill and border metrics are only applied to files that have a rectangle (i.e., files that pass the size-metric filter), so no special handling is needed for fill/border when binaries are excluded.
- This feature does not introduce any new CLI flags; the behaviour change is automatic based on the existing `--size` flag value.
- Git-based size metrics (file-age, file-freshness, author-count) are meaningful for binary files, so binary files remain included when these metrics are used for sizing.
