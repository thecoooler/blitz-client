package replay

import (
	"fmt"
	"math"
	"math/big"

	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/pickle"
)

// BasePlayerCreate is the arena context pickled in the first packet.
type BasePlayerCreate struct {
	PlayersBattleCategoriesIDs *map[uint64][2]uint32 `json:"players_battle_categories_ids"`
	BattleLevel                uint8                 `json:"battle_level"`
	BattleCategoryID           uint8                 `json:"battle_category_id"`
	IsMouseEnabled             bool                  `json:"is_mouse_enabled"`
	MatchmakerType             uint8                 `json:"matchmaker_type"`
	CamouflageSlot             uint8                 `json:"camouflage_slot"`
	// AverageMMRatings holds per-team average ratings for rating battles; an
	// element is nil when every player of the team is calibrating.
	AverageMMRatings       *[]*jsonx.F64           `json:"average_mm_ratings"`
	PlayerWaitTimes        *map[uint64]jsonx.F64   `json:"player_wait_times"`
	AccountDatabaseIDs     *[]uint64               `json:"account_database_ids"`
	TurboBattlesStatistics *TurboBattlesStatistics `json:"turbo_battles_statistics"`
}

// TurboBattlesStatistics is turbo-battle matchmaking context.
type TurboBattlesStatistics struct {
	BattleChance              jsonx.F64 `json:"battle_chance"`
	LeaveChance               jsonx.F64 `json:"leave_chance"`
	PossibleBattlesNotForSkip uint32    `json:"possible_battles_not_for_skip"`
	PossibleBattlesForSkip    uint32    `json:"possible_battles_for_skip"`
	BattlesForSkip            uint32    `json:"battles_for_skip"`
	BattlesNotForSkip         uint32    `json:"battles_not_for_skip"`
	ABSkip                    int32     `json:"ab_skip"`
	Beta                      jsonx.F64 `json:"beta"`
	PremiumVehicleDifference  int32     `json:"premium_vehicle_difference"`
	HTDifference              int32     `json:"ht_difference"`
	Alpha                     jsonx.F64 `json:"alpha"`
}

// unexpected describes a pickle value like serde's Unexpected.
func unexpected(value pickle.Value) string {
	switch typed := value.(type) {
	case nil:
		return "unit value"
	case bool:
		return fmt.Sprintf("boolean `%t`", typed)
	case int64:
		return fmt.Sprintf("integer `%d`", typed)
	case *big.Int:
		return fmt.Sprintf("integer `%s`", typed)
	case float64:
		return fmt.Sprintf("floating point `%v`", typed)
	case string:
		return fmt.Sprintf("string %q", typed)
	case pickle.Bytes:
		return "byte array"
	case *pickle.List, pickle.Tuple, *pickle.Set:
		return "sequence"
	case *pickle.Dict:
		return "map"
	}
	return "unresolved global"
}

func pyInteger(value pickle.Value, expected string, minimum, maximum *big.Int) (*big.Int, error) {
	var number *big.Int
	switch typed := value.(type) {
	case int64:
		number = big.NewInt(typed)
	case *big.Int:
		number = typed
	default:
		return nil, fmt.Errorf("invalid type: %s, expected %s", unexpected(value), expected)
	}
	if number.Cmp(minimum) < 0 || number.Cmp(maximum) > 0 {
		return nil, fmt.Errorf("invalid value: integer `%s`, expected %s", number, expected)
	}
	return number, nil
}

var (
	bigZero = big.NewInt(0)
	maxU8   = big.NewInt(math.MaxUint8)
	maxU32  = big.NewInt(math.MaxUint32)
	maxU64  = new(big.Int).SetUint64(math.MaxUint64)
	minI32  = big.NewInt(math.MinInt32)
	maxI32  = big.NewInt(math.MaxInt32)
)

func pyU8(value pickle.Value) (uint8, error) {
	number, err := pyInteger(value, "u8", bigZero, maxU8)
	if err != nil {
		return 0, err
	}
	return uint8(number.Uint64()), nil
}

func pyU32(value pickle.Value) (uint32, error) {
	number, err := pyInteger(value, "u32", bigZero, maxU32)
	if err != nil {
		return 0, err
	}
	return uint32(number.Uint64()), nil
}

func pyU64(value pickle.Value) (uint64, error) {
	number, err := pyInteger(value, "u64", bigZero, maxU64)
	if err != nil {
		return 0, err
	}
	return number.Uint64(), nil
}

func pyI32(value pickle.Value) (int32, error) {
	number, err := pyInteger(value, "i32", minI32, maxI32)
	if err != nil {
		return 0, err
	}
	return int32(number.Int64()), nil
}

