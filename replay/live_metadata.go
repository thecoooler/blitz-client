package replay

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	maxLiveMetadataBytes = 1024 * 1024
	// A bounded refresh also catches rewrites hidden by coarse or restored mtimes
	// and replacement on platforms without a reliable file identity.
	metadataRefreshInterval = 2 * time.Second
)

// A metadata file may be absent or still being written. Cache only a complete,
// useful document and invalidate it when the file's identity or attributes
// change. Failed reads and incomplete documents are retried on the next poll.
type liveMetadataCache struct {
	path   string
	stamp  metadataStamp
	entry  *liveMetadata
	readAt time.Time
}

type metadataStamp struct {
	identity fileIdentity
	size     int64
	modified int64
	mode     os.FileMode
}

type liveMetadata struct {
	arenaID uint64
	mapID   uint32
	mapName string
}

func metadataStampForPath(path string) (metadataStamp, error) {
	info, err := os.Stat(path)
	if err != nil {
		return metadataStamp{}, err
	}
	identity, err := pathIdentity(path, info)
	if err != nil {
		return metadataStamp{}, err
	}
	return metadataStampFromInfo(info, identity), nil
}

func metadataStampFromInfo(info os.FileInfo, identity fileIdentity) metadataStamp {
	return metadataStamp{identity: identity, size: info.Size(), modified: info.ModTime().UnixNano(), mode: info.Mode()}
}

func (c *liveMetadataCache) snapshot(directory string, arenaID uint64) *LiveBattleMap {
	if arenaID == 0 {
		return nil
	}
	path := filepath.Join(directory, metaEntry)
	stamp, err := metadataStampForPath(path)
	if err != nil || !stamp.mode.IsRegular() || stamp.size > maxLiveMetadataBytes {
		*c = liveMetadataCache{}
		return nil
	}
	if c.entry == nil || c.path != path || c.stamp != stamp || time.Since(c.readAt) >= metadataRefreshInterval {
		*c = liveMetadataCache{}
		entry := readLiveMetadata(path, stamp)
		if entry == nil {
			return nil
		}
		c.path, c.stamp, c.entry, c.readAt = path, stamp, entry, time.Now()
	}
	if c.entry.arenaID != arenaID {
		return nil
	}
	result := &LiveBattleMap{Name: c.entry.mapName}
	if c.entry.mapID != 0 {
		id := c.entry.mapID
		result.ID = &id
	}
	return result
}

func readLiveMetadata(path string, expected metadataStamp) *liveMetadata {
	f, err := openReplay(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil
	}
	identity, err := handleIdentity(f, info)
	if err != nil || metadataStampFromInfo(info, identity) != expected {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(f, maxLiveMetadataBytes+1))
	if err != nil || len(data) > maxLiveMetadataBytes {
		return nil
	}
	info, err = f.Stat()
	if err != nil || metadataStampFromInfo(info, identity) != expected {
		return nil
	}
	current, err := metadataStampForPath(path)
	if err != nil || current != expected {
		return nil
	}
	var raw struct {
		ArenaID json.RawMessage `json:"arenaUniqueId"`
		MapID   uint32          `json:"mapId"`
		MapName string          `json:"mapName"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	var idText string
	if json.Unmarshal(raw.ArenaID, &idText) != nil {
		idText = string(raw.ArenaID)
	}
	id, err := strconv.ParseUint(idText, 10, 64)
	if err != nil || id == 0 || (raw.MapID == 0 && raw.MapName == "") {
		return nil
	}
	return &liveMetadata{arenaID: id, mapID: raw.MapID, mapName: raw.MapName}
}
