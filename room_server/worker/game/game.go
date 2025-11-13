package game

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type gamePlayer struct {
	player      *dao.GamePlayerInfo
	roundPlayer *dao.PlayerRoundInfo
	totalLostHP int64
	currentHP   int64
}

func (p *gamePlayer) PlayerAddress() types.PlayerAddress {
	addr := types.PlayerAddress{}
	addr.FromDao(*p.player)
	return addr
}

func (p *gamePlayer) String() string {
	return fmt.Sprintf("%s_%s", p.player.WalletAddress, p.player.TemporaryAddress)
}

type Game struct {
	ctx                 context.Context
	lock                sync.RWMutex
	gameInfo            *dao.Game
	*Round              // Embedded Round for battle execution
	workerMangerService *worker.WorkerManager
	chainSvc            ContractClient
	gameContextHandler  GameHandler
	wg                  *sync.WaitGroup
}

func NewGame(
	ctx context.Context,
	wg *sync.WaitGroup,
	players []types.PlayerAddress,
	workerMangerService *worker.WorkerManager,
	chainSvc ContractClient,
	gameContinuer GameHandler,
	gameArgs *dao.GameArgs) *Game {
	daoPlayers := make([]*dao.GamePlayerInfo, 0, len(players))
	gamePlayers := make(map[string]*gamePlayer)
	for _, player := range players {
		daoPlayer := player.ToDao()
		daoPlayers = append(daoPlayers, daoPlayer)
		gamePlayers[player.TemporaryAddress] = &gamePlayer{
			player:    daoPlayer,
			currentHP: gameArgs.InitialHP,
		}
	}
	game := &Game{
		ctx: ctx,
		wg:  wg,
		gameInfo: &dao.Game{
			Players:  daoPlayers,
			Type:     types.GameTypePVP,
			GameArgs: *gameArgs,
		},
		Round:               &Round{round: nil, gamePlayers: gamePlayers, battleStates: nil},
		workerMangerService: workerMangerService,
		chainSvc:            chainSvc,
		gameContextHandler:  gameContinuer,
	}
	game.setupNewRound()
	wg.Add(1)
	return game
}

func NewGameFromGameInfo(
	ctx context.Context,
	wg *sync.WaitGroup,
	workerMangerService *worker.WorkerManager,
	gameContinuer GameHandler,
	gameInfo *dao.Game,
	chainSvc ContractClient) *Game {
	gamePlayers := make(map[string]*gamePlayer)
	g := &Game{
		ctx:                 ctx,
		wg:                  wg,
		gameInfo:            gameInfo,
		Round:               &Round{round: nil, gamePlayers: gamePlayers, battleStates: nil},
		workerMangerService: workerMangerService,
		chainSvc:            chainSvc,
		gameContextHandler:  gameContinuer,
	}
	wg.Add(1)
	var terminateGame = func() {
		log.Errorw("game expired, terminate", "game id", gameInfo.ID, "status", gameInfo.Status)
		if gameInfo.Status == proto.GameStatus_GAME_INIT {
			err := g.handleGameAbortInit()
			if err != nil {
				log.Errorf("expired game abort failed, game: %d, err %s", gameInfo.ID, err)
			}
		} else {
			err := g.handleRoundEnd(proto.RoundCompleteReason_ROUND_COMPLETE_SERVER_INTERNAL_TIMEOUT)
			if err != nil {
				log.Errorf("expired game terminate failed, game: %d, err %s", gameInfo.ID, err)
			}
		}
	}
	shouldTerminate := false
	if time.Since(gameInfo.CreatedAt) > time.Duration(gameInfo.GameArgs.RoundTimeout)*time.Second*time.Duration(gameInfo.GameArgs.MaxRounds) {
		shouldTerminate = true
	}

	for _, playerInfo := range g.gameInfo.Players {
		g.setGamePlayer(playerInfo.TemporaryAddress, &gamePlayer{
			player:    playerInfo,
			currentHP: g.gameInfo.InitialHP,
		})
	}
	if len(g.gameInfo.Rounds) != 0 {
		roundNum := uint32(0)
		sort.Slice(g.gameInfo.Rounds, func(i, j int) bool {
			return g.gameInfo.Rounds[i].RoundNumber < g.gameInfo.Rounds[j].RoundNumber
		})
		for _, r := range g.gameInfo.Rounds {
			if r.RoundNumber > roundNum {
				roundNum = r.RoundNumber
				g.Round.round = r
			}
		}
		// Initialize game state based on submitted cards
		// Initialize turnNumber from submitted cards count if available
		if len(g.Round.round.PlayerRoundInfos) > 0 && len(g.Round.round.PlayerRoundInfos[0].SubmittedCards) > 0 {
			g.Round.turnNumber = uint32(len(g.Round.round.PlayerRoundInfos[0].SubmittedCards)) + 1
		} else {
			g.Round.turnNumber = 1
		}
		for _, roundPlayer := range g.Round.round.PlayerRoundInfos {
			player, err := g.getGamePlayer(roundPlayer.TemporaryAddress)
			if err != nil {
				// should never happen
				log.Fatalf("getGamePlayer failed, err: %v", err)
			}
			player.roundPlayer = roundPlayer
			if len(player.roundPlayer.SubmittedCards) != 0 {
				player.currentHP = currentHpFromCards(player.roundPlayer.SubmittedCards)
			}
			player.totalLostHP = int64(player.roundPlayer.LostHP)
		}
		if shouldTerminate {
			terminateGame()
			return nil
		}
		if g.Round.round.Status == proto.RoundStatus_ROUND_COMPLETED {
			g.setupNewRound()
		} else {
			g.sendTimerEventByCurrentRound()
		}
	} else {
		g.setupNewRound()
	}

	return g
}

