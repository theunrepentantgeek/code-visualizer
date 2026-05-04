package git

// Register adds all git metric providers to the global registry.
func Register() {
	registerAll()
}
