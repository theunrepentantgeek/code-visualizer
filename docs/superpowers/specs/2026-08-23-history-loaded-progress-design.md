# History Loaded Progress Design

## Goal

Report successful completion of git history loading consistently with filesystem scanning. After history is loaded, emit an informational `History loaded` message with the number of commits loaded.

## Behavior

`LoadGitHistory` will log:

```text
History loaded commits=<count>
```

The message is emitted only after the history operation succeeds, returns at least one commit, and stores the result in `CommonState.GitHistory`. Quiet mode suppresses it, matching the existing start and progress messages.

Failures continue to return their existing errors without emitting a success-shaped completion message.

## Implementation

Add the completion log directly to `LoadGitHistory`, immediately after assigning the loaded commits to `CommonState`. Use `len(commits)` as the authoritative count.

This keeps completion reporting at the operation boundary. Logging from the progress ticker's stop callback was rejected because `stop` also runs after loading errors and could incorrectly report success. A separate completion callback abstraction would add unnecessary indirection for one message.

## Testing

Extend the existing `LoadGitHistory` logging test to assert that default mode emits `History loaded` with the fixture's three-commit count. Add coverage that quiet mode omits the completion message. Existing history-loading tests continue to cover successful state population and error behavior.
