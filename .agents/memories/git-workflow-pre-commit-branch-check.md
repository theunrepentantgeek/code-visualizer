# Pre-commit Branch Check

Before every `git commit`, run `git branch --show-current` and verify you are NOT on `main`.

If on main, create and switch to a feature branch first. No exceptions — not even for "just a spec file" or "just a config change."

**Rationale:** The rule "never commit to main" is known, but the mistake recurs because there's no mechanical checkpoint enforcing it. This memory encodes the specific action to take, not just the principle to follow.
