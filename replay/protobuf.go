package replay

// Schemas adapted from wotbreplay-parser 0.4.2, copyright 2022 Pavel
// Perestoronin. Distributed under the MIT license; see LICENSE-schemas.

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf8"
)

const protobufRecursionLimit = 100

type wireType int

const (
	wireVarint wireType = iota
	wireFixed64
	wireBytes
	wireStartGroup
	wireEndGroup
	wireFixed32
)

var wireNames = []string{"Varint", "SixtyFourBit", "LengthDelimited", "StartGroup", "EndGroup", "ThirtyTwoBit"}

type protoError struct{ message string }

func (e *protoError) Error() string { return "failed to decode Protobuf message: " + e.message }

func protoFail(format string, args ...any) error {
	return &protoError{fmt.Sprintf(format, args...)}
}

type protoReader struct {
	data  []byte
	depth int
}

func (r *protoReader) varint() (uint64, error) {
	var result uint64
	for index := 0; index < 10; index++ {
		if index >= len(r.data) {
			return 0, protoFail("invalid varint")
		}
		b := r.data[index]
		if index == 9 && b > 1 {
			return 0, protoFail("invalid varint")
		}
		result |= uint64(b&0x7f) << (7 * index)
		if b&0x80 == 0 {
			r.data = r.data[index+1:]
			return result, nil
		}
	}
	return 0, protoFail("invalid varint")
}

func (r *protoReader) take(n uint64) ([]byte, error) {
	if n > uint64(len(r.data)) {
		return nil, protoFail("buffer underflow")
	}
	out := r.data[:n]
	r.data = r.data[n:]
	return out, nil
}

func (r *protoReader) key() (uint32, wireType, error) {
	key, err := r.varint()
	if err != nil {
		return 0, 0, err
	}
	if key > math.MaxUint32 {
		return 0, 0, protoFail("invalid key value: %d", key)
	}
	wire := wireType(key & 7)
	if wire > wireFixed32 {
		return 0, 0, protoFail("invalid wire type value: %d", wire)
	}
	tag := uint32(key >> 3)
	if tag == 0 {
		return 0, 0, protoFail("invalid tag value: 0")
	}
	return tag, wire, nil
}

func (r *protoReader) skip(tag uint32, wire wireType) error {
	switch wire {
	case wireVarint:
		_, err := r.varint()
		return err
	case wireFixed64:
		_, err := r.take(8)
		return err
	case wireFixed32:
		_, err := r.take(4)
		return err
	case wireBytes:
		length, err := r.varint()
		if err != nil {
			return err
		}
		_, err = r.take(length)
		return err
	case wireStartGroup:
		if r.depth >= protobufRecursionLimit {
			return protoFail("recursion limit reached")
		}
		r.depth++
		defer func() { r.depth-- }()
		for {
			inner, innerWire, err := r.key()
			if err != nil {
				return err
			}
			if innerWire == wireEndGroup {
				if inner != tag {
					return protoFail("unexpected end group tag")
				}
				return nil
			}
			if err := r.skip(inner, innerWire); err != nil {
				return err
			}
		}
	default:
		return protoFail("unexpected end group tag")
	}
}

func expectWire(actual, expected wireType) error {
	if actual != expected {
		return protoFail("invalid wire type: %s (expected %s)", wireNames[actual], wireNames[expected])
	}
	return nil
}

