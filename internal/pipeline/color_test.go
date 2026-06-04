package pipeline

// Color is a simple test type used to test the pipeline's ability to store and retrieve values of a specific type.
type Color struct {
	name string
}

// SetColor returns a function that sets the name of a Color and returns it.
// name is the name of the color to set.
// Returns a pipeline function that takes a Color, sets its name, and returns the updated Color.
func SetColor(name string) func(Color) (Color, error) {
	return func(c Color) (Color, error) {
		c.name = name
		return c, nil
	}
}

// ExtractColor returns a function that extracts the name of a Color and stores it in the provided pointer.
// v is a pointer to a string pointer where the color name will be stored.
// Returns a pipeline function that takes a Color, extracts its name, and stores it in the provided pointer.
func ExtractColor(v **string) func(Color) (Color, error) {
	return func(c Color) (Color, error) {
		*v = &c.name
		return c, nil
	}
}