func pyF64(value pickle.Value) (jsonx.F64, error) {
	switch typed := value.(type) {
	case float64:
		return jsonx.F64(typed), nil
	case int64:
		return jsonx.F64(typed), nil
	case *big.Int:
		if typed.IsInt64() || typed.IsUint64() {
			number, _ := new(big.Float).SetInt(typed).Float64()
			return jsonx.F64(number), nil
		}
	}
	return 0, fmt.Errorf("invalid type: %s, expected f64", unexpected(value))
}

func pyBool(value pickle.Value) (bool, error) {
	// The game usually stores a Python bool, but some 26.10 battles write an
	// integer flag for mouseEnabled (observed 2). Treat it as truthiness
	// instead of rejecting the whole replay.
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case int64:
		return typed != 0, nil
	case *big.Int:
		return typed.Sign() != 0, nil
	}
	return false, fmt.Errorf("invalid type: %s, expected a boolean", unexpected(value))
}

func pySequence(value pickle.Value, expected string) ([]pickle.Value, error) {
	switch typed := value.(type) {
	case *pickle.List:
		return typed.Items, nil
	case pickle.Tuple:
		return typed, nil
	case *pickle.Set:
		return typed.Items, nil
	}
	return nil, fmt.Errorf("invalid type: %s, expected %s", unexpected(value), expected)
}

func pyDict(value pickle.Value, expected string) (*pickle.Dict, error) {
	if typed, ok := value.(*pickle.Dict); ok {
		return typed, nil
	}
	return nil, fmt.Errorf("invalid type: %s, expected %s", unexpected(value), expected)
}

// pyFields maps struct field names to values, ignoring unknown keys.
func pyFields(value pickle.Value, structName string) (map[string]pickle.Value, error) {
	dict, err := pyDict(value, "struct "+structName)
	if err != nil {
		return nil, err
	}
	fields := map[string]pickle.Value{}
	for _, entry := range dict.Entries {
		var key string
		switch typed := entry.Key.(type) {
		case string:
			key = typed
		case pickle.Bytes:
			key = string(typed)
		default:
			continue
		}
		if _, exists := fields[key]; exists {
			return nil, fmt.Errorf("duplicate field `%s`", key)
		}
		fields[key] = entry.Value
	}
	return fields, nil
}

type fieldReader struct {
	fields map[string]pickle.Value
	err    error
}

// required decodes a mandatory field; optional ones accept missing or None.
func (r *fieldReader) required(name string, decode func(pickle.Value) error) {
	if r.err != nil {
		return
	}
	value, ok := r.fields[name]
	if !ok {
		r.err = fmt.Errorf("missing field `%s`", name)
		return
	}
	r.err = decode(value)
}

func (r *fieldReader) optional(name string, decode func(pickle.Value) error) {
	if r.err != nil {
		return
	}
	if value, ok := r.fields[name]; ok && value != nil {
		r.err = decode(value)
	}
}

