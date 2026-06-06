package pipeline

import (
	"fmt"
)

// ApplyFuncXR updates pipeline state by applying a function that takes an input of type X and produces an output of
// type R.
// It retrieves the value of type X from the state, applies the function, and stores the result back in the state.
// If the value of type X is not found in the state, panics (as this is a programming error).
// If the function returns an error, it stores the error in the state.
// If already in an error state, it does not apply the function and simply returns.
func ApplyFuncXR[X any, R any](
	s *State,
	f func(X) (R, error),
) {
	if s.Err() != nil {
		return
	}

	v, ok := Lookup[X](s)
	if !ok {
		msg := fmt.Sprintf("state does not contain value of type %s", keyOf[X]())
		panic(msg)
	}

	r, err := f(v)
	if err != nil {
		s.setErr(err)
		return
	}

	store(s, r)
}

// ApplyFuncXYR is a variant of ApplyFuncXR that works with functions that take two inputs (X and Y) and produce an
// output of type R.
// It retrieves the values of type X and Y from the state, applies the function, and stores the result back in the
// state.
// If either value of type X or Y is not found in the state, panics (as this is a programming error).
// If the function returns an error, it stores the error in the state.
// If already in an error state, it does not apply the function and simply returns.
func ApplyFuncXYR[X any, Y any, R any](
	s *State,
	f func(X, Y) (R, error),
) {
	if s.Err() != nil {
		return
	}

	vx, ok := Lookup[X](s)
	if !ok {
		msg := fmt.Sprintf("state does not contain value of type %s", keyOf[X]())
		panic(msg)
	}

	vy, ok := Lookup[Y](s)
	if !ok {
		msg := fmt.Sprintf("state does not contain value of type %s", keyOf[Y]())
		panic(msg)
	}

	r, err := f(vx, vy)
	if err != nil {
		s.setErr(err)
		return
	}

	store(s, r)
}