// decodeMessage calls field for each known tag; unknown fields are skipped.
func decodeMessage(data []byte, depth int, field func(tag uint32, wire wireType, r *protoReader) (bool, error)) error {
	if depth > protobufRecursionLimit {
		return protoFail("recursion limit reached")
	}
	r := &protoReader{data: data, depth: depth}
	for len(r.data) > 0 {
		tag, wire, err := r.key()
		if err != nil {
			return err
		}
		handled, err := field(tag, wire, r)
		if err != nil {
			return err
		}
		if !handled {
			if err := r.skip(tag, wire); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *protoReader) uint32Field(wire wireType) (uint32, error) {
	if err := expectWire(wire, wireVarint); err != nil {
		return 0, err
	}
	value, err := r.varint()
	return uint32(value), err
}

func (r *protoReader) int32Field(wire wireType) (int32, error) {
	value, err := r.uint32Field(wire)
	return int32(value), err
}

func (r *protoReader) int64Field(wire wireType) (int64, error) {
	if err := expectWire(wire, wireVarint); err != nil {
		return 0, err
	}
	value, err := r.varint()
	return int64(value), err
}

func (r *protoReader) floatField(wire wireType) (float32, error) {
	if err := expectWire(wire, wireFixed32); err != nil {
		return 0, err
	}
	data, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
}

func (r *protoReader) bytesField(wire wireType) ([]byte, error) {
	if err := expectWire(wire, wireBytes); err != nil {
		return nil, err
	}
	length, err := r.varint()
	if err != nil {
		return nil, err
	}
	return r.take(length)
}

func (r *protoReader) stringField(wire wireType) (string, error) {
	data, err := r.bytesField(wire)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", protoFail("invalid string value: data is not UTF-8 encoded")
	}
	return string(data), nil
}

// UpdateArena is the EntityMethod 47 argument carrying the arena roster.
type UpdateArena struct {
	Players *ArenaPlayers `json:"players"`
}

// ArenaPlayers is the roster list inside UpdateArena.
type ArenaPlayers struct {
	Players []ArenaPlayer `json:"players"`
}

// ArenaPlayer is one roster entry.
type ArenaPlayer struct {
	// Destroyed is populated in live snapshots from known vehicle HP.
	Destroyed       bool    `json:"destroyed,omitempty"`
	VehicleTypeID   uint32  `json:"vehicle_type_id,omitempty"`
	VehicleLevel    int     `json:"vehicle_level,omitempty"`
	VehicleName     string  `json:"vehicle_name,omitempty"`
	VehicleEntityID *uint32 `json:"vehicle_entity_id"`
	Nickname        string  `json:"nickname"`
	TeamNumber      uint32  `json:"team_number"`
	AccountID       uint32  `json:"account_id"`
	ClanTag         *string `json:"clan_tag"`
	PlatoonNumber   *uint32 `json:"platoon_number"`
}

func decodeUpdateArena(data []byte) (*UpdateArena, error) {
	arena := &UpdateArena{}
	err := decodeMessage(data, 0, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
		if tag != 1 {
			return false, nil
		}
		body, err := r.bytesField(wire)
		if err != nil {
			return true, err
		}
		if arena.Players == nil {
			arena.Players = &ArenaPlayers{Players: []ArenaPlayer{}}
		}
		return true, decodeMessage(body, 1, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
			if tag != 1 {
				return false, nil
			}
			body, err := r.bytesField(wire)
			if err != nil {
				return true, err
			}
			player, err := decodeArenaPlayer(body)
			if err == nil {
				arena.Players.Players = append(arena.Players.Players, *player)
			}
			return true, err
		})
	})
	return arena, err
}

func decodeArenaPlayer(data []byte) (*ArenaPlayer, error) {
	player := &ArenaPlayer{}
	err := decodeMessage(data, 2, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
		var err error
		switch tag {
		case 1:
			var value uint32
			if value, err = r.uint32Field(wire); err == nil {
				player.VehicleEntityID = &value
			}
		case 2:
			var descriptor []byte
			if descriptor, err = r.bytesField(wire); err == nil && len(descriptor) >= 2 && descriptor[0]&15 == 1 {
				player.VehicleTypeID = uint32(binary.LittleEndian.Uint16(descriptor))
			}
		case 3:
			player.Nickname, err = r.stringField(wire)
		case 4:
			player.TeamNumber, err = r.uint32Field(wire)
		case 7:
			player.AccountID, err = r.uint32Field(wire)
		case 8:
			var value string
			if value, err = r.stringField(wire); err == nil {
				player.ClanTag = &value
			}
		case 11:
			var value uint32
			if value, err = r.uint32Field(wire); err == nil {
				player.PlatoonNumber = &value
			}
		default:
			return false, nil
		}
		return true, err
	})
	return player, err
}

// BattleResults is the decoded battle_results.dat protobuf.
type BattleResults struct {
	// ModeMapID: the two least-significant bytes are the map ID; the third
	// byte is probably the game mode.
	ModeMapID        uint32          `json:"mode_map_id"`
	TimestampSecs    int64           `json:"timestamp_secs"`
	WinnerTeamNumber *int32          `json:"winner_team_number"`
	Author           Author          `json:"author"`
	RoomType         int32           `json:"room_type"`
	FreeXP           uint32          `json:"free_xp"`
	Players          []ResultPlayer  `json:"players"`
	PlayerResults    []PlayerResults `json:"player_results"`
}

// ResultPlayer is a battle participant.
type ResultPlayer struct {
	AccountID uint32           `json:"account_id"`
	Info      ResultPlayerInfo `json:"info"`
}

