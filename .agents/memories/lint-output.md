# Lint Output Handling

Do not pipe lint/ci output through grep to filter results — it discards issue details. Use a subagent or redirect to a file, then inspect.

## Rationale

When debugging lint failures, piping `task lint` through grep captures that issues exist but loses critical details about the nature of each issue. This causes wasted extra lint cycles. Lint output should be captured in full — either via a subagent (as the project guidelines already suggest) or by redirecting to a file and then inspecting.