func (g *Game) GetBattleInfo(roundNum uint32) (*proto.RoundResult, *proto.GameResult) {
	g.lock.RLock()
	defer g.lock.RUnlock()
	var gameRes *proto.GameResult
	if g.gameInfo.GameResult != nil {
		gameRes = conversion.DbGameResultToProtoGameResult(g.gameInfo.GameResult)
		gameRes.GameContinueTimeout = uint64(g.gameInfo.GameArgs.ContinueTimeout)
	}
	for _, round := range g.gameInfo.Rounds {
		if round.RoundNumber == (roundNum) {
			roundRes := conversion.DbRoundToRoundResult(round)
			roundRes.RoundConfirmTimeout = uint64(g.gameInfo.GameArgs.RoundConfirmTimeout)
			return roundRes, gameRes
		}
	}
	return nil, nil
}

func (g *Game) ToProto() *proto.GameInfo {
	g.lock.RLock()
	defer g.lock.RUnlock()
	gameProto := conversion.DbGameInfoToProtoGameInfo(g.gameInfo)
	return gameProto
}

func (g *Game) GetGameResult() *proto.GameResult {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return conversion.DbGameResultToProtoGameResult(g.gameInfo.GameResult)
}

func (g *Game) GetGamePhase() *proto.GamePhase {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return conversion.DbGameToProtoGamePhase(g.gameInfo, g.Round.round)
}

func (g *Game) saveGame() error {
	err := db.SaveGame(g.gameInfo)
	if err != nil {
		log.Errorw("saveGame failed", "err", err, "game id", g.gameInfo.ID)
	}
	return nil
}

func (g *Game) savePlayerRoundInfo(roundPlayer *dao.PlayerRoundInfo) error {
	err := db.SavePlayerRoundInfo(roundPlayer)
	if err != nil {
		log.Errorw("savePlayerRoundInfo failed", "err", err, "game id", g.gameInfo.ID, "round num", g.Round.round.RoundNumber)
	}
	return nil
}

func (g *Game) saveRound(round *dao.Round) error {
	err := db.SaveRound(round)
	if err != nil {
		log.Errorw("saveRound failed", "err", err, "game id", g.gameInfo.ID, "round num", round.RoundNumber)
	}
	return nil
}

// pushStateToContractCreating is used for continue games to immediately start contract creation
// It marks all players as ready and initiates contract creation
func (g *Game) pushStateToContractCreating() error {
	g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
	allPlayers := make([]types.PlayerAddress, 0, len(g.Round.gamePlayers))
	for _, player := range g.Round.gamePlayers {
		allPlayers = append(allPlayers, player.PlayerAddress())
		player.roundPlayer.PlayerReady = true
	}
	if err := g.sendContractCreation(allPlayers); err != nil {
		g.handleGameAbortInternalError()
		return err
	}
	g.Round.round.Status = proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN
	return nil
}

// getCurrentTurnNumber returns the current turn number (1-3) from the Round struct
func (g *Game) getCurrentTurnNumber() uint32 {
	if g.Round == nil || g.Round.turnNumber == 0 {
		return 1
	}
	return g.Round.turnNumber
}

