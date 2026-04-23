package queue

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/timer"
)

func normalizePairOrder(a, b types.PlayerAddress) (types.PlayerAddress, types.PlayerAddress) {
	a.TemporaryAddress = strings.ToLower(a.TemporaryAddress)
	b.TemporaryAddress = strings.ToLower(b.TemporaryAddress)
	if a.Id < b.Id || (a.Id == b.Id && a.TemporaryAddress < b.TemporaryAddress) {
		return a, b
	}
	return b, a
}

// createPvpMatchFromQueue inserts game_match and notifies players.
func (q *Queue) createPvpMatchFromQueue(players []types.PlayerAddress) error {
	if len(players) != 2 {
		return fmt.Errorf("match requires 2 players, got %d", len(players))
	}
	p1, p2 := normalizePairOrder(players[0], players[1])
	m := &dao.GameMatch{
		Player1ID:          p1.Id,
		Player1TempAddress: p1.TemporaryAddress,
		Player2ID:          p2.Id,
		Player2TempAddress: p2.TemporaryAddress,
		GameType:           types.GameTypePVP,
		Status:             dao.GameMatchStatusPending,
	}
	if err := db.InsertGameMatch(q.ctx, m); err != nil {
		return err
	}
	matchID := m.ID
	pair := []types.PlayerAddress{p1, p2}
	ok, err := q.lobbyState.SetPendingPair(q.ctx, matchID, p1, p2)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("set pending pair conflict for match_id=%d", matchID)
	}
	q.publishMatchPending(matchID, pair, nil)
	log.Infow("pvp match pending confirmations", "match_id", matchID, "p1", p1.String(), "p2", p2.String())
	q.schedulePendingMatchConfirmationTimeout(matchID, q.matchConfirmationTimeout, 0)
	return nil
}

// createContinueRematchMatch inserts game_match (with last_game_id), registers pending confirmations, and publishes TYPE_MATCHED (GameMatched with LastGameId) per player.
func (q *Queue) createContinueRematchMatch(players []types.PlayerAddress, lastGameID int64) (int64, error) {
	if len(players) != 2 {
		return 0, fmt.Errorf("continue rematch requires 2 players, got %d", len(players))
	}
	p1, p2 := normalizePairOrder(players[0], players[1])
	m := &dao.GameMatch{
		Player1ID:          p1.Id,
		Player1TempAddress: p1.TemporaryAddress,
		Player2ID:          p2.Id,
		Player2TempAddress: p2.TemporaryAddress,
		GameType:           types.GameTypePVP,
		Status:             dao.GameMatchStatusPending,
		LastGameID:         lastGameID,
	}
	if err := db.InsertGameMatch(q.ctx, m); err != nil {
		return 0, err
	}
	matchID := m.ID
	ok, err := q.lobbyState.SetPendingPair(q.ctx, matchID, p1, p2)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("set pending pair conflict for continue rematch match_id=%d", matchID)
	}
	pair := []types.PlayerAddress{p1, p2}
	lg := lastGameID
	q.publishMatchPending(matchID, pair, &lg)
	log.Infow("continue rematch pending confirmations", "match_id", matchID, "last_game_id", lastGameID, "p1", p1.String(), "p2", p2.String())
	return matchID, nil
}

func (q *Queue) publishMatchPending(matchID int64, players []types.PlayerAddress, lastGameID *int64) {
	topics := make([]*pb.PlayerAddress, 0, len(players))
	for _, pl := range players {
		topics = append(topics, pl.ToProto())
	}
	gm := &pb.GameMatched{
		ConfirmationTimeout: q.matchConfirmationTimeout,
		Players:             topics,
		MatchId:             matchID,
	}
	if lastGameID != nil {
		lid := *lastGameID
		gm.LastGameId = &lid
	}
	out := &pb.Event{
		Type:      pb.EventType_TYPE_MATCHED,
		Receivers: topics,
		Event: &pb.Event_GameMatched{
			GameMatched: gm,
		},
	}
	if err := pubsub.Publish(q.ctx, q.publisher, pubsub.TopicLobby, out); err != nil {
		log.Errorw("publish match pending failed", "topic", pubsub.TopicLobby, "err", err)
	}
}

func (q *Queue) publishMatchCanceled(matchID int64, players []types.PlayerAddress, fromTimeout bool) {
	topics := make([]*pb.PlayerAddress, 0, len(players))
	for _, pl := range players {
		topics = append(topics, pl.ToProto())
	}
	out := &pb.Event{
		Type:      pb.EventType_TYPE_MATCH_CANCELED,
		Receivers: topics,
		Event: &pb.Event_MatchCanceled{
			MatchCanceled: &pb.MatchCanceled{
				MatchId:     matchID,
				Players:     topics,
				FromTimeout: fromTimeout,
			},
		},
	}
	if err := pubsub.Publish(q.ctx, q.publisher, pubsub.TopicLobby, out); err != nil {
		log.Errorw("publish match canceled failed", "topic", pubsub.TopicLobby, "err", err)
	}
}

