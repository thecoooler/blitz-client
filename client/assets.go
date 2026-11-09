package client

// ReadClientAsset reads a plain or DVPL resource and returns its payload.
func ReadClientAsset(path string) ([]byte, error) {
	return decodeResourceFile(path)
}

// ResolveClientAsset finds a resource as a plain file or as its DVPL
// sibling. A relative path ending in .dvpl prefers the unpacked file. The
// result is empty when neither exists.
func ResolveClientAsset(dataDir, relative string) string {
	file, _ := resolveResourceFile(dataDir, relative)
	return file
}