// setupNewTurn sends event to chain manager to setup a new turn
// Note: For the first turn of the first round, this is not needed as the contract creation handles it
func (g *Game) setupNewTurn() error {
	// Skip for the first turn of the first round
	if g.Round.round.RoundNumber == 1 && g.getCurrentTurnNumber() == 1 {
		return nil
	}
	if g.gameInfo.RoomContract == "" {
		return errors.New("room contract empty, cannot setup new turn")
	}
	turnNumber := g.getCurrentTurnNumber()
	log.Infow("setup new turn", "game id", g.gameInfo.ID, "round number", g.Round.round.RoundNumber, "turn number", turnNumber)
	err := g.sendTurnReady()
	if err != nil {
		return err
	}
	g.Round.round.Status = proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN
	return nil
}

// incrementTurnNumber increments the turn number for the current round
func (g *Game) incrementTurnNumber() {
	g.Round.turnNumber++
}

func (g *Game) stopGame() {
	g.workerMangerService.CloseWorker(g.workerID())
	g.wg.Done()
}

func (g *Game) createSelf() {
	g.workerMangerService.SpwanWorker(g.ctx, g.workerID(), types.WORKER_TYPE_GAME, g)
}

func (g *Game) workerID() string {
	return fmt.Sprint(g.gameInfo.ID)
}

func (g *Game) setupNewRound() {
	roundNum := uint32(1)
	if g.Round.round != nil {
		roundNum = g.Round.round.RoundNumber + 1
	}
	newRound := &dao.Round{
		GameID:      g.gameInfo.ID,
		RoundNumber: roundNum,
		Status:      proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION,
	}
	for _, player := range g.Round.gamePlayers {
		playerRoundInfo := &dao.PlayerRoundInfo{
			WalletAddress:    player.player.WalletAddress,
			TemporaryAddress: player.player.TemporaryAddress,
			SubmittedCards:   make([]*dao.RoundSubmittedCard, 0),
		}
		newRound.PlayerRoundInfos = append(newRound.PlayerRoundInfos, playerRoundInfo)
		player.roundPlayer = playerRoundInfo
	}
	g.Round.round = newRound // Update the embedded Round's reference
	g.Round.turnNumber = 1   // Start with turn 1 for each new round
	g.gameInfo.Rounds = append(g.gameInfo.Rounds, newRound)
	g.sendTimerEventByCurrentRound()
}

func (g *Game) sendEventsToAllPlayers(events ...*types.Event) {
	for _, player := range g.Round.gamePlayers {
		for _, event := range events {
			g.workerMangerService.SendEvent(player.String(), event)
		}
	}
}

func (g *Game) setGamePlayer(tempAddr string, player *gamePlayer) {
	g.Round.gamePlayers[strings.ToLower(tempAddr)] = player
}

func (g *Game) getGamePlayer(tempAddr string) (*gamePlayer, error) {
	player, ok := g.Round.gamePlayers[strings.ToLower(tempAddr)]
	if !ok {
		return nil, fmt.Errorf("player %s not found", tempAddr)
	}
	return player, nil
}

func (g *Game) sendContractCreation(allPlayers []types.PlayerAddress) error {
	return g.chainSvc.CreateRoomContract(&types.RequireContractCreationEvent{
		GameID:         g.gameInfo.ID,
		Players:        allPlayers,
		InitialHP:      g.gameInfo.InitialHP,
		RoundTimeout:   g.gameInfo.RoundTimeout,
		MaxRoundNumber: g.gameInfo.MaxRounds,
	})
}

func (g *Game) sendTurnReady() error {
	return g.chainSvc.SetTurnReady(&types.RequireSetupNewTurnEvent{
		GameID:          g.gameInfo.ID,
		ContractAddress: g.gameInfo.RoomContract,
		RoundNumber:     uint32(g.Round.round.RoundNumber),
		TurnNumber:      g.getCurrentTurnNumber(),
	})
}

func (g *Game) abortedGameResult() *dao.GameResult {
	gameRes := &dao.GameResult{
		GameResultType: proto.GameResultType_GAME_ABORTED,
		BattleReward: &dao.BattleReward{
			PlayerRewards: []*dao.PlayerReward{},
		},
	}
	for _, player := range g.Round.gamePlayers {
		playerReward := &dao.PlayerReward{
			WalletAddress:    player.player.WalletAddress,
			TemporaryAddress: player.player.TemporaryAddress,
		}
		gameRes.BattleReward.PlayerRewards = append(gameRes.BattleReward.PlayerRewards, playerReward)
	}
	return gameRes
}

func currentHpFromCards(cards []*dao.RoundSubmittedCard) int64 {
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].CardNumber < cards[j].CardNumber
	})
	lastCard := cards[len(cards)-1]
	return int64(lastCard.HealthAfter)
}
