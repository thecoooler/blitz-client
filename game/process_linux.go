package game

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const taskCommLen = 15 // TASK_COMM_LEN without the terminator

// findProcesses scans /proc. Under Wine and Proton the first argument is the
// Windows path of the game executable, while comm holds its truncated name.
func findProcesses(names matcher) ([]process, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var out []process
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		dir := filepath.Join("/proc", entry.Name())
		comm, err := os.ReadFile(filepath.Join(dir, "comm"))
		if err != nil || !names.matchTruncated(strings.TrimSuffix(string(comm), "\n"), taskCommLen) {
			continue
		}
		executable := ""
		if cmdline, err := os.ReadFile(filepath.Join(dir, "cmdline")); err == nil {
			if end := bytes.IndexByte(cmdline, 0); end >= 0 {
				cmdline = cmdline[:end]
			}
			executable = string(cmdline)
		}
		if executable == "" {
			executable, _ = os.Readlink(filepath.Join(dir, "exe"))
		}
		if executable != "" && !names.match(executable) {
			continue
		}
		out = append(out, process{pid: pid, executable: executable})
	}
	return out, nil
}