// ResultPlayerInfo is a participant's extended information.
type ResultPlayerInfo struct {
	Nickname  string  `json:"nickname"`
	PlatoonID *uint32 `json:"platoon_id"`
	Team      int32   `json:"team"`
	ClanTag   *string `json:"clan_tag"`
	Avatar    Avatar  `json:"avatar"`
	Rank      *uint32 `json:"rank"`
}

// PlayerResults is one player's battle results.
type PlayerResults struct {
	ResultID uint32            `json:"result_id"`
	Info     PlayerResultsInfo `json:"info"`
}

// PlayerResultsInfo holds a player's result counters.
type PlayerResultsInfo struct {
	CreditsEarned               uint32   `json:"credits_earned"`
	BaseXP                      uint32   `json:"base_xp"`
	NShots                      uint32   `json:"n_shots"`
	NHitsDealt                  uint32   `json:"n_hits_dealt"`
	NPenetrationsDealt          uint32   `json:"n_penetrations_dealt"`
	DamageDealt                 uint32   `json:"damage_dealt"`
	DamageAssisted1             uint32   `json:"damage_assisted_1"`
	DamageAssisted2             uint32   `json:"damage_assisted_2"`
	NHitsReceived               uint32   `json:"n_hits_received"`
	NNonPenetratingHitsReceived uint32   `json:"n_non_penetrating_hits_received"`
	NPenetrationsReceived       uint32   `json:"n_penetrations_received"`
	NEnemiesDamaged             uint32   `json:"n_enemies_damaged"`
	NEnemiesDestroyed           uint32   `json:"n_enemies_destroyed"`
	VictoryPointsEarned         uint32   `json:"victory_points_earned"`
	VictoryPointsSeized         uint32   `json:"victory_points_seized"`
	AccountID                   uint32   `json:"account_id"`
	TankID                      uint32   `json:"tank_id"`
	MMRating                    *float32 `json:"mm_rating"`
	DamageBlocked               uint32   `json:"damage_blocked"`
}

// Author holds the replay author's results.
type Author struct {
	// HitpointsLeft is -2 when the player is auto-destroyed by inactivity.
	HitpointsLeft int32  `json:"hitpoints_left"`
	TotalCredits  uint32 `json:"total_credits"`
	TotalXP       uint32 `json:"total_xp"`
	NShots        uint32 `json:"n_shots"`
	NHits         uint32 `json:"n_hits"`
	NSplashes     uint32 `json:"n_splashes"`
	NPenetrations uint32 `json:"n_penetrations"`
	DamageDealt   uint32 `json:"damage_dealt"`
	AccountID     uint32 `json:"account_id"`
	TeamNumber    int32  `json:"team_number"`
}

// Avatar is a player's avatar.
type Avatar struct {
	Info AvatarInfo `json:"info"`
}

// AvatarInfo describes avatar graphics.
type AvatarInfo struct {
	GfxURL  string `json:"gfx_url"`
	Gfx2URL string `json:"gfx2_url"`
	Kind    string `json:"kind"`
}

func decodeBattleResults(data []byte) (*BattleResults, error) {
	results := &BattleResults{Players: []ResultPlayer{}, PlayerResults: []PlayerResults{}}
	err := decodeMessage(data, 0, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
		var err error
		switch tag {
		case 1:
			results.ModeMapID, err = r.uint32Field(wire)
		case 2:
			results.TimestampSecs, err = r.int64Field(wire)
		case 3:
			var value int32
			if value, err = r.int32Field(wire); err == nil {
				results.WinnerTeamNumber = &value
			}
		case 8:
			var body []byte
			if body, err = r.bytesField(wire); err == nil {
				err = decodeAuthor(body, &results.Author)
			}
		case 9:
			results.RoomType, err = r.int32Field(wire)
		case 137:
			results.FreeXP, err = r.uint32Field(wire)
		case 201:
			var body []byte
			if body, err = r.bytesField(wire); err == nil {
				var player ResultPlayer
				if err = decodeResultPlayer(body, &player); err == nil {
					results.Players = append(results.Players, player)
				}
			}
		case 301:
			var body []byte
			if body, err = r.bytesField(wire); err == nil {
				var row PlayerResults
				if err = decodePlayerResults(body, &row); err == nil {
					results.PlayerResults = append(results.PlayerResults, row)
				}
			}
		default:
			return false, nil
		}
		return true, err
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func decodeAuthor(data []byte, author *Author) error {
	return decodeMessage(data, 1, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
		var err error
		switch tag {
		case 1:
			author.HitpointsLeft, err = r.int32Field(wire)
		case 2:
			author.TotalCredits, err = r.uint32Field(wire)
		case 3:
			author.TotalXP, err = r.uint32Field(wire)
		case 4:
			author.NShots, err = r.uint32Field(wire)
		case 5:
			author.NHits, err = r.uint32Field(wire)
		case 6:
			author.NSplashes, err = r.uint32Field(wire)
		case 7:
			author.NPenetrations, err = r.uint32Field(wire)
		case 8:
			author.DamageDealt, err = r.uint32Field(wire)
		case 101:
			author.AccountID, err = r.uint32Field(wire)
		case 102:
			author.TeamNumber, err = r.int32Field(wire)
		default:
			return false, nil
		}
		return true, err
	})
}

func decodeResultPlayer(data []byte, player *ResultPlayer) error {
	return decodeMessage(data, 1, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
		var err error
		switch tag {
		case 1:
			player.AccountID, err = r.uint32Field(wire)
		case 2:
			var body []byte
			if body, err = r.bytesField(wire); err == nil {
				err = decodeResultPlayerInfo(body, &player.Info)
			}
		default:
			return false, nil
		}
		return true, err
	})
}

