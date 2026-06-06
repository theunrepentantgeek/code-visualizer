package pipeline

import "fmt"

// Texture is a simple test type used to test the pipeline's ability to store and retrieve values of a specific type.
type Texture struct {
	name string
}

// CreateTexture is a pipeline funciton creates a new Texture with the given Color and Kind.
// name is the name of the texture to set.
func CreateTexture(color Color, kind Kind) (Texture, error) {
	var t Texture
	t.name = fmt.Sprintf("%s-%s", color.name, kind.name)
	return t, nil
}

// ExtractTexture returns a function that extracts the name of a Texture and stores it in the provided pointer.
// v is a pointer to a string pointer where the texture name will be stored.
// Returns a pipeline function that takes a Texture, extracts its name, and stores it in the provided pointer.
func ExtractTexture(v **string) func(Texture) (Texture, error) {
	return func(t Texture) (Texture, error) {
		*v = &t.name
		return t, nil
	}
}
