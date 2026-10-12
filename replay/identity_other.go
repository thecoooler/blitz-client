//go:build !unix

package replay

import (
	"os"
	"syscall"
)

// fileIdentity uses the creation time where device/inode numbers are unavailable.
type fileIdentity struct{ created int64 }

func identityOf(info os.FileInfo) fileIdentity {
	if data, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return fileIdentity{data.CreationTime.Nanoseconds()}
	}
	return fileIdentity{}
}
