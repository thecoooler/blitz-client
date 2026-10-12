// Package blitzclient contains the error and path types shared by the client,
// export, mods, replay, and resources packages.
package blitzclient

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"
	"syscall"
)

// Error is the stable, serializable error shape for the CLI, IPC, and other Go
// callers. Every error returned by this module's packages is an *Error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

// NewError returns an error without a path.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Errorf returns an error without a path, formatting its message.
func Errorf(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// ErrorAt returns an error associated with a filesystem path.
func ErrorAt(code, message, path string) *Error {
	return &Error{Code: code, Message: message, Path: path}
}

// IOError wraps an operating-system error with context and a path.
func IOError(context, path string, err error) *Error {
	return ErrorAt("IO_ERROR", context+": "+IOMessage(err), path)
}

// IOMessage formats an operating-system error like Rust's io::Error Display
// ("No such file or directory (os error 2)"), without the operation and path
// prefix Go adds to *os.PathError.
func IOMessage(err error) string {
	if errors.Is(err, exec.ErrNotFound) || err == fs.ErrNotExist {
		err = syscall.ENOENT
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		text := errno.Error()
		if text != "" {
			text = strings.ToUpper(text[:1]) + text[1:]
		}
		return fmt.Sprintf("%s (os error %d)", text, int(errno))
	}
	var wrapped interface{ Unwrap() error }
	if errors.As(err, &wrapped) {
		if inner := wrapped.Unwrap(); inner != nil {
			return inner.Error()
		}
	}
	return err.Error()
}

func (e *Error) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Path)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// AsError converts any error into an *Error, using INTERNAL_ERROR for errors
// that did not originate in this module.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var typed *Error
	if errors.As(err, &typed) {
		return typed
	}
	return NewError("INTERNAL_ERROR", err.Error())
}
