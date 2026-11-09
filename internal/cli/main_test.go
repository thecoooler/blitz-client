package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// cliBinary is the compiled command line, exercised as a subprocess like a user would.
var cliBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "blitz-cli-test-")
	if err != nil {
		panic(err)
	}
	cliBinary = filepath.Join(dir, "blitz-client")
	if runtime.GOOS == "windows" {
		cliBinary += ".exe"
	}
	build := exec.Command("go", "build", "-o", cliBinary, "github.com/thecoooler/blitz-client/cmd/blitz-client")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type result struct {
	code           int
	stdout, stderr []byte
}

func (r result) json(t *testing.T) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(r.stdout, &value); err != nil {
		t.Fatalf("%v: %s %s", err, r.stdout, r.stderr)
	}
	return value
}

func run(t *testing.T, env []string, args ...string) result {
	t.Helper()
	command := exec.Command(cliBinary, args...)
	command.Env = append(filteredEnv("CLICOLOR_FORCE", "NO_COLOR"), env...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return result{code, stdout.Bytes(), stderr.Bytes()}
}

func filteredEnv(remove ...string) []string {
	var out []string
	for _, entry := range os.Environ() {
		keep := true
		for _, name := range remove {
			if len(entry) > len(name) && entry[:len(name)+1] == name+"=" {
				keep = false
			}
		}
		if keep {
			out = append(out, entry)
		}
	}
	return out
}

// get walks a decoded JSON value: string keys index objects, ints index arrays.
func get(value any, path ...any) any {
	for _, key := range path {
		switch typed := key.(type) {
		case string:
			object, _ := value.(map[string]any)
			value = object[typed]
		case int:
			array, _ := value.([]any)
			if typed >= len(array) {
				return nil
			}
			value = array[typed]
		}
	}
	return value
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
