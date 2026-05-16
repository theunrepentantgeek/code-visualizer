package pipeline

// Stage is a single step in a pipeline. It receives the state and returns an
// error if execution should halt. When the type argument S is a pointer type,
// mutations made by the stage are visible to subsequent stages and to the
// caller of Run.
type Stage[S any] func(S) error

// Run executes stages in order against initialState. If any stage returns an
// error, execution halts immediately and the (possibly partially mutated)
// state plus the unwrapped error are returned. Run does not wrap stage
// errors; callers and stages own wrapping conventions.
func Run[S any](initialState S, stages ...Stage[S]) (S, error) {
	state := initialState
	for _, stage := range stages {
		if err := stage(state); err != nil {
			return state, err
		}
	}

	return state, nil
}
