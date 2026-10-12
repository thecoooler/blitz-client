// Package pathx reproduces Rust Path semantics used in reports and ordering.
package pathx

import (
	"path/filepath"
	"strings"
)

// Extension returns the extension the way Rust's Path::extension does: a
// leading dot alone does not start an extension.
func Extension(path string) (string, bool) {
	name := filepath.Base(path)
	if name == ".." || name == "." || name == string(filepath.Separator) {
		return "", false
	}
	index := strings.LastIndexByte(name, '.')
	if index <= 0 {
		return "", false
	}
	return name[index+1:], true
}

// WithoutExtension removes the final extension like Path::with_extension("").
func WithoutExtension(path string) string {
	ext, ok := Extension(path)
	if !ok {
		return path
	}
	return path[:len(path)-len(ext)-1]
}

// components splits a path like Path::components, dropping empty and
// interior "." components.
func components(path string) []string {
	var out []string
	for index, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == "" || (part == "." && index > 0) {
			continue
		}
		out = append(out, part)
	}
	return out
}

// Compare orders paths component-wise like Rust's PathBuf.
func Compare(a, b string) int {
	left, right := components(a), components(b)
	for index := 0; index < len(left) && index < len(right); index++ {
		if c := strings.Compare(left[index], right[index]); c != 0 {
			return c
		}
	}
	return len(left) - len(right)
}

// HasPrefix reports whether path equals prefix or is inside it.
func HasPrefix(path, prefix string) bool {
	path, prefix = filepath.Clean(path), filepath.Clean(prefix)
	if path == prefix {
		return true
	}
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(path, prefix)
}

// Canonicalize resolves a path to an absolute path without symlinks. Like
// Rust's fs::canonicalize, it reports the on-disk letter case where the
// file system is case-insensitive.
func Canonicalize(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	return diskCase(resolved), nil
}

func isSeparator(c byte) bool { return c == '/' || c == filepath.Separator }

// Join appends elem like PathBuf::join, without cleaning the base path, so
// user-supplied prefixes such as "./" survive in reports.
func Join(base string, elems ...string) string {
	for _, elem := range elems {
		switch {
		case elem == "":
			if base != "" && !isSeparator(base[len(base)-1]) {
				base += string(filepath.Separator)
			}
		case filepath.IsAbs(elem) || base == "":
			base = elem
		case isSeparator(base[len(base)-1]):
			base += elem
		default:
			base += string(filepath.Separator) + elem
		}
	}
	return base
}

// Parent mirrors Path::parent: "" for a single relative component and for
// the root itself.
func Parent(path string) string {
	trimmed := strings.TrimRight(path, "/"+string(filepath.Separator))
	if trimmed == "" {
		return ""
	}
	index := strings.LastIndexAny(trimmed, "/"+string(filepath.Separator))
	if index < 0 {
		return ""
	}
	parent := strings.TrimRight(trimmed[:index], "/"+string(filepath.Separator))
	if parent == "" {
		return trimmed[:1]
	}
	return parent
}

// Equal compares paths component-wise like Rust's Path equality.
func Equal(a, b string) bool {
	return Compare(a, b) == 0 && strings.HasPrefix(a, "/") == strings.HasPrefix(b, "/")
}
