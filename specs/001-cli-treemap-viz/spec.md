# Feature Specification: CLI Treemap Visualization

**Feature Branch**: `001-cli-treemap-viz`
**Created**: 2026-04-04
**Status**: Draft
**Input**: User description: "CLI treemap visualization of file trees and git repos with configurable metrics and colour palettes, outputting PNG images"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Visualize a Directory by File Size (Priority: P1)

A developer points the CLI tool at a local directory and generates a treemap PNG where each rectangle represents a file, sized proportionally by file size in bytes. The directory structure determines the nesting of rectangles. The output is a single PNG image saved to a specified path.

**Why this priority**: This is the most fundamental use case — scanning a directory and rendering a treemap with one metric. It validates the entire pipeline from scanning to layout to image output, and delivers immediate visual value with no git dependency.

**Independent Test**: Run the tool against a sample directory with known file sizes, verify a PNG is produced, and confirm that larger files occupy visually larger rectangles.

**Acceptance Scenarios**:

1. **Given** a directory containing files of varying sizes, **When** the user runs the tool specifying file-size as the size metric and an output path, **Then** a PNG image is produced where rectangle areas are proportional to file size.
2. **Given** a directory with nested subdirectories, **When** the tool is run, **Then** the treemap reflects the directory hierarchy with nested groupings.
3. **Given** an empty directory, **When** the tool is run, **Then** the tool exits with a clear error message indicating no files were found.

---

### User Story 2 - Colour Files by a Metric (Priority: P2)

A developer generates a treemap where the fill colour of each file rectangle is determined by a chosen metric (e.g., file age) mapped through a chosen colour palette (e.g., Temperature). This allows visual comparison across the codebase — for instance, quickly spotting old files versus recently created ones.

**Why this priority**: Colour mapping is the primary way users will interpret metric data visually. Without it, the treemap only conveys one dimension (size). Adding fill colour doubles the information density and is essential for practical use.

**Independent Test**: Run the tool with a size metric and a fill-colour metric plus palette, verify the output PNG contains rectangles coloured according to the palette scale, and confirm that files with extreme metric values receive the extreme palette colours.

**Acceptance Scenarios**:

1. **Given** a directory scanned with file-age as the fill metric and the Temperature palette, **When** the treemap is rendered, **Then** the oldest files appear dark blue and the newest files appear bright red, with intermediate ages mapped to intermediate colours.
2. **Given** a directory scanned with file-type as the fill metric and the Categorization palette, **When** the treemap is rendered, **Then** each distinct file extension receives a distinct colour from the 12-colour set.
3. **Given** a metric with values that all fall within a narrow range, **When** the treemap is rendered, **Then** the palette maps still distinguish the values across available steps rather than collapsing to a single colour.

---

### User Story 3 - Border Colour by a Second Metric (Priority: P3)

A developer generates a treemap that uses one metric for fill colour and a different metric for border colour, each with their own palette. This enables three-dimensional analysis in a single image: size shows one metric, fill shows another, and border shows a third.

**Why this priority**: Border colouring adds a third data dimension. It builds on the fill-colour infrastructure (P2) and is a natural extension, but is less critical than having fill colour working first.

**Independent Test**: Run the tool with distinct metrics for size, fill, and border, verify the output PNG shows borders coloured differently from fills, and confirm the border palette mapping is independent of the fill palette.

**Acceptance Scenarios**:

1. **Given** file-size as the size metric, file-age as fill (Temperature palette), and author-count as border (Good/Bad palette), **When** the treemap is rendered, **Then** each rectangle has a fill colour from Temperature and a border colour from Good/Bad, independently mapped.
2. **Given** the same metric chosen for both fill and border but with different palettes, **When** the treemap is rendered, **Then** fill and border colours reflect the same underlying values but through their respective palette scales.

---

### User Story 4 - Git-Aware Metrics (Priority: P4)

A developer points the CLI tool at a git repository and generates a treemap using git-derived metrics: file age (time since first commit), file freshness (time since most recent commit), and number of distinct authors. These metrics require access to git history.

**Why this priority**: Git metrics are a key differentiator of this tool versus basic file explorers. However, the scanning, layout, and rendering pipeline must work first (P1–P3) before adding git-specific data collection.

**Independent Test**: Run the tool against a git repository with known commit history, verify that file-age, file-freshness, and author-count metrics produce correct values by comparing against `git log` output for specific files.

**Acceptance Scenarios**:

