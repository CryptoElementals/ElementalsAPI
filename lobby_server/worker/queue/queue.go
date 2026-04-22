package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CryptoElementals/common/cmd/ele-stat/proto"
	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/lobby_server/bot_manager"
	"github.com/CryptoElementals/common/lobby_server/player_info"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/timer"
	"google.golang.org/grpc"
)

// EventPublisher publishes outbound player notifications (same contract as game.Publisher).
type EventPublisher = pubsub.Publisher

type Queue struct {
	ctx        context.Context
	lobbyState *player_info.RedisStore
	botStore   *bot_manager.RedisStore
	// Continue rematch cancel deadline (seconds); same config source as former continue queue timeout.
	continueRematchCancelTimeoutSec    int64
	continueRematchCancelRedundancySec int64
	publisher                          EventPublisher
	gameCreator                        GameCreator
	minTokenToJoinQueue                int32
	matchConfirmationTimeout           int64

	botWaitTime  time.Duration
	botFreshness time.Duration

	statServiceEndpoint string
	statSvcClient       proto.StatServiceClient
}

type GameCreator interface {
	CreateGameAndRun(players []types.PlayerAddress, gameType uint, completedMatchID int64) (int64, error)
}

func NewQueue(
	ctx context.Context,
	pub EventPublisher,
	botStore *bot_manager.RedisStore,
	gameCreator GameCreator,
	matchConfirmationTimeout int64,
	continueTimeout int64,
	continueTimeoutRedundancy int64,
	botWaitTime int64,
	botFreshnessSec int64,
	minTokenToJoinQueue int32,
	statServiceEndpoint string,
) (*Queue, error) {
	if botFreshnessSec <= 0 {
		botFreshnessSec = 20
	}
	q := &Queue{
		ctx:                                ctx,
		publisher:                          pub,
		gameCreator:                        gameCreator,
		continueRematchCancelTimeoutSec:    continueTimeout,
		continueRematchCancelRedundancySec: continueTimeoutRedundancy,
		botWaitTime:                        time.Duration(botWaitTime) * time.Second,
		botFreshness:                       time.Duration(botFreshnessSec) * time.Second,
		minTokenToJoinQueue:                minTokenToJoinQueue,
		matchConfirmationTimeout:           matchConfirmationTimeout,
		statServiceEndpoint:                statServiceEndpoint,
	}
	lobbyState, err := player_info.NewRedisStore("")
	if err != nil {
		return nil, fmt.Errorf("lobby redis state: %w", err)
	}
	q.lobbyState = lobbyState
	if botStore == nil {
		botStore, err = bot_manager.NewRedisStore("")
		if err != nil {
			return nil, fmt.Errorf("lobby redis bots: %w", err)
		}
	}
	q.botStore = botStore
	q.registerPendingMatchConfirmationTimeoutHandler()
	q.registerBotDispatchTickHandler()
	return q, nil
}

func (q *Queue) start() error {
	if ep := strings.TrimSpace(q.statServiceEndpoint); ep != "" {
		conn, err := grpc.DialContext(q.ctx, ep, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Errorf("failed to connect to stat service: %s", err.Error())
			return err
		}
		q.statSvcClient = proto.NewStatServiceClient(conn)
	} else {
		log.Infow("stat service endpoint empty; skipping gRPC dial and post-game UpdatePlayerStats")
	}
	q.addBotRoutine()
	return nil
}

func (q *Queue) close() {
	if err := timer.UnregisterBotDispatchRecurring(); err != nil {
		log.Errorw("unregister bot dispatch recurring failed", "err", err)
	}
}

func (q *Queue) HandleJoinQueueEvent(req *pb.PlayerAddress) error {
	var address types.PlayerAddress
	address.FromProto(req)
	inQueue, err := q.lobbyState.IsInQueue(q.ctx, address)
	if err != nil {
		return err
	}
	if inQueue {
		return errors.New("player already in queue")
	}
	log.Infow("join queue", "player id", address.Id, "temporary address", address.TemporaryAddress)
	err = q.lockToken(&address)
	if err != nil {
		log.Errorf("cannot join queue, err: %s", err.Error())
		return err
	}
	now := time.Now()
	candidate, queued, err := q.lobbyState.JoinQueueOrGetMatchCandidate(q.ctx, address, now.UnixMilli())
	if err != nil {
		_ = q.unlockToken(&address)
		return err
	}
	if candidate != nil {
		return q.matchPlayers([]types.PlayerAddress{address, *candidate})
	}
	if !queued {
		_ = q.unlockToken(&address)
		return errors.New("player cannot enter queue")
	}
	return nil
}

