//go:build windows

package replay

import (
	"os"
	"syscall"
)

// fileIdentity is the NTFS file ID. Creation time is not an identity on
// Windows: file tunneling gives a file renamed over a recently replaced name
// the creation time of its predecessor.
type fileIdentity struct{ volume, high, low uint32 }

func pathIdentity(path string, _ os.FileInfo) (fileIdentity, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fileIdentity{}, &os.PathError{Op: "open", Path: path, Err: err}
	}
	// Attribute-only access with full sharing never blocks the game.
	handle, err := syscall.CreateFile(name, 0,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return fileIdentity{}, &os.PathError{Op: "open", Path: path, Err: err}
	}
	defer syscall.CloseHandle(handle)
	return identityOfHandle(handle, path)
}

func handleIdentity(file *os.File, _ os.FileInfo) (fileIdentity, error) {
	return identityOfHandle(syscall.Handle(file.Fd()), file.Name())
}

func identityOfHandle(handle syscall.Handle, path string) (fileIdentity, error) {
	var data syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(handle, &data); err != nil {
		return fileIdentity{}, &os.PathError{Op: "stat", Path: path, Err: err}
	}
	return fileIdentity{data.VolumeSerialNumber, data.FileIndexHigh, data.FileIndexLow}, nil
}
