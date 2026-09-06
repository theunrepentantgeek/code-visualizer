# Metrics Progress Completion Design

## Problem

Metric loaders update their aggregate progress counter correctly, but the
one-second ticker can emit its final periodic line before loading finishes.
Ticker shutdown then discards the terminal counter state, so output can end
below the stated total and omit the completion of the metric-loading stage.

PR #729 attempted to pass the operation outcome into ticker cleanup by changing
the stop callback to `func(bool)`. Cleanup should not interpret an unlabelled
boolean or own the success state of the operation it observes.

Metric progress is meaningful only when file-based metric observations are
expected. A zero total represents no file-metric work and must produce neither
a starting progress line nor a completion line.

## Design

`RunProviders` will own the metric progress lifecycle because it already owns
the loading operation and receives its result:

1. Calculate the expected file-progress total.
2. Build the progress reporter and start its ticker only when the total is
   greater than zero.
3. Run authorship and file-based metric loading.
4. Stop the ticker and wait for its goroutine to exit.
5. If loading succeeded and progress was enabled, emit one terminal
   `Loaded metrics` line with `loaded=N/N` and `percentage=100.0`.
6. If loading failed, return the existing wrapped error without emitting a
   success line.

`BuildMetricProgress` retains its existing
`(provider.MetricProgress, func())` API. The returned function only stops the
ticker; it does not accept or infer an operation outcome.

The generic ticker shutdown will become synchronous. Its stop function will
signal the goroutine and wait for it to exit, ensuring that no periodic
`Loading metrics.` line can appear after the terminal `Loaded metrics` line.

The provider-loading control flow will funnel every result through one ticker
shutdown point. Completion logging will happen after that shutdown and only on
the successful path.

## Error Handling

Existing error wrapping remains unchanged. Authorship-loading and provider
errors stop progress reporting before returning. Neither an incomplete count
nor a loader that fails after reporting every expected observation can produce
a `Loaded metrics` success line.

Quiet mode continues to suppress progress. A zero file-progress total also
suppresses progress regardless of verbosity because no file-metric loading
work exists.

## Testing

Focused tests will verify:

- a zero total creates no reporter and emits no progress output;
- successful progress starts at `0/N`, ends at `N/N`, and emits the terminal
  line after all periodic lines;
- failed provider loading does not emit `Loaded metrics`, even if the loader
  reported all expected observations;
- the successful `RunProviders` path emits `Loaded metrics`,
  `loaded=N/N`, and `percentage=100.0` using an actual registered provider;
- existing quiet-mode and periodic-progress behavior remains intact.

No unrelated files, including `.custom-gcl.yml`, will be changed.