func (q *Queue) matchPlayers(players []types.PlayerAddress) error {
	return q.createPvpMatchFromQueue(players)
}

func (q *Queue) HandleExitQueueEvent(req *pb.PlayerAddress) error {
	var address types.PlayerAddress
	address.FromProto(req)
	ok, err := q.lobbyState.IsInQueue(q.ctx, address)
	if err != nil {
		return err
	}
	if !ok {
		log.Debugw("player not in queue", "player", address.String())
		return nil
	}
	return q.removePlayerFromQueue(address)
}

func (q *Queue) removePlayerFromQueue(player types.PlayerAddress) error {
	if err := q.lobbyState.RemoveQueue(q.ctx, player); err != nil {
		log.Errorw("remove player from redis queue failed", "player", player.String(), "err", err)
	}
	err := q.unlockToken(&player)
	if err != nil {
		log.Errorf("unlock user token failed: %s", err.Error())
	}
	return err
}

func (q *Queue) GameResultSettlement(event *types.GameCompletedEvent) error {
	if event == nil || event.GameID == 0 {
		return nil
	}
	gameID := event.GameID
	gr, err := db.LoadGameResultByGameID(gameID)
	if err != nil {
		return fmt.Errorf("queue settlement: load game result %d: %w", gameID, err)
	}

	bots := Set[types.PlayerAddress]{}
	forEachSettlementParticipant(gr, func(playerID int64, temporaryAddress string) {
		addr := types.NewPlayerAddress(playerID, temporaryAddress)
		q.markPlayerOutOfGame(*addr)
		isBot := q.releaseInGameBot(*addr)
		if isBot {
			bots.Add(*addr)
		}
	})
	skippedDup, err := db.BattleResultSettlement(gr)
	if err != nil {
		log.Errorw("BattleResultSettlement failed", "err", err)
		return err
	}
	if gr.GameResultType == pb.GameResultType_GAME_ABORTED {
		return nil
	}

	q.publishGameSettlementResult(gameID, gr)

	if len(bots) > 0 {
		log.Infow("skipping continue rematch: game included bots", "game_id", gameID)
		q.publishNotMatchableForHumans(gr, gameID, bots)
	} else if q.anyHumanPlayerBelowQueueThreshold(gr, bots) {
		log.Infow("skipping continue rematch: insufficient tokens after settlement", "game_id", gameID)
		q.publishNotMatchableForHumans(gr, gameID, bots)
	} else if !skippedDup {
		q.tryStartContinueRematchAfterGame(gameID, gr)
	}

	go func() {
		if skippedDup {
			return
		}
		if q.statSvcClient == nil {
			return
		}
		playerIDs := settlementPlayerIDs(gr)
		resp, err := q.statSvcClient.UpdatePlayerStats(q.ctx, &proto.UpdatePlayerStatsRequest{
			PlayerIds: playerIDs,
		})
		if err != nil {
			log.Errorw("UpdatePlayerStats error", "err", err)
			return
		}
		if !resp.Ok {
			log.Errorw("UpdatePlayerStats failed", "message", resp.Message)
			return
		}
		log.Infow("UpdatePlayerStats success", "players", playerIDs)
	}()

	return nil
}

func (q *Queue) isPlayerInQueue(address types.PlayerAddress) bool {
	ok, err := q.lobbyState.IsInQueue(q.ctx, address)
	if err != nil {
		log.Errorw("redis is in queue check failed", "player", address.String(), "err", err)
		return false
	}
	return ok
}

func (q *Queue) isPlayerInGame(address types.PlayerAddress) bool {
	address.TemporaryAddress = strings.ToLower(address.TemporaryAddress)
	ok, err := q.lobbyState.IsInGame(q.ctx, address)
	if err != nil {
		log.Errorw("redis is in game check failed", "player", address.String(), "err", err)
		return false
	}
	return ok
}

