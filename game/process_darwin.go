package game

import (
	"bytes"

	"golang.org/x/sys/unix"
)

const maxComLen = 16 // MAXCOMLEN: p_comm keeps at most 16 bytes

func findProcesses(names matcher) ([]process, error) {
	procs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}
	var out []process
	for i := range procs {
		pid := int(procs[i].Proc.P_pid)
		comm := procs[i].Proc.P_comm[:]
		if end := bytes.IndexByte(comm, 0); end >= 0 {
			comm = comm[:end]
		}
		if pid <= 0 || !names.matchTruncated(string(comm), maxComLen) {
			continue
		}
		path := executablePath(pid)
		if path != "" && !names.match(path) {
			continue // a truncated name that belongs to another program
		}
		out = append(out, process{pid: pid, executable: path})
	}
	return out, nil
}

// executablePath reads the executable path that starts KERN_PROCARGS2 after
// the argument count. Other users' processes are not readable.
func executablePath(pid int) string {
	data, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil || len(data) < 4 {
		return ""
	}
	data = data[4:]
	if end := bytes.IndexByte(data, 0); end >= 0 {
		data = data[:end]
	}
	return string(data)
}
