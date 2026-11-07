package cli

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/pathx"
	"github.com/thecoooler/blitz-client/replay"
)

func resolveFilePath(value string) (string, error) {
	parent := pathx.Parent(value)
	if parent == "" {
		parent = "."
	}
	canonical, err := pathx.Canonicalize(parent)
	if err != nil {
		return "", blitz.ErrorAt("IO_ERROR", "cannot resolve replay path: "+blitz.IOMessage(err), value)
	}
	name := filepath.Base(value)
	if name == "." || name == ".." || strings.HasSuffix(value, "..") {
		return "", usageError("expected a file path")
	}
	return filepath.Join(canonical, name), nil
}

func (c ReplayFollowCmd) Run(a *app) error {
	if c.PollMS == 0 {
		return usageError("--poll-ms requires milliseconds between 1 and 86400000")
	}
	path, out := c.Path, c.Out
	pollMS := uint64(c.PollMS)
	var idleTimeout time.Duration
	if c.IdleTimeoutMS > 0 {
		idleTimeout = time.Duration(c.IdleTimeoutMS) * time.Millisecond
	}
	var output io.Writer = a.stdout
	if out != nil {
		source, err := resolveFilePath(path)
		if err != nil {
			return err
		}
		target, err := resolveFilePath(*out)
		if err != nil {
			return err
		}
		if source == target {
			return usageError("--out must differ from the live replay path")
		}
		// Create exclusively: never truncate the source or an existing output file.
		file, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if err != nil {
			return blitz.ErrorAt("IO_ERROR", "cannot create NDJSON output: "+blitz.IOMessage(err), *out)
		}
		defer file.Close()
		output = file
	}
	writer := bufio.NewWriter(output)
	follower := replay.NewFollower(path)
	defer follower.Close()
	lastActivity := time.Now()
	for {
		reset := false
		count, err := follower.Poll(func(event replay.FollowEvent) error {
			reset = reset || event.Event.Kind == replay.EventReset
			line, err := jsonx.Marshal(&event)
			if err != nil {
				return blitz.NewError("JSON_SERIALIZE_FAILED", err.Error())
			}
			writer.Write(line)
			writer.WriteByte('\n')
			if err := writer.Flush(); err != nil {
				return blitz.NewError("REPLAY_OUTPUT_FAILED", blitz.IOMessage(err))
			}
			return nil
		})
		if err != nil {
			return err
		}
		if count > 0 || reset {
			lastActivity = time.Now()
		}
		if idleTimeout > 0 && time.Since(lastActivity) >= idleTimeout {
			return nil
		}
		wait := time.Duration(pollMS) * time.Millisecond
		if idleTimeout > 0 {
			wait = min(wait, max(idleTimeout-time.Since(lastActivity), 0))
		}
		time.Sleep(wait)
	}
}

func (c ReplayInspectCmd) Run(a *app) error {
	return a.replayAnalysis(c.Path, "", false, c.Data, c.Out)
}

func (c ReplayLatestCmd) Run(a *app) error {
	return a.replayAnalysis("", derefFlag(c.Dir), c.Dir != nil, c.Data, c.Out)
}

func (a *app) replayAnalysis(path, directory string, explicitDir, includeData bool, out *string) error {
	var target string
	if explicitDir {
		var err error
		if target, err = findLatest(directory); err != nil {
			return err
		}
	} else {
		target = path
	}
	analysis, err := replay.AnalyzeReplay(target, replay.Options{IncludeData: includeData})
	if err != nil {
		return err
	}
	if out != nil {
		data, err := jsonx.MarshalIndent(analysis)
		if err != nil {
			return blitz.NewError("JSON_SERIALIZE_FAILED", err.Error())
		}
		if err := os.WriteFile(*out, data, 0o666); err != nil {
			return blitz.ErrorAt("IO_ERROR", "cannot write the replay JSON: "+blitz.IOMessage(err), *out)
		}
	}
	if a.json {
		a.printJSON(analysis)
	} else {
		a.printReplaySummary(analysis)
		if out != nil {
			a.printf("json: %s\n", *out)
		}
	}
	return nil
}

func findLatest(directory string) (string, error) {
	if directory == "" {
		// An empty explicit directory cannot be scanned.
		return "", blitz.IOError("cannot scan the replay folder", "", os.ErrNotExist)
	}
	return replay.FindLatestReplay(directory)
}

// field formats a JSON member like the original summary: strings raw, null
// as "?", and other values as compact JSON.
func field(value any, key string) string {
	object, _ := value.(jsonx.Object)
	switch item := object[key].(type) {
	case nil:
		return "?"
	case string:
		return item
	default:
		data, _ := jsonx.Marshal(item)
		return string(data)
	}
}

func asFloat(value any) float64 {
	switch number := value.(type) {
	case jsonx.F64:
		return float64(number)
	case interface{ Float64() (float64, error) }:
		parsed, _ := number.Float64()
		return parsed
	}
	return 0
}

func (a *app) printReplaySummary(analysis *replay.Analysis) {
	meta, _ := analysis.Meta.(jsonx.Object)
	results, _ := analysis.BattleResults.(jsonx.Object)
	a.println(analysis.Source)
	client := field(analysis.Meta, "version")
	if analysis.Data != nil {
		client = field(analysis.Data, "client_version")
	}
	a.printf("client: %s\n", client)
	if analysis.Meta != nil {
		author := results["author"]
		a.printf("map: %s (id %s), vehicle: %s\n", field(meta, "mapName"), field(meta, "mapId"), field(meta, "playerVehicleName"))
		a.printf("author: %s (dbid %s, damage %s, shots %s, hits %s, hp left %s)\n",
			field(meta, "playerName"), field(meta, "dbid"), field(author, "damage_dealt"),
			field(author, "n_shots"), field(author, "n_hits"), field(author, "hitpoints_left"))
		players := 0
		if list, ok := results["players"].([]any); ok {
			players = len(list)
		}
		a.printf("battle: %.1fs, arena %s, winner team %s, players %d\n",
			asFloat(meta["battleDuration"]), field(meta, "arenaUniqueId"), field(results, "winner_team_number"), players)
	}
	if analysis.Telemetry == nil {
		return
	}
	if data, ok := analysis.Data.(jsonx.Object); ok {
		if packets, ok := data["packets"].([]any); ok {
			a.printf("packets: %d\n", len(packets))
		}
	}
	telemetry := analysis.Telemetry
	a.printf("decoder: %s\n", field(telemetry, "decoder"))
	summary := telemetry.(jsonx.Object)["summary"]
	if supported, _ := summary.(jsonx.Object)["packet_layout_decoding_supported"].(bool); !supported {
		a.printf("telemetry: packet layouts are verified only for %s\n", field(telemetry, "supported_client_version"))
		return
	}
	a.printf("telemetry: shots %s/%s, hits %s/%s, health updates %s, transform updates %s, results snapshots %s\n",
		field(summary, "projectile_position_packets"), field(summary, "shots_in_battle_results"),
		field(summary, "hit_messages"), field(summary, "hits_received_in_battle_results"),
		field(summary, "vehicle_health_updates"), field(summary, "vehicle_transform_updates"),
		field(summary, "battle_results_snapshot_packets"))
}
