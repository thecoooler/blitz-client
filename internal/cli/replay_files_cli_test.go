package cli

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/thecoooler/blitz-client/resources"
)

func TestLocateReturnsReplayAndCachePathsEvenBeforeFolderCreation(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS paths")
	}
	temp := t.TempDir()
	home, game := filepath.Join(temp, "home"), filepath.Join(temp, "game")
	os.Mkdir(game, 0o755)
	output := run(t, []string{"HOME=" + home}, "client", "locate", "--game-dir", game, "--json")
	value := output.json(t)
	replayDir := filepath.Join(home, "Library/Containers/com.tanksblitz/Data/Documents/tb-project/replays")
	cacheDir := filepath.Join(home, "Library/Containers/com.tanksblitz/Data/Library/Caches/packs")
	canonical, _ := filepath.EvalSymlinks(game)
	if output.code != 0 || exists(replayDir) || exists(cacheDir) || value["replay_dir"] != replayDir ||
		value["cache_dir"] != cacheDir || value["source"] != "explicit" || value["game_dir"] != canonical {
		t.Fatalf("%s", output.stdout)
	}
}

func TestFollowFlushesBeforeExitAndWaitsForPartialPackets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.replay")
	command := exec.Command(cliBinary, "replay", "follow", path, "--poll-ms", "5", "--idle-timeout-ms", "1000", "--json")
	stdout, _ := command.StdoutPipe()
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	lines := make(chan string, 16)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(nil, 1<<20)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	next := func(timeout time.Duration) (map[string]any, bool) {
		select {
		case line, ok := <-lines:
			if !ok {
				return nil, false
			}
			var value map[string]any
			json.Unmarshal([]byte(line), &value)
			return value, true
		case <-time.After(timeout):
			return nil, false
		}
	}
	writer, _ := os.Create(path)
	defer writer.Close()
	header := binary.LittleEndian.AppendUint32(nil, 0x12345678)
	header = binary.LittleEndian.AppendUint64(header, 0)
	writer.Write(append(header, "\x00\x01v\x00"...))
	if event, ok := next(5 * time.Second); !ok || event["event"] != "header" {
		t.Fatal(event)
	}
	packet := binary.LittleEndian.AppendUint32(nil, 3)
	packet = binary.LittleEndian.AppendUint32(packet, 99)
	packet = binary.LittleEndian.AppendUint32(packet, math.Float32bits(0.5))
	packet = append(packet, 1, 2, 3)
	writer.Write(packet[:13])
	if event, ok := next(50 * time.Millisecond); ok {
		t.Fatal("partial packet emitted", event)
	}
	writer.Write(packet[13:])
	event, ok := next(5 * time.Second)
	if !ok || event["event"] != "packet" || event["seq"] != 0.0 || event["raw_payload"] != "010203" {
		t.Fatal(event)
	}
	if err := command.Wait(); err != nil || stderr.Len() > 0 {
		t.Fatal(err, stderr.String())
	}
	if _, ok := next(0); ok {
		t.Fatal("unexpected extra output")
	}
}

func TestFollowProtectsSourceAndKeepsErrorsOffStdout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.replay")
	if result := run(t, nil, "replay", "follow", path, "--out", path); result.code == 0 || exists(path) {
		t.Fatal("source used as output")
	}
	os.WriteFile(path, make([]byte, 4), 0o644)
	result := run(t, nil, "replay", "follow", path, "--json")
	if result.code == 0 || len(result.stdout) > 0 || !strings.Contains(string(result.stderr), "REPLAY_DATA_INVALID") {
		t.Fatalf("%d %s %s", result.code, result.stdout, result.stderr)
	}
	if result := run(t, nil, "replay", "follow", path, "--poll-ms", "0"); result.code == 0 || readFile(t, path) != "\x00\x00\x00\x00" {
		t.Fatal("invalid poll interval accepted")
	}
}

type filesFixture struct{ root string }

func newFilesFixture(t *testing.T) filesFixture {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "game/Data"), 0o755)
	os.MkdirAll(filepath.Join(root, "packs"), 0o755)
	return filesFixture{root}
}

func (f filesFixture) path(relative string) string { return filepath.Join(f.root, relative) }

