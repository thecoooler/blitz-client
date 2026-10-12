package mods

import (
	"bytes"
	"os/exec"
	"strings"
	"unicode/utf8"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/lossy"
	"github.com/thecoooler/blitz-client/internal/rs"
)

func collectChanges(repo, base string) ([]string, error) {
	if !isDir(repo) {
		return nil, blitz.ErrorAt("GIT_REPO_NOT_FOUND", "repository directory does not exist", repo)
	}
	if base == "" || strings.HasPrefix(base, "-") {
		return nil, blitz.NewError("GIT_REVISION_INVALID", "base revision must be a non-empty revision name")
	}
	status, err := runGit(repo, "diff", "--name-status", "-z", "--no-renames", base, "--")
	if err != nil {
		return nil, err
	}
	var fields [][]byte
	for _, part := range bytes.Split(status, []byte{0}) {
		if len(part) > 0 {
			fields = append(fields, part)
		}
	}
	var eligible, unsupported []string
	for index := 0; index+1 < len(fields); index += 2 {
		state := ""
		if utf8.Valid(fields[index]) {
			state = string(fields[index])
		}
		if !utf8.Valid(fields[index+1]) {
			return nil, blitz.NewError("GIT_PATH_NOT_UTF8", "repository contains a modified path that is not UTF-8")
		}
		path := string(fields[index+1])
		switch {
		case path == "mod.yaml" || path == "mod.yml" || path == "mod.patch" || strings.HasPrefix(path, "patches/"):
		case state == "M":
			eligible = append(eligible, path)
		default:
			unsupported = append(unsupported, state+" "+path)
		}
	}
	if len(unsupported) > 0 {
		return nil, blitz.NewError("GIT_DIFF_UNSUPPORTED_CHANGE",
			"only modifications to existing tracked files are supported; found "+strings.Join(unsupported, ", "))
	}

	var changes []string
	for _, path := range eligible {
		if strings.HasSuffix(path, ".dvpl") {
			return nil, blitz.ErrorAt("GIT_DIFF_DVPL_INPUT",
				"generate diffs from unpacked text resources, not compressed .dvpl containers", path)
		}
		output, err := runGit(repo, "diff", "--no-ext-diff", "--no-textconv", "--no-color", "--no-renames",
			"--unified=3", "--src-prefix=a/", "--dst-prefix=b/", base, "--", path)
		if err != nil {
			return nil, err
		}
		if !utf8.Valid(output) {
			return nil, blitz.ErrorAt("GIT_DIFF_NOT_UTF8", "git produced a non-UTF-8 text diff", path)
		}
		diff := string(output)
		for _, line := range rs.Lines(diff) {
			if strings.HasPrefix(line, "Binary files ") || strings.HasPrefix(line, "GIT binary patch") {
				return nil, blitz.ErrorAt("GIT_DIFF_BINARY_FILE", "binary changes cannot be converted to a text patch", path)
			}
		}
		for _, line := range rs.Lines(diff) {
			if strings.HasPrefix(line, "old mode ") || strings.HasPrefix(line, "new mode ") {
				return nil, blitz.ErrorAt("GIT_DIFF_UNSUPPORTED_CHANGE", "file mode changes are unsupported", path)
			}
		}
		if !strings.Contains(diff, "@@ ") {
			continue
		}
		changes = append(changes, diff)
	}
	return changes, nil
}

func runGit(repo string, args ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-c", "core.quotePath=false", "-C", repo}, args...)...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Start(); err != nil {
		return nil, blitz.NewError("GIT_UNAVAILABLE", "could not start git: "+blitz.IOMessage(err))
	}
	if err := command.Wait(); err != nil {
		if _, exited := err.(*exec.ExitError); !exited {
			return nil, blitz.NewError("GIT_UNAVAILABLE", "could not start git: "+err.Error())
		}
		detail := strings.TrimSpace(lossy.String(stderr.Bytes()))
		if detail == "" {
			detail = "git diff failed"
		}
		return nil, blitz.NewError("GIT_COMMAND_FAILED", detail)
	}
	return stdout.Bytes(), nil
}
