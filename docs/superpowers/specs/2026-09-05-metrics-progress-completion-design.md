# Metrics Progress Completion Design

## Problem

Metric loaders update the aggregate progress counter correctly, but the one-second
ticker can emit its last periodic line before loading finishes. Stopping the ticker
then discards the terminal counter state, so output may end below the stated total
and never reports completion.

## Design

The metric progress stop function will receive the stage outcome and wait for the
ticker goroutine to exit before examining the tracker. When loading succeeded and
the loaded count exactly matches the expected total, it will emit one terminal
`Loaded metrics` line containing the final `loaded=N/N` and `percentage=100.0`
fields. It will not emit a success line for failed or incomplete metric loading.
Successful zero-work loading will report `loaded=0/0 percentage=100.0`.

Ticker shutdown remains idempotent only to the extent required by existing callers:
each returned stop function is called once. Quiet mode continues to return a no-op
stop function and emit no progress output.

## Testing

Focused tests will drive the returned progress adapter through all expected
observations and assert the initial and terminal log lines. A separate incomplete
case will assert that stopping does not claim success. Existing provider tests will
continue to verify that multiple selected file metrics produce the exact count
returned by `FileProgressTotal`.

The implementation will also be exercised through the donut-tree sample command to
confirm that normal output ends the metric stage with `Loaded metrics` at the stated
total before rendering begins.
