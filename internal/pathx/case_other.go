//go:build !darwin

package pathx

func diskCase(path string) string { return path }
