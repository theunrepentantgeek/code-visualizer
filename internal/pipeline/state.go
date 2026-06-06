package pipeline

import "reflect"

// State is a simple key-value store where the key is the type of the value.
type State struct {
	content map[reflect.Type]any
	err     error
}

// NewState creates a new State pre-populated with the given values. Each
// value is keyed by its dynamic Go type. Panics if any value is nil (no
// usable type information) or if the same type is supplied twice.
func NewState(values ...any) *State {
	s := &State{content: map[reflect.Type]any{}}
	for _, v := range values {
		if v == nil {
			panic("pipeline.NewState: nil value has no type")
		}
		key := reflect.TypeOf(v)
		if _, exists := s.content[key]; exists {
			panic("pipeline.NewState: duplicate value for type " + key.String())
		}
		s.content[key] = v
	}
	return s
}

// Lookup retrieves a value of type S from the state.
// It returns the value and a boolean indicating whether the value was found.
func Lookup[S any](s *State) (S, bool) {
	var zero S

	key := keyOf[S]()
	if v, ok := s.content[key]; ok {
		//nolint:revive // Invariant is that this value will be of type S
		return v.(S), true
	}

	return zero, false
}

// Store saves a value of type S in the state, overwriting any existing
// value of the same type.
func Store[S any](s *State, value S) {
	key := keyOf[S]()
	s.content[key] = value
}

// keyOf returns a key to use for the specified type.
func keyOf[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

// setErr sets the error in the state. This is used to store any error that occurred during pipeline execution.
func (s *State) setErr(err error) {
	s.err = err
}

// Err returns the error stored in the state, if any.
func (s *State) Err() error {
	return s.err
}
