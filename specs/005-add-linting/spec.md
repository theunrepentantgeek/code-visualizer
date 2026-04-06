# Feature Specification: Configure Linting

**Feature Branch**: `005-add-linting`  
**Created**: 2026-04-06  
**Status**: Draft  
**Input**: User description: "Add linting as per GH Issue #11"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Developer Runs Comprehensive Linting (Priority: P1)

As a developer working on this project, I want to run a comprehensive set of linters via a single command so that I can catch code quality issues, nil-safety problems, and style inconsistencies before committing.

**Why this priority**: The core purpose of this feature — without linting working locally, the entire feature has no value.

**Independent Test**: Can be fully tested by running `task lint` and verifying that the expanded linter set (including nilaway) runs against the codebase and reports findings.

**Acceptance Scenarios**:

1. **Given** a developer has set up the project, **When** they run `task lint`, **Then** a custom-built linter binary (with nilaway plugin) executes against the codebase and reports any issues found.
2. **Given** a developer has set up the project, **When** they run `task lint` on clean code, **Then** the command exits with a zero exit code and no warnings.

---

### User Story 2 - Developer Sets Up Environment with Custom Linter (Priority: P1)

As a developer setting up this project for the first time (or rebuilding the devcontainer), I want the install-dependencies script to automatically build the custom linter binary so that I don't need manual setup steps.

**Why this priority**: Equally critical — developers cannot lint without the custom binary being built first.

**Independent Test**: Can be tested by running the install-dependencies script and verifying that the custom linter binary is produced in the tools directory.

**Acceptance Scenarios**:

1. **Given** a fresh devcontainer build, **When** the install-dependencies script runs, **Then** a custom linter binary is built and placed in the tools directory.
2. **Given** the custom binary already exists and skip-installed mode is used, **When** the script runs, **Then** the build step is skipped.

---

### User Story 3 - Developer Fixes Existing Lint Issues (Priority: P2)

As a developer, I want all existing code to pass the new expanded linter configuration so that the codebase starts clean and future regressions are immediately visible.

**Why this priority**: Without a clean baseline, the expanded linting adds noise rather than value — but this depends on the linters being configured first.

**Independent Test**: Can be tested by running the lint command on the full codebase and confirming a zero exit code with no suppressions or ignore directives that hide real issues.

**Acceptance Scenarios**:

1. **Given** the expanded linter configuration is in place, **When** the lint command is run against the entire codebase, **Then** no lint errors are reported.
2. **Given** an existing lint issue is found, **When** the developer addresses it, **Then** the fix corrects the root cause rather than suppressing the warning (preferred approach).

---

### Edge Cases

- What happens when the template expansion tool is not available in the build environment?
- What happens when a new linter in the expanded set conflicts with an existing code pattern used throughout the project?
- What happens when the custom linter build fails due to a plugin version incompatibility?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST use a custom-built linter binary that includes the nilaway plugin.
- **FR-002**: The install-dependencies script MUST build the custom linter binary using the linter's custom build command with a templated configuration file.
- **FR-003**: The lint task MUST invoke the custom linter binary instead of the standard linter executable.
- **FR-004**: The linter configuration MUST include a comprehensive set of linters matching the configuration used by the reference project (go-vcr-tidy).
- **FR-005**: All existing code MUST pass the new expanded linter configuration without errors.
- **FR-006**: The preference MUST be to fix lint issues at the root cause rather than suppress warnings, unless suppression is clearly justified.
- **FR-007**: The tidy task MUST also use the custom linter binary for its auto-fix invocation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Running the lint command produces a zero exit code with no errors on the entire codebase.
- **SC-002**: The custom linter binary includes the nilaway plugin and all configured linters run successfully.
- **SC-003**: The developer setup script builds the custom linter binary without manual intervention.
- **SC-004**: The linter configuration covers the same set of linters as the reference project.

## Assumptions

- The devcontainer base image includes the template expansion tool (or it can be added as a dependency).
- The linter's custom build and plugin system are available in the version already installed (v2.8.0).
- The reference project's linter configuration serves as the baseline; the imported configuration will be adapted only where project-specific differences require it.
- The nilaway plugin version is compatible with the installed linter version.
- Fixing lint issues is preferred over suppression; any necessary suppressions will be documented with justification.