// HandleCancelMatch cancels a pending game_match if the caller is a participant (both players receive TYPE_MATCH_CANCELED).
func (q *Queue) HandleCancelMatch(req *pb.CancelMatchRequest) error {
	if req.PlayerAddress == nil {
		return errors.New("missing player address")
	}
	var addr types.PlayerAddress
	addr.FromProto(req.PlayerAddress)
	matchID := req.GetMatchId()
	if matchID == 0 {
		return errors.New("missing match id")
	}
	mapID, ok := q.pendingMatchID(addr)
	if !ok || mapID != matchID {
		return fmt.Errorf("no pending match for player")
	}
	m, err := db.GetGameMatchByID(q.ctx, matchID)
	if err != nil {
		return err
	}
	if m.Status != dao.GameMatchStatusPending {
		return fmt.Errorf("match is not pending")
	}
	if !addressInGameMatch(m, addr) {
		return fmt.Errorf("player not in match")
	}
	return q.abortPendingMatch(matchID, true, false)
}

func addressInGameMatch(m *dao.GameMatch, addr types.PlayerAddress) bool {
	if m.Player1ID == addr.Id && strings.EqualFold(m.Player1TempAddress, addr.TemporaryAddress) {
		return true
	}
	if m.Player2ID == addr.Id && strings.EqualFold(m.Player2TempAddress, addr.TemporaryAddress) {
		return true
	}
	return false
}

// HandleConfirmMatch records queue-side confirmation for a pending game_match row.
func (q *Queue) HandleConfirmMatch(req *pb.ConfirmMatchRequest) error {
	if req.PlayerAddress == nil {
		return errors.New("missing player address")
	}
	var addr types.PlayerAddress
	addr.FromProto(req.PlayerAddress)
	matchID := req.GetMatchId()
	if matchID == 0 {
		return errors.New("missing match id")
	}

	mapID, ok := q.pendingMatchID(addr)
	if !ok || mapID != matchID {
		return fmt.Errorf("no pending match for player")
	}

	m, _, both, err := db.TryConfirmGameMatchPlayer(q.ctx, matchID, addr.Id, addr.TemporaryAddress)
	if err != nil {
		return err
	}
	if both {
		return q.finalizeConfirmedGameMatch(m)
	}
	return nil
}

func (q *Queue) finalizeConfirmedGameMatch(m *dao.GameMatch) error {
	claimedRow, claimed, err := db.ClaimGameMatchForCreation(q.ctx, m.ID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	players := []types.PlayerAddress{
		*types.NewPlayerAddress(claimedRow.Player1ID, claimedRow.Player1TempAddress),
		*types.NewPlayerAddress(claimedRow.Player2ID, claimedRow.Player2TempAddress),
	}
	gt := claimedRow.GameType
	if gt == 0 {
		gt = types.GameTypePVP
	}
	gid, err := q.gameCreator.CreateGameAndRun(players, gt, m.ID)
	if err != nil {
		if revErr := db.RevertGameMatchToPending(q.ctx, m.ID); revErr != nil {
			log.Errorw("revert game_match after failed game create", "match_id", m.ID, "err", revErr)
		}
		return err
	}
	if err := db.CompleteClaimedGameMatch(q.ctx, m.ID, gid); err != nil {
		log.Errorw("complete claimed game_match", "match_id", m.ID, "game_id", gid, "err", err)
	}
	ok, err := q.lobbyState.FinalizeConfirmedPair(q.ctx, m.ID, players[0], players[1])
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("finalize confirmed pair conflict for match_id=%d", m.ID)
	}
	for _, p := range players {
		if q.isPlayerBot(p) {
			continue
		}
		if err := db.SetLockedTokenGameID(q.ctx, p.Id, p.TemporaryAddress, gid); err != nil {
			log.Errorf("set locked token game id failed, player: %s, err: %s", p.String(), err.Error())
		}
	}
	return nil
}

// IsPlayerPendingMatch reports whether the player is waiting on game_match confirmations.
func (q *Queue) IsPlayerPendingMatch(address types.PlayerAddress) bool {
	_, ok := q.pendingMatchID(address)
	return ok
}

func (q *Queue) schedulePendingMatchConfirmationTimeout(matchID int64, timeoutSec, redundancySec int64) {
	if timeoutSec <= 0 {
		return
	}
	d := time.Duration(timeoutSec+redundancySec) * time.Second
	if err := timer.ProcessIn(timer.ScopeLobby, d, &pendingMatchConfirmationTimeoutEvent{MatchID: matchID}, true); err != nil {
		log.Errorw("schedule pending match confirmation timeout failed", "match_id", matchID, "err", err)
	}
}
