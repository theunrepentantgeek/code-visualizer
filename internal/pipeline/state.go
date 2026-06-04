package pipeline

// State is a simple key-value store where the key is the type of the value.
type State struct {
	content map[string]any
}

// NewState creates a new State with an initial value of type S.
// initial is the initial value to store in the state.
// Returns a pointer to a new State containing the initial value.
func NewState[S any](initial S) *State {
	key := keyOf[S]()

	return &State{
		content: map[string]any{
			key: initial,
		},
	}
}

// Lookup retrieves a value of type S from the state.
// It returns the value and a boolean indicating whether the value was found.
func Lookup[S any](s *State) (S, bool) {
	var zero S
	key := keyOf[S]()
	if v, ok := s.content[key]; ok {
		return v.(S), true
	}

	return zero, false
}

// store saves a value of type S in the state.
func store[S any](s *State, value S) {
	key := keyOf[S]()
	s.content[key] = value
}
