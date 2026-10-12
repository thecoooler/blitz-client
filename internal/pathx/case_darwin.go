package pathx

import (
	"bytes"
	"os"
	"syscall"
	"unsafe"
)

const fGetPath = 50 // F_GETPATH

// diskCase asks the kernel for the stored path, which carries the on-disk
// letter case on case-insensitive volumes.
func diskCase(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return path
	}
	defer file.Close()
	buffer := make([]byte, 1024) // MAXPATHLEN
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, file.Fd(), fGetPath, uintptr(unsafe.Pointer(&buffer[0])))
	if errno != 0 {
		return path
	}
	if end := bytes.IndexByte(buffer, 0); end > 0 {
		return string(buffer[:end])
	}
	return path
}
