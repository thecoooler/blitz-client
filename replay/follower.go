package replay

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"

	blitz "github.com/thecoooler/blitz-client"
)

// FollowEvent is one NDJSON record of a followed replay.
type FollowEvent struct {
	SessionID uint64
	Event     StreamEvent
	// Telemetry is set for packets; it is nil (JSON null) for unsupported
	// client versions.
	Telemetry    any
	HasTelemetry bool
}

// MarshalJSON flattens the stream event into the record.
func (e *FollowEvent) MarshalJSON() ([]byte, error) {
	fields := append(orderedObject{{"session_id", e.SessionID}}, e.Event.fields()...)
	if e.HasTelemetry {
		fields = append(fields, field{"telemetry", e.Telemetry})
	}
	return fields.MarshalJSON()
}

// streamFollower polls a growing data.replay for raw stream events.
// Callers control scheduling, cancellation, and output. Missing files
// are awaited; replacement or observed truncation starts a new session.
// Like tail, it expects append-only writes within each session.
type streamFollower struct {
	path       string
	file       *os.File
	identity   *fileIdentity
	readOffset int64
	anchor     []byte
	sessionID  uint64
	decoder    *StreamDecoder
}

// newStreamFollower returns a raw stream follower for path.
func newStreamFollower(path string) *streamFollower {
	return &streamFollower{path: path, sessionID: 1, decoder: &StreamDecoder{}}
}

// Close releases the open file handle.
func (f *streamFollower) Close() error {
	if f.file != nil {
		err := f.file.Close()
		f.file = nil
		return err
	}
	return nil
}

func (f *streamFollower) reset(reason string, emit func(FollowEvent) error) error {
	f.Close()
	f.identity = nil
	f.readOffset = 0
	f.anchor = nil
	f.sessionID++
	f.decoder = &StreamDecoder{}
	return emit(FollowEvent{SessionID: f.sessionID, Event: StreamEvent{Kind: EventReset, Reason: reason}})
}

// Poll reads newly appended bytes and returns their count. EOF leaves
// incomplete packets pending.
func (f *streamFollower) Poll(emit func(FollowEvent) error) (int64, error) {
	info, err := os.Stat(f.path)
	var identity fileIdentity
	if err == nil {
		if !info.Mode().IsRegular() {
			return 0, dataInvalid("live replay path is not a regular file")
		}
		identity, err = pathIdentity(f.path, info)
	}
	if errors.Is(err, fs.ErrNotExist) {
		if f.identity != nil {
			if err := f.reset("file_removed", emit); err != nil {
				return 0, err
			}
		}
		return 0, nil
	}
	if err != nil {
		return 0, blitz.IOError("cannot stat live replay", f.path, err)
	}
	if f.identity != nil && *f.identity != identity {
		if err := f.reset("file_replaced", emit); err != nil {
			return 0, err
		}
	} else if info.Size() < f.readOffset {
		if err := f.reset("file_truncated", emit); err != nil {
			return 0, err
		}
	}
	if f.file == nil {
		file, err := openReplay(f.path)
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		if err != nil {
			return 0, blitz.IOError("cannot open live replay", f.path, err)
		}
		handleInfo, err := file.Stat()
		if err != nil {
			file.Close()
			return 0, blitz.IOError("cannot stat live replay handle", f.path, err)
		}
		opened, err := handleIdentity(file, handleInfo)
		if err != nil {
			file.Close()
			return 0, blitz.IOError("cannot stat live replay handle", f.path, err)
		}
		f.identity = &opened
		f.file = file
	}
	// Do not retain a handle between polls: the game finalizes recordings by
	// renaming/removing files and their containing directory on Windows.
	defer f.Close()
	// Detect a truncate-and-regrow between polls, even if the size has
	// already caught up. Checking the last read bytes is cheap and avoids
	// silently combining frames from two battles in the usual rewrite case.
	if len(f.anchor) > 0 {
		anchor := make([]byte, len(f.anchor))
		_, err := f.file.ReadAt(anchor, f.readOffset-int64(len(f.anchor)))
		rewritten := false
		switch {
		case err == nil:
			rewritten = !bytes.Equal(anchor, f.anchor)
		case errors.Is(err, io.EOF):
			rewritten = true
		default:
			return 0, blitz.IOError("cannot read replay anchor", f.path, err)
		}
		if rewritten {
			// Open again on the next poll, after any concurrent rewrite.
			return 0, f.reset("file_rewritten", emit)
		}
	}
	handleInfo, err := f.file.Stat()
	if err != nil {
		return 0, blitz.IOError("cannot stat live replay handle", f.path, err)
	}
	remaining := max(handleInfo.Size()-f.readOffset, 0)
	var readBytes int64
	chunk := make([]byte, 64*1024)
	for remaining > 0 {
		count, err := f.file.ReadAt(chunk[:min(remaining, int64(len(chunk)))], f.readOffset)
		if err != nil && !errors.Is(err, io.EOF) {
			return readBytes, blitz.IOError("cannot read live replay", f.path, err)
		}
		if count == 0 {
			break
		}
		f.anchor = append(f.anchor, chunk[:count]...)
		if len(f.anchor) > 64 {
			f.anchor = append([]byte(nil), f.anchor[len(f.anchor)-64:]...)
		}
		f.readOffset += int64(count)
		remaining -= int64(count)
		readBytes += int64(count)
		sessionID := f.sessionID
		err = f.decoder.Feed(chunk[:count], func(event StreamEvent) error {
			return emit(FollowEvent{SessionID: sessionID, Event: event})
		})
		if err != nil {
			return readBytes, err
		}
	}
	return readBytes, nil
}

// Follower polls a growing data.replay. Callers control scheduling,
// cancellation, and output. Missing files are awaited; replacement or
// observed truncation starts a new session. Like tail, it expects
// append-only writes within each session.
type Follower struct {
	stream    *streamFollower
	telemetry *TelemetryDecoder
}

// NewFollower returns an enriched reporting follower for path.
func NewFollower(path string) *Follower {
	return &Follower{
		stream:    newStreamFollower(path),
		telemetry: NewTelemetryDecoder(),
	}
}

// Close releases the open file handle.
func (f *Follower) Close() error {
	return f.stream.Close()
}

// Poll reads newly appended bytes, enriches packet events with reporting
// telemetry, and returns their count. EOF leaves incomplete packets pending.
func (f *Follower) Poll(emit func(FollowEvent) error) (int64, error) {
	return f.stream.Poll(func(event FollowEvent) error {
		switch event.Event.Kind {
		case EventReset:
			f.telemetry = NewTelemetryDecoder()
		case EventPacket:
			event.Telemetry = f.telemetry.OnPacket(event.Event.Packet, event.Event.Seq, f.stream.decoder.ClientVersion())
			event.HasTelemetry = true
		}
		return emit(event)
	})
}