func TestCLIExportsSelectedEffectiveFilesAndEmitsOneJSONReport(t *testing.T) {
	f := newFilesFixture(t)
	writeFile(t, f.path("game/Data/config.yaml"), "base")
	packed, _ := resources.EncodeDVPL([]byte("cached config"), resources.DVPL(1))
	os.WriteFile(f.path("packs/config.yaml.dvpl"), packed, 0o644)
	writeFile(t, f.path("game/Data/tank.scg"), "model")
	writeFile(t, f.path("game/Data/image.webp"), "image")
	export := func() result {
		return run(t, nil, "client", "files", "unpack", "--game-dir", f.path("game"), "--cache-dir", f.path("packs"),
			"--output-dir", f.path("output"), "--types", "configs,images", "--json")
	}
	first := export()
	report := first.json(t)
	if first.code != 0 || len(first.stderr) > 0 || report["file_count"] != 2.0 || report["cache_file_count"] != 1.0 ||
		report["written_count"] != 2.0 || readFile(t, f.path("output/config.yaml")) != "cached config" ||
		exists(f.path("output/config.yaml.dvpl")) || exists(f.path("output/tank.scg")) {
		t.Fatalf("%s %s", first.stdout, first.stderr)
	}
	if again := export().json(t); again["unchanged_count"] != 2.0 || again["written_count"] != 0.0 {
		t.Fatal(again)
	}
}

func TestInvalidCLIOptionsFailBeforeCreatingOutput(t *testing.T) {
	f := newFilesFixture(t)
	output := f.path("output")
	for _, args := range [][]string{
		{"--types", "configs,unknown"}, {"--types", "configs,"}, {"--types"}, {"--exclude"},
		{"--cache-dir", ".", "--no-cache"}, {"--unknown"},
	} {
		full := append(append([]string{"client", "files", "unpack", "--output-dir", output}, args...), "--json")
		result := run(t, nil, full...)
		if result.code == 0 || get(result.json(t), "error", "code") != "USAGE" || exists(output) {
			t.Fatalf("%v: %s", args, result.stdout)
		}
	}
}

func TestDryRunCountsStoredEffectiveFilesWithoutDecodingOrWriting(t *testing.T) {
	f := newFilesFixture(t)
	output := f.path("new/output")
	writeFile(t, f.path("game/Data/config.yaml"), "shadowed")
	// Intentionally invalid DVPL: dry-run must inspect metadata only.
	writeFile(t, f.path("packs/config.yaml.dvpl"), "packed")
	writeFile(t, f.path("game/Data/note.txt"), "text")
	writeFile(t, f.path("game/Data/image.webp"), "excluded")
	writeFile(t, f.path("game/Data/note.txt.dvpl"), "shadowed sibling")
	dryRun := func(json bool) result {
		args := []string{"client", "files", "unpack", "--game-dir", f.path("game"), "--cache-dir", f.path("packs"),
			"--output-dir", output, "--types", "configs", "--dry-run"}
		if json {
			args = append(args, "--json")
		}
		return run(t, nil, args...)
	}
	result := dryRun(true)
	report := result.json(t)
	if result.code != 0 || len(result.stderr) > 0 || report["dry_run"] != true || report["file_count"] != 2.0 ||
		report["source_bytes"] != 10.0 || report["dvpl_count"] != 1.0 || report["cache_file_count"] != 1.0 {
		t.Fatalf("%s", result.stdout)
	}
	for _, field := range []string{"written_count", "unchanged_count", "output_bytes"} {
		if report[field] != 0.0 {
			t.Fatal(field)
		}
	}
	if exists(f.path("new")) {
		t.Fatal("dry run created output")
	}
	writeFile(t, filepath.Join(output, "config.yaml"), "keep existing")
	text := dryRun(false)
	entries, _ := os.ReadDir(output)
	if text.code != 0 || len(text.stderr) > 0 || !strings.Contains(string(text.stdout), "2 file(s), 10 bytes") ||
		readFile(t, filepath.Join(output, "config.yaml")) != "keep existing" || len(entries) != 1 {
		t.Fatalf("%s", text.stdout)
	}
}

func TestCLIAcceptsRepeatedExclusionsForExportAndDryRun(t *testing.T) {
	f := newFilesFixture(t)
	output := f.path("output")
	writeFile(t, f.path("game/Data/Gfx/image.webp.dvpl"), "invalid DVPL")
	writeFile(t, f.path("game/Data/skip.txt"), "skip")
	writeFile(t, f.path("game/Data/keep.yaml"), "keep")
	for _, dryRun := range []bool{true, false} {
		args := []string{"client", "files", "unpack", "--game-dir", f.path("game"), "--output-dir", output,
			"--no-cache", "--exclude", "Gfx/**", "--exclude", "skip.txt", "--json"}
		if dryRun {
			args = append(args, "--dry-run")
		}
		result := run(t, nil, args...)
		report := result.json(t)
		if result.code != 0 || report["excluded_file_count"] != 2.0 || !reflect.DeepEqual(report["exclude"], []any{"Gfx/**", "skip.txt"}) ||
			report["file_count"] != 1.0 || report["source_bytes"] != 4.0 || (dryRun && exists(output)) {
			t.Fatalf("%s %s", result.stdout, result.stderr)
		}
	}
	entries, _ := os.ReadDir(output)
	if readFile(t, filepath.Join(output, "keep.yaml")) != "keep" || len(entries) != 1 {
		t.Fatal("unexpected output")
	}
}
