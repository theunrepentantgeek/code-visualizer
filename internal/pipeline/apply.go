package pipeline

import (
	"fmt"
)

// ApplyFuncX updates pipeline state by applying a function that takes an input of type X.
// It retrieves the value of type X from the state and applies the function.
// If the value of type X is not found in the state, panics (as this is a programming error).
// If the function returns an error, it stores the error in the state.
// If already in an error state, it does not apply the function.
// Returns the state so calls can be chained.
func (s *State) ApplyFuncX[X any](
	f func(X) error,
) *State {
	if s.Err() != nil {
		return s
	}

	v, ok := s.lookup[X]()
	if !ok {
		msg := fmt.Sprintf("state does not contain value of type %s", keyOf[X]())
		panic(msg)
	}

	err := f(v)
	if err != nil {
		s.setErr(err)
	}

	return s
}

// ApplyFuncXR updates pipeline state by applying a function that takes an input of type X and produces an output of
// type R.
// It retrieves the value of type X from the state, applies the function, and stores the result back in the state.
// If the value of type X is not found in the state, panics (as this is a programming error).
// If the function returns an error, it stores the error in the state.
// If already in an error state, it does not apply the function.
// Returns the state so calls can be chained.
func (s *State) ApplyFuncXR[X any, R any](
	f func(X) (R, error),
) *State {
	if s.Err() != nil {
		return s
	}

	v, ok := s.lookup[X]()
	if !ok {
		msg := fmt.Sprintf("state does not contain value of type %s", keyOf[X]())
		panic(msg)
	}

	r, err := f(v)
	if err != nil {
		s.setErr(err)

		return s
	}

	s.store(r)

	return s
}

// ApplyFuncXYR is a variant of ApplyFuncXR that works with functions that take two inputs (X and Y) and produce an
// output of type R.
// It retrieves the values of type X and Y from the state, applies the function, and stores the result back in the
// state.
// If either value of type X or Y is not found in the state, panics (as this is a programming error).
// If the function returns an error, it stores the error in the state.
// If already in an error state, it does not apply the function.
// Returns the state so calls can be chained.
func (s *State) ApplyFuncXYR[X any, Y any, R any](
	f func(X, Y) (R, error),
) *State {
	if s.Err() != nil {
		return s
	}

	vx, ok := s.lookup[X]()
	if !ok {
		msg := fmt.Sprintf("state does not contain value of type %s", keyOf[X]())
		panic(msg)
	}

	vy, ok := s.lookup[Y]()
	if !ok {
		msg := fmt.Sprintf("state does not contain value of type %s", keyOf[Y]())
		panic(msg)
	}

	r, err := f(vx, vy)
	if err != nil {
		s.setErr(err)

		return s
	}

	s.store(r)

	return s
}

// ApplyFuncXY updates pipeline state by applying an error-returning function
// that consumes two typed inputs and mutates them in place. Panics if either
// input type is absent from the state. Short-circuits when state already
// holds an error. Returns the state so calls can be chained.
func (s *State) ApplyFuncXY[X any, Y any](
	f func(X, Y) error,
) *State {
	if s.Err() != nil {
		return s
	}

	vx, ok := s.lookup[X]()
	if !ok {
		panic(fmt.Sprintf("state does not contain value of type %s", keyOf[X]()))
	}

	vy, ok := s.lookup[Y]()
	if !ok {
		panic(fmt.Sprintf("state does not contain value of type %s", keyOf[Y]()))
	}

	if err := f(vx, vy); err != nil {
		s.setErr(err)
	}

	return s
}

// ApplyFuncXYZ is the three-input variant of ApplyFuncXY. It returns the
// state so calls can be chained.
func (s *State) ApplyFuncXYZ[X any, Y any, Z any](
	f func(X, Y, Z) error,
) *State {
	if s.Err() != nil {
		return s
	}

	vx, ok := s.lookup[X]()
	if !ok {
		panic(fmt.Sprintf("state does not contain value of type %s", keyOf[X]()))
	}

	vy, ok := s.lookup[Y]()
	if !ok {
		panic(fmt.Sprintf("state does not contain value of type %s", keyOf[Y]()))
	}

	vz, ok := s.lookup[Z]()
	if !ok {
		panic(fmt.Sprintf("state does not contain value of type %s", keyOf[Z]()))
	}

	if err := f(vx, vy, vz); err != nil {
		s.setErr(err)
	}

	return s
}
