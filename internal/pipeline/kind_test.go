package pipeline

// Kind is a simple test type used to test the pipeline's ability to store and retrieve values of a specific type.
type Kind struct {
	name string
}

// SetKind returns a function that sets the name of a Kind and returns it.
// name is the name of the kind to set.
// Returns a pipeline function that takes a Kind, sets its name, and returns the updated Kind.
func SetKind(name string) func(Kind) (Kind, error) {
	return func(k Kind) (Kind, error) {
		k.name = name

		return k, nil
	}
}

// ExtractKind returns a function that extracts the name of a Kind and stores it in the provided pointer.
// v is a pointer to a string pointer where the kind name will be stored.
// Returns a pipeline function that takes a Kind, extracts its name, and stores it in the provided pointer.
func ExtractKind(v **string) func(Kind) (Kind, error) {
	return func(k Kind) (Kind, error) {
		*v = &k.name

		return k, nil
	}
}