func (q *Queue) pendingMatchID(address types.PlayerAddress) (int64, bool) {
	address.TemporaryAddress = strings.ToLower(address.TemporaryAddress)
	mid, ok, err := q.lobbyState.PendingMatchID(q.ctx, address)
	if err != nil {
		log.Errorw("redis pending match lookup failed", "player", address.String(), "err", err)
		return 0, false
	}
	return mid, ok
}

func (q *Queue) markPlayerOutOfGame(address types.PlayerAddress) {
	address.TemporaryAddress = strings.ToLower(address.TemporaryAddress)
	if err := q.lobbyState.MarkPlayersOutOfGame(q.ctx, address); err != nil {
		log.Errorw("mark player out of game in redis failed", "player", address.String(), "err", err)
	}
}

func (q *Queue) lockToken(address *types.PlayerAddress) error {
	log.Infow("lock user token", "addr", address.String(), "token amount", q.minTokenToJoinQueue)
	return db.LockUserToken(q.ctx, address.Id, address.TemporaryAddress, q.minTokenToJoinQueue, "")
}

func (q *Queue) unlockToken(address *types.PlayerAddress) error {
	log.Infow("unlock user token", "addr", address.String(), "token amount", q.minTokenToJoinQueue)
	return db.UnlockUserToken(q.ctx, address.Id, address.TemporaryAddress, false)
}

func winnerPlayerIDFromGameResult(gr *dao.GameResult) int64 {
	if gr == nil {
		return 0
	}
	for _, pri := range gr.PlayerResultInfos {
		if pri != nil && pri.IsWinner {
			return pri.PlayerId
		}
	}
	return 0
}

func (q *Queue) publishGameSettlementResult(gameID int64, gr *dao.GameResult) {
	if gr == nil {
		return
	}
	br, err := db.LoadBattleRewardPVPByGameID(gameID)
	if err != nil {
		log.Errorw("load battle reward for settlement pub", "game_id", gameID, "err", err)
		return
	}
	playerRewards := conversion.DbPlayerRewardsToProto(br.PlayerRewards, gr.PlayerResultInfos)
	receivers := make([]*pb.PlayerAddress, 0, len(gr.PlayerResultInfos))
	for _, pri := range gr.PlayerResultInfos {
		if pri == nil {
			continue
		}
		receivers = append(receivers, types.NewPlayerAddress(pri.PlayerId, pri.TemporaryAddress).ToProto())
	}
	out := &pb.Event{
		Type:      pb.EventType_TYPE_GAME_SETTLEMENT_RESULT,
		Receivers: receivers,
		Event: &pb.Event_GameSettlementResult{
			GameSettlementResult: &pb.GameSettlementResult{
				GameId:         gameID,
				SystemFee:      br.SystemFee,
				PlayerRewards:  playerRewards,
				Multiplier:     gr.Multiplier,
				WinnerPlayerId: winnerPlayerIDFromGameResult(gr),
			},
		},
	}
	if err := pubsub.Publish(q.ctx, q.publisher, pubsub.TopicLobby, out); err != nil {
		log.Errorw("publish game settlement result failed", "topic", pubsub.TopicLobby, "game_id", gameID, "err", err)
	}
}

func (q *Queue) publishNotMatchableForHumans(gr *dao.GameResult, lastGameID int64, bots Set[types.PlayerAddress]) {
	receivers := make([]*pb.PlayerAddress, 0, 4)
	forEachSettlementParticipant(gr, func(playerID int64, temporaryAddress string) {
		addr := types.NewPlayerAddress(playerID, temporaryAddress)
		if bots.Contains(*addr) {
			return
		}
		receivers = append(receivers, addr.ToProto())
	})
	evt := &pb.Event{
		Type:      pb.EventType_TYPE_NOT_MATCHABLE,
		Receivers: receivers,
		Event: &pb.Event_NotMatchable{
			NotMatchable: &pb.NotMatchable{LastGameId: lastGameID},
		},
	}
	if err := pubsub.Publish(q.ctx, q.publisher, pubsub.TopicLobby, evt); err != nil {
		log.Errorw("publish not matchable failed", "topic", pubsub.TopicLobby, "last_game_id", lastGameID, "err", err)
	}
}
