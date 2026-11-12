package blitzclient

import (
	"io/fs"
	"strings"
	"testing"
)

func TestIOErrorUsesStablePermissionCode(t *testing.T) {
	err := IOError("write failed", "/game/Data/file", &fs.PathError{Op: "open", Path: "/game/Data/file", Err: fs.ErrPermission})
	if err.Code != "IO_PERMISSION_DENIED" {
		t.Fatalf("code = %q", err.Code)
	}
	if err.Path != "/game/Data/file" {
		t.Fatalf("path = %q", err.Path)
	}
	if !strings.Contains(err.Message, "write failed") {
		t.Fatalf("message = %q", err.Message)
	}
}

func TestIOErrorKeepsGenericCodeForOtherFailures(t *testing.T) {
	err := IOError("read failed", "file", &fs.PathError{Op: "open", Path: "file", Err: fs.ErrNotExist})
	if err.Code != "IO_ERROR" {
		t.Fatalf("code = %q", err.Code)
	}
}
