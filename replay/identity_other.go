//go:build !unix && !windows

package replay

import "os"

// fileIdentity is unavailable here; replacement is detected by content only.
type fileIdentity struct{}

func pathIdentity(string, os.FileInfo) (fileIdentity, error) { return fileIdentity{}, nil }

func handleIdentity(*os.File, os.FileInfo) (fileIdentity, error) { return fileIdentity{}, nil }
