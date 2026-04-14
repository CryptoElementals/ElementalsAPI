package db

import (
	"strings"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// WinnerFromPlayerResultInfos returns the winner from persisted per-player rows (IsWinner or PLAYER_WIN).
func WinnerFromPlayerResultInfos(infos []*dao.PlayerResultInfo) (playerId int64, temp string, ok bool) {
	for _, p := range infos {
		if p == nil {
			continue
		}
		if p.IsWinner || p.PlayerGameResultStatus == proto.PlayerGameResultStatus_PLAYER_WIN {
			return p.PlayerId, p.TemporaryAddress, true
		}
	}
	return 0, "", false
}

// PlayerResultInfoByPlayerID builds a lookup for joining economy rows to outcome metadata.
func PlayerResultInfoByPlayerID(infos []*dao.PlayerResultInfo) map[int64]*dao.PlayerResultInfo {
	m := make(map[int64]*dao.PlayerResultInfo)
	for _, p := range infos {
		if p != nil {
			m[p.PlayerId] = p
		}
	}
	return m
}

// TemporaryAddressForPlayer returns the room temp address for a player from result infos, or empty.
func TemporaryAddressForPlayer(infos []*dao.PlayerResultInfo, playerID int64) string {
	if pri, ok := PlayerResultInfoByPlayerID(infos)[playerID]; ok && pri != nil {
		return pri.TemporaryAddress
	}
	return ""
}

// AddrsEqualFold compares two temporary addresses case-insensitively.
func AddrsEqualFold(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
