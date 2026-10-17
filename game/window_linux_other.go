//go:build linux && !amd64 && !arm64

package game

func windowsSupported() bool { return false }

func inspectWindows([]int, []int) (windowReport, error) { return windowReport{}, nil }
