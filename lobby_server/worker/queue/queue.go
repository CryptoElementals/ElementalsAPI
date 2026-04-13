package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	"google.golang.org/grpc"
)

// EventPublisher publishes outbound player notifications (same contract as game.Publisher).
type EventPublisher = pubsub.Publisher

type Queue struct {
	ctx        context.Context
	lock       sync.RWMutex
	lobbyState *player_info.RedisStore
	botStore   *bot_manager.RedisStore
	// Continue rematch cancel deadline (seconds); same config source as former continue queue timeout.
	continueRematchCancelTimeoutSec    int64
	continueRematchCancelRedundancySec int64
	publisher                          EventPublisher
	closing                            bool
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
	botStore, err := bot_manager.NewRedisStore("")
	if err != nil {
		return nil, fmt.Errorf("lobby redis bots: %w", err)
	}
	q.botStore = botStore
	q.registerPendingMatchConfirmationTimeoutHandler()
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
	q.lock.Lock()
	defer q.lock.Unlock()
	q.closing = true
}

func (q *Queue) HandleJoinQueueEvent(req *pb.PlayerAddress) error {
	var address types.PlayerAddress
	address.FromProto(req)
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.closing {
		log.Debugw("cannot join queue, server is closing", "addr", address.String())
		return errors.New("server is closing")
	}
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
	q.lock.Lock()
	defer q.lock.Unlock()
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
	bots := Set[types.PlayerAddress]{}
	q.lock.Lock()
	for _, p := range event.GameInfo.Players {
		addr := types.NewPlayerAddress(p.PlayerId, p.TemporaryAddress)
		q.markPlayerOutOfGameLocked(*addr)
		isBot := q.releaseInGameBotLocked(*addr)
		if isBot {
			bots.Add(*addr)
		}
	}
	q.lock.Unlock()
	err := db.BattleResultSettlement(event.GameInfo, bots)
	if err != nil {
		log.Errorw("BattleResultSettlement failed", "err", err)
		return err
	}
	if event.GameInfo.Status == pb.GameStatus_GAME_ABORTED {
		return nil
	}

	q.publishGameSettlementResult(event.GameInfo)

	if len(bots) > 0 {
		log.Infow("skipping continue rematch: game included bots", "game_id", event.GameInfo.ID)
		q.publishNotMatchableForHumans(event.GameInfo, event.GameInfo.ID, bots)
	} else if q.anyHumanPlayerBelowQueueThreshold(event.GameInfo, bots) {
		log.Infow("skipping continue rematch: insufficient tokens after settlement", "game_id", event.GameInfo.ID)
		q.publishNotMatchableForHumans(event.GameInfo, event.GameInfo.ID, bots)
	} else {
		q.lock.Lock()
		q.tryStartContinueRematchAfterGame(event.GameInfo)
		q.lock.Unlock()
	}

	go func() {
		if q.statSvcClient == nil {
			return
		}
		playerIDs := make([]int64, 0, len(event.GameInfo.Players))
		for _, p := range event.GameInfo.Players {
			playerIDs = append(playerIDs, p.PlayerId)
		}
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

func (q *Queue) markPlayerOutOfGameLocked(address types.PlayerAddress) {
	address.TemporaryAddress = strings.ToLower(address.TemporaryAddress)
	if err := q.lobbyState.MarkPlayersOutOfGame(q.ctx, address); err != nil {
		log.Errorw("mark player out of game in redis failed", "player", address.String(), "err", err)
	}
}

func (q *Queue) lockToken(address *types.PlayerAddress) error {
	log.Infow("lock user token", "addr", address.String(), "token amount", q.minTokenToJoinQueue)
	return db.LockUserToken(q.ctx, address.Id, address.TemporaryAddress, q.minTokenToJoinQueue)
}

func (q *Queue) unlockToken(address *types.PlayerAddress) error {
	log.Infow("unlock user token", "addr", address.String(), "token amount", q.minTokenToJoinQueue)
	return db.UnlockUserToken(q.ctx, address.Id, address.TemporaryAddress)
}

func (q *Queue) publishGameSettlementResult(game *dao.Game) {
	if game == nil || game.GameResult == nil || game.GameResult.BattleReward == nil {
		return
	}
	br := game.GameResult.BattleReward
	playerRewards := conversion.DbPlayerRewardsToProto(br.PlayerRewards)
	receivers := make([]*pb.PlayerAddress, 0, len(game.Players))
	for _, p := range game.Players {
		if p == nil {
			continue
		}
		receivers = append(receivers, types.NewPlayerAddress(p.PlayerId, p.TemporaryAddress).ToProto())
	}
	out := &pb.Event{
		Type:      pb.EventType_TYPE_GAME_SETTLEMENT_RESULT,
		Receivers: receivers,
		Event: &pb.Event_GameSettlementResult{
			GameSettlementResult: &pb.GameSettlementResult{
				GameId:        game.ID,
				SystemFee:     br.SystemFee,
				PlayerRewards: playerRewards,
			},
		},
	}
	if err := pubsub.Publish(q.ctx, q.publisher, pubsub.TopicLobby, out); err != nil {
		log.Errorw("publish game settlement result failed", "topic", pubsub.TopicLobby, "game_id", game.ID, "err", err)
	}
}

func (q *Queue) publishNotMatchableForHumans(game *dao.Game, lastGameID int64, bots Set[types.PlayerAddress]) {
	receivers := make([]*pb.PlayerAddress, 0, len(game.Players))
	for _, p := range game.Players {
		if p == nil {
			continue
		}
		addr := types.NewPlayerAddress(p.PlayerId, p.TemporaryAddress)
		if bots.Contains(*addr) {
			continue
		}
		receivers = append(receivers, addr.ToProto())
	}
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
