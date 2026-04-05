# Feature Specification: Use Goldie for Golden File Testing

**Feature Branch**: `003-use-goldie`  
**Created**: 2026-04-05  
**Status**: Draft  
**Input**: User description: "Use goldie for unit tests as detailed in GH issue #3"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Replace handwritten golden file infrastructure with Goldie (Priority: P1)

As a developer working on code-visualizer, I want golden file tests to use the Goldie library so that I don't have to maintain custom golden file comparison infrastructure and can rely on a proven, well-supported library instead.

**Why this priority**: This is the core purpose of the feature. Eliminating custom golden file infrastructure reduces maintenance burden and provides a consistent, battle-tested approach to snapshot testing across all test suites.

**Independent Test**: Can be fully tested by running the existing test suite (`task test`) after migration — all golden file tests pass using Goldie, and no handwritten golden file comparison code remains.

**Acceptance Scenarios**:

1. **Given** the existing golden file tests in the render package, **When** a developer runs `task test`, **Then** all golden file tests pass using Goldie for comparison instead of custom pixel-by-pixel comparison code.
2. **Given** a developer has made a change that alters rendered output, **When** they run tests with Goldie's update flag, **Then** the golden files are regenerated and stored in the expected location.
3. **Given** the Goldie migration is complete, **When** a developer inspects the test code, **Then** no handwritten golden file read/write/compare logic remains.

---

### User Story 2 - Consistent golden file update workflow (Priority: P2)

As a developer, I want a single consistent mechanism to update golden files so that I don't need to remember custom environment variables or different update commands for different test packages.

**Why this priority**: A unified update mechanism improves developer experience and reduces the chance of mistakes when regenerating golden files.

**Independent Test**: Can be tested by triggering Goldie's update flag and verifying that all golden files across the project are updated consistently.

**Acceptance Scenarios**:

1. **Given** a developer wants to update golden files, **When** they use Goldie's standard update mechanism (`-update` flag), **Then** all golden files are regenerated without needing custom environment variables (e.g., `UPDATE_GOLDEN`).
2. **Given** the project uses a task runner, **When** a developer runs `task test`, **Then** existing golden files are compared (not overwritten) by default.

---

### Edge Cases

- What happens when a golden file does not yet exist for a new test? Goldie should fail the test and prompt the developer to generate it using the update flag.
- What happens when the golden file directory structure changes? Goldie manages its own `testdata/fixtures` directory (or configured equivalent), so the migration must ensure golden files are in the expected location.
- What happens when binary golden files (PNG images) are used? Goldie must support binary comparison for rendered PNG output, not just text-based diffs.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: All existing golden file tests MUST be migrated to use the Goldie library (`github.com/sebdah/goldie`) for snapshot comparison.
- **FR-002**: All handwritten golden file infrastructure (file reading, writing, pixel comparison, update-via-environment-variable logic) MUST be removed after migration.
- **FR-003**: Golden file updates MUST use Goldie's standard `-update` flag mechanism instead of custom environment variables like `UPDATE_GOLDEN`.
- **FR-004**: All existing golden file test cases MUST continue to pass after migration with no change in test coverage or correctness.
- **FR-005**: The Goldie dependency MUST be added to the project's `go.mod`.
- **FR-006**: Golden files MUST be stored in a location consistent with Goldie's conventions (configurable via Goldie options if needed to preserve existing `testdata/` paths).
- **FR-007**: Binary file comparison (PNG images) MUST be supported, as the render package uses golden files for rendered treemap images.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of existing golden file tests pass after migration to Goldie.
- **SC-002**: Zero lines of handwritten golden file read/write/compare infrastructure remain in the codebase.
- **SC-003**: Developers can update all golden files using a single standard mechanism (Goldie's `-update` flag).
- **SC-004**: No regression in test coverage — the same test scenarios are covered before and after migration.

## Assumptions

- The Goldie library (`github.com/sebdah/goldie`) supports binary file comparison suitable for PNG golden files, or can be configured/extended to do so.
- The existing `testdata/` directory structure in the render package can be preserved or adapted to work with Goldie's expected directory layout.
- Only the render package currently uses golden file testing; if additional packages adopt golden files in the future, they will also use Goldie.
- The custom `UPDATE_GOLDEN` environment variable mechanism will be fully retired and replaced by Goldie's `-update` flag.