func decodeResultPlayerInfo(data []byte, info *ResultPlayerInfo) error {
	return decodeMessage(data, 2, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
		var err error
		switch tag {
		case 1:
			info.Nickname, err = r.stringField(wire)
		case 2:
			var value uint32
			if value, err = r.uint32Field(wire); err == nil {
				info.PlatoonID = &value
			}
		case 3:
			info.Team, err = r.int32Field(wire)
		case 5:
			var value string
			if value, err = r.stringField(wire); err == nil {
				info.ClanTag = &value
			}
		case 7:
			var body []byte
			if body, err = r.bytesField(wire); err == nil {
				err = decodeMessage(body, 3, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
					if tag != 2 {
						return false, nil
					}
					body, err := r.bytesField(wire)
					if err != nil {
						return true, err
					}
					return true, decodeAvatarInfo(body, &info.Avatar.Info)
				})
			}
		case 9:
			var value uint32
			if value, err = r.uint32Field(wire); err == nil {
				info.Rank = &value
			}
		default:
			return false, nil
		}
		return true, err
	})
}

func decodeAvatarInfo(data []byte, info *AvatarInfo) error {
	return decodeMessage(data, 4, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
		var err error
		switch tag {
		case 2:
			info.GfxURL, err = r.stringField(wire)
		case 3:
			info.Gfx2URL, err = r.stringField(wire)
		case 4:
			info.Kind, err = r.stringField(wire)
		default:
			return false, nil
		}
		return true, err
	})
}

func decodePlayerResults(data []byte, row *PlayerResults) error {
	return decodeMessage(data, 1, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
		var err error
		switch tag {
		case 1:
			row.ResultID, err = r.uint32Field(wire)
		case 2:
			var body []byte
			if body, err = r.bytesField(wire); err == nil {
				err = decodePlayerResultsInfo(body, &row.Info)
			}
		default:
			return false, nil
		}
		return true, err
	})
}

func decodePlayerResultsInfo(data []byte, info *PlayerResultsInfo) error {
	fields := map[uint32]*uint32{
		2: &info.CreditsEarned, 3: &info.BaseXP, 4: &info.NShots, 5: &info.NHitsDealt,
		7: &info.NPenetrationsDealt, 8: &info.DamageDealt, 9: &info.DamageAssisted1,
		10: &info.DamageAssisted2, 12: &info.NHitsReceived, 13: &info.NNonPenetratingHitsReceived,
		15: &info.NPenetrationsReceived, 17: &info.NEnemiesDamaged, 18: &info.NEnemiesDestroyed,
		32: &info.VictoryPointsEarned, 33: &info.VictoryPointsSeized, 101: &info.AccountID,
		103: &info.TankID, 117: &info.DamageBlocked,
	}
	return decodeMessage(data, 2, func(tag uint32, wire wireType, r *protoReader) (bool, error) {
		if target, ok := fields[tag]; ok {
			value, err := r.uint32Field(wire)
			*target = value
			return true, err
		}
		if tag == 107 {
			value, err := r.floatField(wire)
			if err == nil {
				info.MMRating = &value
			}
			return true, err
		}
		return false, nil
	})
}
