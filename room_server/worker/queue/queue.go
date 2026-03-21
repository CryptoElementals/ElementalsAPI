package queue

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/cmd/ele-stat/proto"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
)

const queueInfoPrefix = "queue_info"
const lockedTokenPrefix = "locked_token"
const queueInfoVal = "v"

type Queue struct {
	ctx                 context.Context
	lock                sync.RWMutex
	queue               map[types.PlayerAddress]time.Time
	continueManager     *continueManager
	workerManager       *worker.WorkerManager
	publisher           EventPublisher
	queueCache          cache.Cache
	lockedTokenCache    cache.Cache
	closing             bool
	gameCreator         GameCreator
	minTokenToJoinQueue int32

	botMgr      *botManager
	botWaitTime time.Duration

	statServiceEndpoint string
	statSvcClient       proto.StatServiceClient
}

type GameCreator interface {
	HandleGameMatchedEvent(evt *types.GameMatchedEvent) (uint, error)
	HandleGameContinueEvent(evt *types.GameContinueEvent) error
}

func NewQueue(
	ctx context.Context,
	workerManager *worker.WorkerManager,
	pub EventPublisher,
	c cache.Cache,
	gameCreator GameCreator,
	continueTimeout int64,
	continueTimeoutRedundancy int64,
	botWaitTime int64,
	minTokenToJoinQueue int32,
	statServiceEndpoint string,
) *Queue {
	queueCache := cache.WithPrefix(queueInfoPrefix, c)
	tokenCache := cache.WithPrefix(lockedTokenPrefix, c)
	q := &Queue{
		ctx:                 ctx,
		queue:               make(map[types.PlayerAddress]time.Time),
		workerManager:       workerManager,
		publisher:           pub,
		lockedTokenCache:    tokenCache,
		queueCache:          queueCache,
		gameCreator:         gameCreator,
		continueManager:     newContinueManager(ctx, workerManager, pub, continueTimeout, continueTimeoutRedundancy),
		botMgr:              newBotManager(),
		botWaitTime:         time.Duration(botWaitTime) * time.Second,
		minTokenToJoinQueue: minTokenToJoinQueue,
		statServiceEndpoint: statServiceEndpoint,
	}
	return q
}

func (q *Queue) start() error {
	conn, err := grpc.DialContext(q.ctx, q.statServiceEndpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Errorf("failed to connect to stat service: %s", err.Error())
		return err
	}
	q.statSvcClient = proto.NewStatServiceClient(conn)
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
	q.continueManager.removeGameByAddress(address, 0)
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
	evt := &types.GameMatchedEvent{
		Players: players,
	}
	gid, err := q.gameCreator.HandleGameMatchedEvent(evt)
	if err != nil {
		log.Errorf("handle game matched event failed: %s", err.Error())
		return err
	}

	for _, p := range players {
		err = db.SetLockedTokenGameID(q.ctx, p.Id, p.TemporaryAddress, gid)
		if err != nil {
			// bot never lock token
			if err == db.ErrNotFound && q.botMgr.isInGame(p) {
				log.Debugw("bot token lock record not found when setting locked token game id", "player", p)
			} else {
				log.Errorf("set locked token game id failed: %s", err.Error())
				return err
			}

		}
		delete(q.queue, p)
		err = q.queueCache.Delete(p.String())
		if err != nil {
			log.Errorf("delete player from queue cache failed: %s", err.Error())
			return err
		}
	}
	return nil
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

	// Check if players have enough tokens to join queue after settlement
	// If not, send continue cancel event to both players
	shouldSendContinueCancel := false
	for _, p := range event.GameInfo.Players {
		addr := types.NewPlayerAddress(p.PlayerId, p.TemporaryAddress)
		// Skip bot accounts
		if _, ok := bots[*addr]; ok {
			continue
		}

		// Get user token to check available tokens
		userToken, err := db.GetPlayerToken(context.Background(), p.PlayerId)
		if err != nil {
			log.Errorw("failed to get player token after settlement", "player_id", p.PlayerId, "err", err)
			continue
		}

		// Calculate total locked tokens from userToken.LockedTokens
		var totalLockedTokens int32 = 0
		for _, locked := range userToken.LockedTokens {
			totalLockedTokens += locked.TokenAmount
		}
		availableTokens := int(userToken.TokenAmount) - int(totalLockedTokens)

		// Check if player has enough tokens to join queue
		if availableTokens < int(q.minTokenToJoinQueue) {
			log.Infow("player doesn't have enough tokens after settlement",
				"player_id", p.PlayerId,
				"available_tokens", availableTokens,
				"required_tokens", q.minTokenToJoinQueue)
			shouldSendContinueCancel = true
			break
		}
	}

	// If any player doesn't have enough tokens, send continue cancel event to both players
	if shouldSendContinueCancel {
		log.Infow("sending continue cancel events due to insufficient tokens",
			"game_id", event.GameInfo.ID)
		for _, p := range event.GameInfo.Players {
			addr := types.NewPlayerAddress(p.PlayerId, p.TemporaryAddress)
			// Skip bot accounts
			if _, ok := bots[*addr]; ok {
				continue
			}
			notifyContinueCanceled(q.ctx, q.publisher, *addr)
		}
	} else {
		q.continueManager.addGame(event.GameInfo)
	}
	go func() {
		palyerIds := make([]int64, 0, len(event.GameInfo.Players))
		for _, p := range event.GameInfo.Players {
			palyerIds = append(palyerIds, p.PlayerId)
		}

		resp, err := q.statSvcClient.UpdatePlayerStats(q.ctx, &proto.UpdatePlayerStatsRequest{
			PlayerIds: palyerIds,
		})
		if err != nil {
			log.Errorw("UpdatePlayerStats error", "err", err)
		} else {
			if !resp.Ok {
				log.Errorw("UpdatePlayerStats failed", "err", resp.Message)
			} else {
				log.Infow("UpdatePlayerStats success", "players", palyerIds)
			}
		}
	}()

	return nil
}

func (q *Queue) isPlayerInQueue(address types.PlayerAddress) bool {
	q.lock.RLock()
	defer q.lock.RUnlock()
	_, ok := q.queue[address]
	return ok
}

func (q *Queue) getPlayerContinueInfo(address types.PlayerAddress) *types.GameContinueInfo {
	info := q.continueManager.getPlayerContinueInfo(address)
	if info == nil {
		return nil
	}
	return info
}

func (q *Queue) lockToken(address *types.PlayerAddress) error {
	log.Infow("lock user token", "addr", address.String(), "token amount", q.minTokenToJoinQueue)
	return db.LockUserToken(q.ctx, address.Id, address.TemporaryAddress, q.minTokenToJoinQueue)
}

func (q *Queue) unlockToken(address *types.PlayerAddress) error {
	log.Infow("unlock user token", "addr", address.String(), "token amount", q.minTokenToJoinQueue)
	return db.UnlockUserToken(q.ctx, address.Id, address.TemporaryAddress)
}
