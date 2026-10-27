//go:build !windows

package replay

import "os"

func openReplay(path string) (*os.File, error) { return os.Open(path) }
