//go:build unix

package replay

import (
	"os"
	"syscall"
)

type fileIdentity struct{ dev, ino uint64 }

func identityOf(info os.FileInfo) fileIdentity {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return fileIdentity{uint64(stat.Dev), uint64(stat.Ino)}
	}
	return fileIdentity{}
}