1. **Given** a git repository, **When** the user selects file-age as a metric, **Then** each file's age is calculated as the duration from its first commit to now.
2. **Given** a git repository, **When** the user selects file-freshness as a metric, **Then** each file's freshness is calculated as the duration from its most recent commit to now.
3. **Given** a git repository, **When** the user selects author-count as a metric, **Then** each file's author count reflects the number of distinct committers who have modified that file.
4. **Given** a directory that is not a git repository, **When** a git-dependent metric is selected, **Then** the tool exits with a clear error message explaining that git history is required for the chosen metric.

---

### User Story 5 - Selecting Colour Palettes (Priority: P5)

A developer chooses among the four built-in colour palettes — Categorization, Temperature, Good/Bad, and Neutral — for either fill or border colour, depending on the semantic meaning of the metric they are visualising.

**Why this priority**: The palettes are a supporting concern; the infrastructure for mapping metrics to colours (P2/P3) must exist first. This story ensures all four palettes are available and correctly defined.

**Independent Test**: Generate treemaps using each of the four palettes in turn and verify visually and programmatically that the correct number of distinct colour steps appear and match the palette definitions.

**Acceptance Scenarios**:

1. **Given** the Categorization palette is selected, **When** a metric with up to 12 distinct values is mapped, **Then** each value receives one of 12 visually distinct colours with no implied ordering.
2. **Given** the Temperature palette is selected, **When** a metric with a positive/negative range is mapped, **Then** the midpoint maps to white, positive extremes to bright red, and negative extremes to dark blue, across 11 steps.
3. **Given** the Good/Bad palette is selected, **When** a metric with a worst-to-best range is mapped, **Then** worst values map to red and best values map to green, progressing through orange and yellow across 13 steps.
4. **Given** the Neutral palette is selected, **When** a metric is mapped, **Then** values range from black to white across 9 monochromatic steps.

---

### Edge Cases

- What happens when a file has zero bytes? It should still appear in the treemap as a minimum-size rectangle, not be invisible.
- What happens when the target directory contains symbolic links? Symbolic links are followed if they point to regular files; circular links must not cause infinite recursion.
- What happens when a binary file is encountered and the line-count metric is selected? Binary files (as identified by git) receive a value of zero for line count and are still displayed in the treemap.
- What happens when the target directory is not a git repository and the line-count metric is selected? Since binary detection defers to git, all files in a non-git directory are treated as text for line counting purposes.
- What happens when a file has no git history (e.g., untracked file) and a git metric is selected? Untracked files receive a sentinel "unknown" value and are rendered using a distinct indicator (e.g., the first or last palette step, consistently).
- What happens when the number of distinct file types exceeds the Categorization palette's 12 colours? Colours wrap around (cycle) with a logged warning that some types share colours.
- What happens when the user specifies an output path in a directory that does not exist? The tool exits with a clear error rather than silently creating directories.
- What happens with permission-denied files during scanning? The tool logs a warning for each inaccessible file and continues scanning the rest of the tree.

## Clarifications

### Session 2026-04-04

