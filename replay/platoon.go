package replay

// PlatoonMembers returns the author's allies that share the same non-zero
// platoon number, in roster order. The author is excluded. A missing or zero
// platoon number means the player is not in a known platoon.
func (r *BattleRoster) PlatoonMembers() []ArenaPlayer {
	if r == nil {
		return nil
	}
	ownIndex := -1
	for i := range r.Allies {
		player := &r.Allies[i]
		if (r.AuthorAccountID != 0 && player.AccountID == r.AuthorAccountID) ||
			(r.AuthorAccountID == 0 && r.AuthorNickname != "" && player.Nickname == r.AuthorNickname) {
			ownIndex = i
			break
		}
	}
	if ownIndex < 0 {
		return nil
	}
	own := &r.Allies[ownIndex]
	if own.PlatoonNumber == nil || *own.PlatoonNumber == 0 {
		return nil
	}
	var members []ArenaPlayer
	for i := range r.Allies {
		member := r.Allies[i]
		if i == ownIndex || member.PlatoonNumber == nil || *member.PlatoonNumber != *own.PlatoonNumber {
			continue
		}
		members = append(members, member)
	}
	return members
}
