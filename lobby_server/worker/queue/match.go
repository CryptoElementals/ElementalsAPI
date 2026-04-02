package queue

import (
	"errors"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	pb "github.com/CryptoElementals/common/rpc/proto"
)

func normalizePairOrder(a, b types.PlayerAddress) (types.PlayerAddress, types.PlayerAddress) {
	a.TemporaryAddress = strings.ToLower(a.TemporaryAddress)
	b.TemporaryAddress = strings.ToLower(b.TemporaryAddress)
	if a.Id < b.Id || (a.Id == b.Id && a.TemporaryAddress < b.TemporaryAddress) {
		return a, b
	}
	return b, a
}

// createPvpMatchFromQueue inserts game_match and notifies players (caller must hold q.lock).
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
	q.pendingMatchByPlayer[p1] = matchID
	q.pendingMatchByPlayer[p2] = matchID
	pair := []types.PlayerAddress{p1, p2}
	for _, p := range pair {
		q.publishMatchPending(p, matchID, pair)
		delete(q.queue, p)
		if err := q.queueCache.Delete(p.String()); err != nil {
			log.Errorf("delete player from queue cache failed: %s", err.Error())
		}
	}
	log.Infow("pvp match pending confirmations", "match_id", matchID, "p1", p1.String(), "p2", p2.String())
	q.schedulePendingMatchConfirmationTimeout(matchID, q.matchConfirmationTimeout, 0)
	return nil
}

// createContinueRematchMatch inserts game_match (with last_game_id), registers pending confirmations, and publishes TYPE_GAME_CONTINUABLE per player. Caller must hold q.lock.
func (q *Queue) createContinueRematchMatch(players []types.PlayerAddress, lastGameID uint) (int64, error) {
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
	q.pendingMatchByPlayer[p1] = matchID
	q.pendingMatchByPlayer[p2] = matchID
	pair := []types.PlayerAddress{p1, p2}
	for _, p := range pair {
		q.publishGameContinuable(p, matchID, lastGameID, pair)
	}
	log.Infow("continue rematch pending confirmations", "match_id", matchID, "last_game_id", lastGameID, "p1", p1.String(), "p2", p2.String())
	return matchID, nil
}

func (q *Queue) publishMatchPending(forPlayer types.PlayerAddress, matchID int64, players []types.PlayerAddress) {
	topics := make([]*pb.PlayerAddress, 0, len(players))
	for _, pl := range players {
		topics = append(topics, pl.ToProto())
	}
	topic := (&forPlayer).String()
	out := &pb.Event{
		Type: pb.EventType_TYPE_MATCHED,
		Event: &pb.Event_GameMatched{
			GameMatched: &pb.GameMatched{
				ConfirmationTimeout: q.matchConfirmationTimeout,
				Players:             topics,
				MatchId:             matchID,
			},
		},
	}
	if err := pubsub.Publish(q.ctx, q.publisher, topic, out); err != nil {
		log.Errorw("publish match pending failed", "topic", topic, "err", err)
	}
}

func (q *Queue) publishGameContinuable(forPlayer types.PlayerAddress, matchID int64, lastGameID uint, players []types.PlayerAddress) {
	topics := make([]*pb.PlayerAddress, 0, len(players))
	for _, pl := range players {
		topics = append(topics, pl.ToProto())
	}
	topic := (&forPlayer).String()
	out := &pb.Event{
		Type: pb.EventType_TYPE_GAME_CONTINUABLE,
		Event: &pb.Event_GameContinuable{
			GameContinuable: &pb.GameContinuable{
				ConfirmationTimeout: q.matchConfirmationTimeout,
				Players:             topics,
				MatchId:             matchID,
				LastGameId:          uint32(lastGameID),
			},
		},
	}
	if err := pubsub.Publish(q.ctx, q.publisher, topic, out); err != nil {
		log.Errorw("publish game continuable failed", "topic", topic, "err", err)
	}
}

func (q *Queue) publishMatchCanceled(forPlayer types.PlayerAddress, matchID int64, players []types.PlayerAddress, fromTimeout bool) {
	topics := make([]*pb.PlayerAddress, 0, len(players))
	for _, pl := range players {
		topics = append(topics, pl.ToProto())
	}
	topic := (&forPlayer).String()
	out := &pb.Event{
		Type: pb.EventType_TYPE_MATCH_CANCELED,
		Event: &pb.Event_MatchCanceled{
			MatchCanceled: &pb.MatchCanceled{
				MatchId:     matchID,
				Players:     topics,
				FromTimeout: fromTimeout,
			},
		},
	}
	if err := pubsub.Publish(q.ctx, q.publisher, topic, out); err != nil {
		log.Errorw("publish match canceled failed", "topic", topic, "err", err)
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
	q.lock.Lock()
	defer q.lock.Unlock()
	mapID, ok := q.pendingMatchByPlayer[addr]
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
	q.abortPendingMatchLocked(matchID, true, false)
	return nil
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

	q.lock.RLock()
	mapID, ok := q.pendingMatchByPlayer[addr]
	q.lock.RUnlock()
	if !ok || mapID != matchID {
		return fmt.Errorf("no pending match for player")
	}

	m, _, both, err := db.TryConfirmGameMatchPlayer(q.ctx, matchID, addr.Id, addr.TemporaryAddress)
	if err != nil {
		return err
	}
	if !both {
		partner := otherPlayerFromMatch(m, addr)
		if partner != nil && q.botMgr.isBot(*partner) {
			_, _, both2, err2 := db.TryConfirmGameMatchPlayer(q.ctx, matchID, partner.Id, partner.TemporaryAddress)
			if err2 != nil {
				return err2
			}
			if both2 {
				m2, err3 := db.GetGameMatchByID(q.ctx, matchID)
				if err3 != nil {
					return err3
				}
				return q.finalizeConfirmedGameMatch(m2)
			}
		}
		return nil
	}
	return q.finalizeConfirmedGameMatch(m)
}

func otherPlayerFromMatch(m *dao.GameMatch, self types.PlayerAddress) *types.PlayerAddress {
	if m.Player1ID == self.Id && strings.EqualFold(m.Player1TempAddress, self.TemporaryAddress) {
		return types.NewPlayerAddress(m.Player2ID, m.Player2TempAddress)
	}
	if m.Player2ID == self.Id && strings.EqualFold(m.Player2TempAddress, self.TemporaryAddress) {
		return types.NewPlayerAddress(m.Player1ID, m.Player1TempAddress)
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
	q.lock.Lock()
	for _, p := range players {
		delete(q.pendingMatchByPlayer, p)
	}
	q.lock.Unlock()
	for _, p := range players {
		if q.botMgr.isBot(p) {
			continue
		}
		if err := db.SetLockedTokenGameID(q.ctx, p.Id, p.TemporaryAddress, gid); err != nil {
			log.Errorf("set locked token game id failed, player: %s, err: %s", p.String(), err.Error())
		}
	}
	return nil
}

// IsPlayerPendingMatch reports whether the player is waiting on game_match confirmations (caller should not hold q.lock).
func (q *Queue) IsPlayerPendingMatch(address types.PlayerAddress) bool {
	address.TemporaryAddress = strings.ToLower(address.TemporaryAddress)
	q.lock.RLock()
	defer q.lock.RUnlock()
	_, ok := q.pendingMatchByPlayer[address]
	return ok
}
