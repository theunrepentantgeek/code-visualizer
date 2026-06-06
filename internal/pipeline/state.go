package pipeline

import "reflect"

// State is a simple key-value store where the key is the type of the value.
type State struct {
	content map[reflect.Type]any
	err     error
}

// NewState creates a new State with an initial value of type S.
// initial is the initial value to store in the state.
// Returns a pointer to a new State containing the initial value.
func NewState[S any](initial S) *State {
	key := reflect.TypeFor[S]()

	return &State{
		content: map[reflect.Type]any{
			key: initial,
		},
	}
}

// Lookup retrieves a value of type S from the state.
// It returns the value and a boolean indicating whether the value was found.
func Lookup[S any](s *State) (S, bool) {
	var zero S
	key := reflect.TypeFor[S]()
	if v, ok := s.content[key]; ok {
		return v.(S), true
	}

	return zero, false
}

// store saves a value of type S in the state.
func store[S any](s *State, value S) {
	key := reflect.TypeFor[S]()
	s.content[key] = value
}

// setErr sets the error in the state. This is used to store any error that occurred during pipeline execution.
func (s *State) setErr(err error) {
	s.err = err
}

// Err returns the error stored in the state, if any.
func (s *State) Err() error {
	return s.err
}
