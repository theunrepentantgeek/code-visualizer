# golangci-lint v2.13.2 Upgrade Design

## Goal

Upgrade the repository's custom golangci-lint build from v2.12.2 to v2.13.2 and adopt every new v2.13 check that is applicable to this general-purpose Go CLI.

## Version Upgrade

Keep the three existing version pins synchronized:

- `.custom-gcl.yml`
- `.devcontainer/.custom-gcl.template.yml`
- `.devcontainer/install-dependencies.sh`

The nilaway custom plugin and existing installation flow remain unchanged.

## Linter Audit

golangci-lint v2.13 introduces no brand-new linters. The audit will therefore focus on changed checks and opt-in behavior in updated linters, especially:

- new `modernize` analyzers;
- the new `iface` `unusedmethod` analyzer;
- expanded `fatcontext`, `gosec`, `staticcheck`, and `errcheck` behavior;
- new configuration options in `dupword`, `goconst`, and related enabled linters.

Checks that are active by default in already-enabled linters will remain active. Opt-in checks will be enabled when they enforce a generally useful correctness, security, performance, or maintainability property without imposing domain-specific policy.

`exhaustruct_v5` will be evaluated but not enabled merely because v2.13 replaces the deprecated `exhaustruct` implementation. This repository did not enable `exhaustruct`, and globally requiring all struct fields is not a generally applicable rule. It will only be adopted if an audit run demonstrates useful, low-noise findings.

## Remediation

All findings from applicable newly active or newly enabled checks will be fixed in production code and tests. Existing exclusions will only be extended when a finding is demonstrably invalid and the exclusion can be narrowly scoped and explained.

## Validation

The completed change must:

1. Build the custom golangci-lint v2.13.2 binary successfully.
2. Pass golangci-lint configuration verification.
3. Pass targeted tests for any remediated code.
4. Pass the repository's `task ci` workflow with no generated changes.

## Scope

This work does not upgrade unrelated tools, adopt stylistic checks unrelated to v2.13, or refactor code that is not implicated by newly applicable findings.
