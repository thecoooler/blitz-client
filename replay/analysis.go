package replay

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/pathx"
)

const (
	metaEntry       = "meta.json"
	resultsEntry    = "battle_results.dat"
	dataEntry       = "data.replay"
	replayExtension = "tbreplay"
)

// Options configures AnalyzeReplay.
type Options struct {
	// IncludeData decodes the data.replay packet stream and adds the
	// recovered telemetry.
	IncludeData bool
}

// Analysis is a decoded .tbreplay archive or standalone data.replay, whose
// metadata and results are null. JSON field names match the original replay
// CLI's schema.
type Analysis struct {
	Source        string `json:"source"`
	Meta          any    `json:"meta"`
	BattleResults any    `json:"battle_results"`
	Telemetry     any    `json:"telemetry,omitempty"`
	Data          any    `json:"data,omitempty"`
	results       *BattleResults
}

type replayData struct {
	clientVersion string
	packets       []*Packet
}

func (d *replayData) value() any {
	packets := make([]any, len(d.packets))
	for index, packet := range d.packets {
		packets[index] = packet.JSONValue()
	}
	return jsonx.Object{"client_version": d.clientVersion, "packets": packets}
}

// AnalyzeReplay reads a replay archive: meta.json, the pickled protobuf
// battle results and, when requested, the framed packet stream with the
// empirically decoded telemetry. Payloads outside the recovered layouts stay raw.
func AnalyzeReplay(path string, options Options) (*Analysis, error) {
	file, err := openReplay(path)
	if err != nil {
		return nil, blitz.IOError("cannot open replay", path, err)
	}
	defer file.Close()
	magic := make([]byte, 4)
	count, err := io.ReadFull(file, magic)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, blitz.IOError("cannot read replay", path, err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, blitz.IOError("cannot rewind replay", path, err)
	}
	if ext, ok := pathx.Extension(path); (count == 4 && binary.LittleEndian.Uint32(magic) == streamMagic) || (ok && ext == "replay") {
		return analyzeDataReplay(path, file)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, blitz.IOError("cannot read replay", path, err)
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return nil, blitz.ErrorAt("REPLAY_ARCHIVE_INVALID", "not a readable ZIP replay archive: "+zipMessage(err), path)
	}

	metaBytes, err := readEntry(archive, path, metaEntry)
	if err != nil {
		return nil, err
	}
	meta, err := jsonx.DecodeValue(metaBytes)
	if err != nil {
		return nil, blitz.ErrorAt("REPLAY_META_INVALID", fmt.Sprintf("cannot decode %s: %v", metaEntry, err), path)
	}
	resultsBytes, err := readEntry(archive, path, resultsEntry)
	if err != nil {
		return nil, err
	}
	blob, err := decodeBattleResultsDat(resultsBytes)
	if err != nil {
		return nil, blitz.ErrorAt("REPLAY_RESULTS_INVALID",
			fmt.Sprintf("cannot decode the %s pickle: REPLAY_RESULTS_INVALID: %v", resultsEntry, err), path)
	}
	results, err := decodeBattleResults(blob)
	if err != nil {
		return nil, blitz.ErrorAt("REPLAY_RESULTS_INVALID",
			fmt.Sprintf("cannot decode the battle results protobuf: REPLAY_RESULTS_INVALID: %v", err), path)
	}

	analysis := &Analysis{Source: filepath.Base(path), Meta: meta, BattleResults: jsonx.ToValue(results), results: results}
	if options.IncludeData {
		dataBytes, err := readEntry(archive, path, dataEntry)
		if err != nil {
			return nil, err
		}
		data, err := readDataReplay(bytes.NewReader(dataBytes), path)
		if err != nil {
			return nil, err
		}
		analysis.Telemetry = recoverTelemetry(data.packets, data.clientVersion, results, blob)
		analysis.Data = data.value()
	}
	return analysis, nil
}

func zipMessage(err error) string {
	if errors.Is(err, zip.ErrFormat) {
		return "invalid Zip archive: Could not find central directory end"
	}
	return err.Error()
}

func readDataReplay(reader io.Reader, path string) (*replayData, error) {
	decoder := &StreamDecoder{}
	data := &replayData{}
	chunk := make([]byte, 64*1024)
	for {
		count, err := reader.Read(chunk)
		if count > 0 {
			feedErr := decoder.Feed(chunk[:count], func(event StreamEvent) error {
				switch event.Kind {
				case EventHeader:
					data.clientVersion = event.ClientVersion
				case EventPacket:
					data.packets = append(data.packets, event.Packet)
				}
				return nil
			})
			if feedErr != nil {
				return nil, feedErr
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, blitz.IOError("cannot read data.replay", path, err)
		}
	}
	if err := decoder.Finish(); err != nil {
		return nil, err
	}
	return data, nil
}

func analyzeDataReplay(path string, file *os.File) (*Analysis, error) {
	data, err := readDataReplay(file, path)
	if err != nil {
		return nil, err
	}
	return &Analysis{
		Source:    filepath.Base(path),
		Telemetry: recoverTelemetry(data.packets, data.clientVersion, nil, nil),
		Data:      data.value(),
	}, nil
}

// FindLatestReplay returns the newest .tbreplay file in directory, or in the
// standard OS folder when directory is empty.
func FindLatestReplay(directory string) (string, error) {
	if directory == "" {
		var err error
		if directory, err = DefaultReplayDir(); err != nil {
			return "", err
		}
	}
	handle, err := os.Open(directory)
	if err != nil {
		return "", blitz.IOError("cannot scan the replay folder", directory, err)
	}
	entries, err := handle.ReadDir(-1)
	handle.Close()
	if err != nil {
		return "", blitz.IOError("cannot read a replay folder entry", directory, err)
	}
	var latest string
	var latestTime time.Time
	found := false
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		if ext, ok := pathx.Extension(path); !ok || ext != replayExtension {
			continue
		}
		modified := time.Unix(0, 0)
		if info, err := entry.Info(); err == nil {
			modified = info.ModTime()
		}
		if !found || modified.After(latestTime) {
			latest, latestTime, found = path, modified, true
		}
	}
	if !found {
		return "", blitz.ErrorAt("REPLAY_NOT_FOUND", "no ."+replayExtension+" files found", directory)
	}
	return latest, nil
}

func readEntry(archive *zip.Reader, path, name string) ([]byte, error) {
	var entry *zip.File
	for _, file := range archive.File {
		if file.Name == name {
			entry = file
			break
		}
	}
	if entry == nil {
		return nil, blitz.ErrorAt("REPLAY_ENTRY_MISSING",
			fmt.Sprintf("replay archive is missing %s (specified file not found in archive)", name), path)
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, blitz.IOError("cannot read "+name, path, err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, blitz.IOError("cannot read "+name, path, err)
	}
	return data, nil
}
