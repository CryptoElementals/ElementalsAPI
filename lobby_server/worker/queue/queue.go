package queue

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/cmd/ele-stat/proto"
	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
)

// EventPublisher publishes outbound player notifications (same contract as game.Publisher).
type EventPublisher = pubsub.Publisher

const queueInfoPrefix = "queue_info"
const lockedTokenPrefix = "locked_token"
const queueInfoVal = "v"

type Queue struct {
	ctx                  context.Context
	lock                 sync.RWMutex
	queue                map[types.PlayerAddress]time.Time
	pendingMatchByPlayer map[types.PlayerAddress]int64
	inGameByPlayer       map[types.PlayerAddress]bool
	// Continue rematch cancel deadline (seconds); same config source as former continue queue timeout.
	continueRematchCancelTimeoutSec    int64
	continueRematchCancelRedundancySec int64
	publisher                          EventPublisher
	queueCache                         cache.Cache
	lockedTokenCache                   cache.Cache
	closing                            bool
	gameCreator                        GameCreator
	minTokenToJoinQueue                int32
	matchConfirmationTimeout           int64

	botMgr      *botManager
	botWaitTime time.Duration

	statServiceEndpoint string
	statSvcClient       proto.StatServiceClient
}

type GameCreator interface {
	CreateGameAndRun(players []types.PlayerAddress, gameType uint, completedMatchID int64) (int64, error)
}

func NewQueue(
	ctx context.Context,
	pub EventPublisher,
	c cache.Cache,
	gameCreator GameCreator,
	matchConfirmationTimeout int64,
	continueTimeout int64,
	continueTimeoutRedundancy int64,
	botWaitTime int64,
	minTokenToJoinQueue int32,
	statServiceEndpoint string,
) *Queue {
	queueCache := cache.WithPrefix(queueInfoPrefix, c)
	tokenCache := cache.WithPrefix(lockedTokenPrefix, c)
	q := &Queue{
		ctx:                                ctx,
		queue:                              make(map[types.PlayerAddress]time.Time),
		pendingMatchByPlayer:               make(map[types.PlayerAddress]int64),
		inGameByPlayer:                     make(map[types.PlayerAddress]bool),
		publisher:                          pub,
		lockedTokenCache:                   tokenCache,
		queueCache:                         queueCache,
		gameCreator:                        gameCreator,
		continueRematchCancelTimeoutSec:    continueTimeout,
		continueRematchCancelRedundancySec: continueTimeoutRedundancy,
		botMgr:                             newBotManager(),
		botWaitTime:                        time.Duration(botWaitTime) * time.Second,
		minTokenToJoinQueue:                minTokenToJoinQueue,
		matchConfirmationTimeout:           matchConfirmationTimeout,
		statServiceEndpoint:                statServiceEndpoint,
	}
	q.registerPendingMatchConfirmationTimeoutHandler()
	return q
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
	keys, err := q.queueCache.List("")
	if err != nil {
		return err
	}
	for _, key := range keys {
		var player types.PlayerAddress
		if err := player.Parse(key); err != nil {
			return err
		}
		q.queue[player] = time.Now()
	}
	q.addBotRoutine()
	return nil
}

func (q *Queue) close() {
	q.lock.Lock()
	defer q.lock.Unlock()
	// drain the queue when closing
	q.closing = true
	for addr := range q.queue {
		q.removePlayerFromQueue(addr)
	}
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
	if _, ok := q.queue[address]; ok {
		return errors.New("player already in queue")
	}
	log.Infow("join queue", "player id", address.Id, "temporary address", address.TemporaryAddress)
	err := q.lockToken(&address)
	if err != nil {
		log.Errorf("cannot join queue, err: %s", err.Error())
		return err
	}
	matched := false
	for player := range q.queue {
		if player.Id == address.Id ||
			player.TemporaryAddress == address.TemporaryAddress {
			continue
		}

		err := q.matchPlayers([]types.PlayerAddress{address, player})
		if err != nil {
			return err
		}
		matched = true
		break
	}
	if !matched {
		q.queue[address] = time.Now()
		err := q.queueCache.Set(address.String(), queueInfoVal, 0)
		if err != nil {
			log.Errorf("set player to queue cache failed: %s", err.Error())
		}
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
	_, ok := q.queue[address]
	if !ok {
		log.Debugw("player not in queue", "player", address.String())
		return nil
	}
	return q.removePlayerFromQueue(address)
}

func (q *Queue) removePlayerFromQueue(player types.PlayerAddress) error {
	delete(q.queue, player)
	q.queueCache.Delete(player.String())
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
		isBot := q.botMgr.releaseInGameBot(*addr)
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

	if q.anyHumanPlayerBelowQueueThreshold(event.GameInfo, bots) {
		log.Infow("skipping continue rematch: insufficient tokens after settlement", "game_id", event.GameInfo.ID)
		q.publishNotMatchableForHumans(event.GameInfo, event.GameInfo.ID, bots)
	} else {
		q.lock.Lock()
		q.tryStartContinueRematchAfterGame(event.GameInfo, bots)
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
	q.lock.RLock()
	defer q.lock.RUnlock()
	_, ok := q.queue[address]
	return ok
}

func (q *Queue) isPlayerInGame(address types.PlayerAddress) bool {
	address.TemporaryAddress = strings.ToLower(address.TemporaryAddress)
	q.lock.RLock()
	defer q.lock.RUnlock()
	return q.inGameByPlayer[address]
}

func (q *Queue) markPlayerInGameLocked(address types.PlayerAddress) {
	address.TemporaryAddress = strings.ToLower(address.TemporaryAddress)
	q.inGameByPlayer[address] = true
}

func (q *Queue) markPlayerOutOfGameLocked(address types.PlayerAddress) {
	address.TemporaryAddress = strings.ToLower(address.TemporaryAddress)
	delete(q.inGameByPlayer, address)
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
	protoReward := conversion.DbBattleRewardToProtoBattleReward(game.GameResult.BattleReward)
	if protoReward == nil {
		return
	}
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
				GameId: game.ID,
				Reward: protoReward,
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