- Q: Should file/directory names appear as text labels on treemap rectangles? → A: Labels shown only on rectangles large enough to display them legibly; omitted on small rectangles.
- Q: Can file-type (categorical) be used as the size metric? → A: No. File-type is only allowed for fill or border colour; the size metric must be numeric.
- Q: What are the defaults when fill colour or border colour are not specified? → A: If no fill metric is specified, the size metric is reused for fill. If no border metric is specified, no borders are displayed.
- Q: What default palette should be used for each metric when no palette is explicitly specified? → A: file-size: Neutral, file-lines: Neutral, file-age: Temperature, file-freshness: Temperature, author-count: Good/Bad, file-type: Categorization.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST accept a directory path as input and recursively scan all files within it.
- **FR-002**: The tool MUST compute the following metrics for each file: size in bytes, length in lines (binary files as identified by git report zero; in non-git directories all files are treated as text), file type (based on file extension).
- **FR-003**: The tool MUST compute the following git-derived metrics when the target is a git repository: file age (duration since first commit), file freshness (duration since most recent commit), number of distinct authors.
- **FR-004**: The tool MUST require the user to select exactly one numeric metric for determining the size of treemap rectangles. Categorical metrics (e.g., file-type) MUST NOT be allowed as the size metric.
- **FR-005**: The tool MUST allow the user to optionally select one metric and one colour palette for the fill colour of treemap rectangles. If no fill metric is specified, the tool MUST default to using the size metric for fill colour. If no palette is specified, the tool MUST use the metric's default palette.
- **FR-006**: The tool MUST allow the user to optionally select one metric and one colour palette for the border colour of treemap rectangles. If no border metric is specified, the tool MUST render rectangles without borders. If a border metric is specified without a palette, the tool MUST use the metric's default palette.
- **FR-007**: The tool MUST produce a treemap layout based on the directory structure, with files as leaf nodes and directories as nested groups.
- **FR-008**: The tool MUST render the treemap to a PNG image file at a user-specified output path.
- **FR-008a**: The tool MUST display file and directory names as text labels on treemap rectangles that are large enough to render them legibly; labels MUST be omitted on rectangles too small for readable text.
- **FR-009**: The tool MUST provide four built-in colour palettes: Categorization (12 colours, unordered), Temperature (11 steps, dark blue through white to bright red), Good/Bad (13 steps, red through orange and yellow to green), and Neutral (9 monochromatic steps, black to white).
- **FR-010**: The tool MUST map metric values to palette colours using discrete steps — no colour interpolation.
- **FR-010a**: Each metric MUST have a default palette used when the user does not specify one: file-size → Neutral, file-lines → Neutral, file-age → Temperature, file-freshness → Temperature, author-count → Good/Bad, file-type → Categorization.
- **FR-011**: The tool MUST exit with a clear, actionable error message when a git-dependent metric is requested but the target directory is not a git repository.
- **FR-012**: The tool MUST log a warning and continue scanning when individual files are inaccessible due to permissions.
- **FR-013**: The tool MUST display files with zero-valued size metrics as minimum-size rectangles rather than omitting them.
- **FR-014**: The tool MUST support both human-readable and JSON output formats for any non-image diagnostic or error output.

### Key Entities

- **FileNode**: Represents a single file in the scanned tree. Attributes: path, name, extension, and computed metric values (size, line count, file type, and optionally age, freshness, author count).
- **DirectoryNode**: Represents a directory in the scanned tree. Contains child FileNodes and child DirectoryNodes. Serves as a grouping container in the treemap layout.
- **Metric**: A named measurement that can be computed for a file. Has a value type (numeric or categorical) and a range/set of possible values. Categorical metrics are restricted to colour mapping only; they cannot be used as the size metric.
- **ColourPalette**: An ordered sequence of predefined colours with a name and a fixed step count. Maps metric values to colours by dividing the metric range into palette-sized buckets.
- **TreemapRectangle**: A visual element in the rendered treemap. Has position, dimensions (determined by size metric), fill colour (determined by fill metric + palette), and border colour (determined by border metric + palette).
- **RenderConfiguration**: The user's chosen combination of target path, output path, size metric, optional fill metric + palette, and optional border metric + palette.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can generate a treemap PNG from a directory of 1,000 files in under 5 seconds on standard hardware.
- **SC-002**: Users can generate a treemap PNG from a git repository of 1,000 files (including git metric collection) in under 15 seconds.
- **SC-003**: The tool correctly computes all six metrics — verified by comparison against independent calculations for a reference repository with known characteristics.
- **SC-004**: All four colour palettes produce visually distinct, correctly ordered colour steps — verified by rendering test images and comparing against reference images.
- **SC-005**: Users can specify size, fill, and border metrics independently in a single command invocation without errors or conflicts.
- **SC-006**: The tool provides clear, actionable error messages for all invalid input combinations (missing directory, non-git repo with git metric, invalid output path) — no silent failures.
- **SC-007**: Generated PNG images are valid, openable in standard image viewers, and accurately reflect the directory structure and metric values of the input.

## Assumptions

- Users have git installed and available on their PATH when using git-derived metrics.
- The target directory is on a local filesystem (no network/remote filesystem support in MVP).
- File type classification is based solely on file extension; files without extensions are grouped as "no extension".
- Binary file detection is deferred to git (using git's built-in binary/text classification). In non-git directories, all files are treated as text for line-counting purposes.
- The default output image dimensions are a reasonable fixed size (e.g., 1920×1080); configurable image dimensions are a future feature.
- Colour palette definitions are hardcoded for MVP; user-defined palettes are a future feature.
- The treemap layout algorithm is squarified treemap (or similar aspect-ratio-optimised algorithm); the specific algorithm choice is an implementation detail.
- Symbolic links are followed for regular files but not for directories, to avoid cycles.
