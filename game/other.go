//go:build !darwin && !linux && !windows

package game

import "errors"

func findProcesses(matcher) ([]process, error) {
	return nil, errors.New("game process detection is not supported on this OS")
}

func windowsSupported() bool { return false }

func inspectWindows([]int, []int) (windowReport, error) { return windowReport{}, nil }