func decodeBasePlayerCreate(value pickle.Value) (*BasePlayerCreate, error) {
	fields, err := pyFields(value, "BasePlayerCreate")
	if err != nil {
		return nil, err
	}
	r := &fieldReader{fields: fields}
	out := &BasePlayerCreate{}
	r.optional("playersBattleCategoriesIds", func(v pickle.Value) error {
		dict, err := pyDict(v, "a map")
		if err != nil {
			return err
		}
		result := map[uint64][2]uint32{}
		for _, entry := range dict.Entries {
			key, err := pyU64(entry.Key)
			if err != nil {
				return err
			}
			items, err := pySequence(entry.Value, "a tuple of size 2")
			if err != nil {
				return err
			}
			if len(items) != 2 {
				return fmt.Errorf("invalid length %d, expected a tuple of size 2", len(items))
			}
			first, err := pyU8(items[0])
			if err != nil {
				return err
			}
			second, err := pyU32(items[1])
			if err != nil {
				return err
			}
			result[key] = [2]uint32{uint32(first), second}
		}
		out.PlayersBattleCategoriesIDs = &result
		return nil
	})
	r.required("battleLevel", func(v pickle.Value) (err error) { out.BattleLevel, err = pyU8(v); return })
	r.required("battleCategoryId", func(v pickle.Value) (err error) { out.BattleCategoryID, err = pyU8(v); return })
	r.required("mouseEnabled", func(v pickle.Value) (err error) { out.IsMouseEnabled, err = pyBool(v); return })
	r.required("mmType", func(v pickle.Value) (err error) { out.MatchmakerType, err = pyU8(v); return })
	r.required("camouflageSlot", func(v pickle.Value) (err error) { out.CamouflageSlot, err = pyU8(v); return })
	r.optional("avgMmr", func(v pickle.Value) error {
		items, err := pySequence(v, "a sequence")
		if err != nil {
			return err
		}
		ratings := []*jsonx.F64{}
		for _, item := range items {
			if item == nil {
				ratings = append(ratings, nil)
				continue
			}
			rating, err := pyF64(item)
			if err != nil {
				return err
			}
			ratings = append(ratings, &rating)
		}
		out.AverageMMRatings = &ratings
		return nil
	})
	r.optional("playerWaitTimes", func(v pickle.Value) error {
		dict, err := pyDict(v, "a map")
		if err != nil {
			return err
		}
		result := map[uint64]jsonx.F64{}
		for _, entry := range dict.Entries {
			key, err := pyU64(entry.Key)
			if err != nil {
				return err
			}
			if result[key], err = pyF64(entry.Value); err != nil {
				return err
			}
		}
		out.PlayerWaitTimes = &result
		return nil
	})
	r.optional("accountDatabaseIds", func(v pickle.Value) error {
		items, err := pySequence(v, "a sequence")
		if err != nil {
			return err
		}
		ids := []uint64{}
		for _, item := range items {
			id, err := pyU64(item)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		out.AccountDatabaseIDs = &ids
		return nil
	})
	r.optional("turboBattlesStats", func(v pickle.Value) error {
		stats, err := decodeTurboStatistics(v)
		out.TurboBattlesStatistics = stats
		return err
	})
	if r.err != nil {
		return nil, r.err
	}
	return out, nil
}

func decodeTurboStatistics(value pickle.Value) (*TurboBattlesStatistics, error) {
	fields, err := pyFields(value, "TurboBattlesStatistics")
	if err != nil {
		return nil, err
	}
	r := &fieldReader{fields: fields}
	out := &TurboBattlesStatistics{}
	f64 := func(target *jsonx.F64) func(pickle.Value) error {
		return func(v pickle.Value) (err error) { *target, err = pyF64(v); return }
	}
	u32 := func(target *uint32) func(pickle.Value) error {
		return func(v pickle.Value) (err error) { *target, err = pyU32(v); return }
	}
	i32 := func(target *int32) func(pickle.Value) error {
		return func(v pickle.Value) (err error) { *target, err = pyI32(v); return }
	}
	r.required("turboBattleChance", f64(&out.BattleChance))
	r.required("turboLeaveChance", f64(&out.LeaveChance))
	r.required("possibleTurboBattlesNotForSkip", u32(&out.PossibleBattlesNotForSkip))
	r.required("possibleTurboBattlesForSkip", u32(&out.PossibleBattlesForSkip))
	r.required("turboBattlesForSkip", u32(&out.BattlesForSkip))
	r.required("turboBattlesNotForSkip", u32(&out.BattlesNotForSkip))
	r.required("abSkip", i32(&out.ABSkip))
	r.required("beta", f64(&out.Beta))
	r.required("premiumVehDiff", i32(&out.PremiumVehicleDifference))
	r.required("htDiff", i32(&out.HTDifference))
	r.required("alpha", f64(&out.Alpha))
	if r.err != nil {
		return nil, r.err
	}
	return out, nil
}

// decodeBattleResultsDat reads the (arena ID, protobuf bytes) pickle tuple.
func decodeBattleResultsDat(data []byte) ([]byte, error) {
	_, blob, err := decodeBattleResultsDatWithArena(data)
	return blob, err
}

func decodeBattleResultsDatWithArena(data []byte) (uint64, []byte, error) {
	value, err := pickle.Decode(data)
	if err != nil {
		return 0, nil, err
	}
	items, err := pySequence(value, "a tuple of size 2")
	if err != nil {
		return 0, nil, err
	}
	if len(items) != 2 {
		return 0, nil, fmt.Errorf("invalid length %d, expected a tuple of size 2", len(items))
	}
	arenaID, err := pyU64(items[0])
	if err != nil {
		return 0, nil, err
	}
	switch buffer := items[1].(type) {
	case pickle.Bytes:
		return arenaID, []byte(buffer), nil
	case string:
		return arenaID, []byte(buffer), nil
	case *pickle.List, pickle.Tuple:
		values, _ := pySequence(buffer, "")
		out := make([]byte, len(values))
		for index, item := range values {
			b, err := pyU8(item)
			if err != nil {
				return 0, nil, err
			}
			out[index] = b
		}
		return arenaID, out, nil
	}
	return 0, nil, fmt.Errorf("invalid type: %s, expected byte array", unexpected(items[1]))
}
