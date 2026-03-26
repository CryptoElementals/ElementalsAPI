package conversion

import (
	"slices"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// RoundView groups persisted turns for one logical round (proto/RPC). It is not a database model.
type RoundView struct {
	RoundNumber    uint32
	Turns          []*dao.Turn
	IsLastRound    bool
	CompleteReason proto.RoundCompleteReason
}

// SyntheticRoundsFromTurns groups flat dao.Turn rows by round_number.
func SyntheticRoundsFromTurns(turns []*dao.Turn) []*RoundView {
	if len(turns) == 0 {
		return nil
	}
	byRound := make(map[uint32][]*dao.Turn)
	var maxRound uint32
	for _, t := range turns {
		if t == nil {
			continue
		}
		rn := t.RoundNumber
		byRound[rn] = append(byRound[rn], t)
		if rn > maxRound {
			maxRound = rn
		}
	}
	out := make([]*RoundView, 0, maxRound)
	for rn := uint32(1); rn <= maxRound; rn++ {
		ts := byRound[rn]
		if len(ts) == 0 {
			continue
		}
		slices.SortFunc(ts, func(a, b *dao.Turn) int {
			if a.TurnNumber < b.TurnNumber {
				return -1
			}
			if a.TurnNumber > b.TurnNumber {
				return 1
			}
			return 0
		})
		out = append(out, &RoundView{
			RoundNumber: rn,
			Turns:       ts,
		})
	}
	return out
}

// RoundByNumber returns the synthetic round view for roundNum, or nil.
func RoundByNumber(g *dao.Game, roundNum uint32) *RoundView {
	if g == nil {
		return nil
	}
	for _, r := range SyntheticRoundsFromTurns(g.Turns) {
		if r.RoundNumber == roundNum {
			return r
		}
	}
	return nil
}
