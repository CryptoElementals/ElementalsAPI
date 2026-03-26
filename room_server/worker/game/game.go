package game

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type Game struct {
	ctx                 context.Context
	gameInfo            *dao.Game
	currentRound        *round
	workerMangerService *worker.WorkerManager
	txPoolEnqueuer      TxPoolEnqueuer
	gameResultSettler   GameResultSettler
	gameContextHandler  GameHandler
	wg                  *sync.WaitGroup
}

func NewGame(
	ctx context.Context,
	wg *sync.WaitGroup,
	players []types.PlayerAddress,
	workerMangerService *worker.WorkerManager,
	txPoolEnqueuer TxPoolEnqueuer,
	gameResultSettler GameResultSettler,
	gameContinuer GameHandler,
	gameArgs *dao.GameArgs) *Game {
	daoPlayers := make([]*dao.GamePlayerInfo, 0, len(players))
	gamePlayers := make(map[string]*gamePlayer)
	for _, player := range players {
		daoPlayer := player.ToDao()
		daoPlayers = append(daoPlayers, daoPlayer)
		gamePlayers[player.TemporaryAddress] = &gamePlayer{
			player:     daoPlayer,
			currentHP:  gameArgs.InitialHP,
			multiplier: uint32(gameArgs.InitialMultiplier),
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
		currentRound:        &round{round: nil, gamePlayers: gamePlayers},
		workerMangerService: workerMangerService,
		txPoolEnqueuer:      txPoolEnqueuer,
		gameResultSettler:   gameResultSettler,
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
	txPoolEnqueuer TxPoolEnqueuer,
	gameResultSettler GameResultSettler) *Game {
	// Initialize gamePlayers from gameInfo.Players
	gamePlayers := make(map[string]*gamePlayer)
	for _, playerInfo := range gameInfo.Players {
		gamePlayers[playerInfo.TemporaryAddress] = &gamePlayer{
			player:     playerInfo,
			currentHP:  gameInfo.InitialHP,
			multiplier: uint32(gameInfo.InitialMultiplier),
		}
	}

	g := &Game{
		ctx:                 ctx,
		wg:                  wg,
		gameInfo:            gameInfo,
		currentRound:        &round{round: nil, gamePlayers: gamePlayers},
		workerMangerService: workerMangerService,
		txPoolEnqueuer:      txPoolEnqueuer,
		gameResultSettler:   gameResultSettler,
		gameContextHandler:  gameContinuer,
	}

	// Setup current round to the last round of the gameInfo, if not exist, create one
	if len(gameInfo.Rounds) > 0 {
		// Find the round with the highest RoundNumber
		var lastRound *dao.Round
		maxRoundNum := uint32(0)
		for _, r := range gameInfo.Rounds {
			if r.RoundNumber > maxRoundNum {
				maxRoundNum = r.RoundNumber
				lastRound = r
			}
		}
		if lastRound != nil {
			g.currentRound.round = lastRound
		} else {
			// If no valid round found, create a new one
			g.setupNewRound()
		}
	} else {
		// If no rounds exist, create a new one
		g.setupNewRound()
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
			err := g.handleGameAbortInternalError()
			if err != nil {
				log.Errorf("expired game abort failed, game: %d, err %s", gameInfo.ID, err)
			}
		}
	}
	terminateGame()
	return nil
	// if time.Since(gameInfo.CreatedAt) > time.Duration(gameInfo.GameArgs.RoundTimeout)*time.Second*time.Duration(gameInfo.GameArgs.MaxRounds) {

	// }

	// for _, playerInfo := range g.gameInfo.Players {
	// 	g.setGamePlayer(playerInfo.TemporaryAddress, &gamePlayer{
	// 		player:    playerInfo,
	// 		currentHP: g.gameInfo.InitialHP,
	// 	})
	// }
	// if len(g.gameInfo.Rounds) != 0 {
	// 	roundNum := uint32(0)
	// 	sort.Slice(g.gameInfo.Rounds, func(i, j int) bool {
	// 		return g.gameInfo.Rounds[i].RoundNumber < g.gameInfo.Rounds[j].RoundNumber
	// 	})
	// 	for _, r := range g.gameInfo.Rounds {
	// 		if r.RoundNumber > roundNum {
	// 			roundNum = r.RoundNumber
	// 			g.currentRound.round = r
	// 		}
	// 	}
	// 	// Initialize game state from Turns
	// 	// Find the latest turn to determine current turn number
	// 	// Since Turns are stored by index (turn 1 at index 0, turn 2 at index 1, turn 3 at index 2),
	// 	// we can find the latest turn by checking the highest index with a non-nil entry
	// 	latestTurnNumber := uint32(0)
	// 	for i := len(g.currentRound.round.Turns) - 1; i >= 0; i-- {
	// 		if g.currentRound.round.Turns[i] != nil {
	// 			latestTurnNumber = g.currentRound.round.Turns[i].TurnNumber
	// 			break
	// 		}
	// 	}
	// 	if latestTurnNumber > 0 {
	// 		g.currentRound.turnNumber = latestTurnNumber + 1
	// 		if g.currentRound.turnNumber > 3 {
	// 			g.currentRound.turnNumber = 1 // Round completed, will setup new round
	// 		}
	// 	} else {
	// 		g.currentRound.turnNumber = 1
	// 	}

	// 	// Reconstruct playerTurnInfos from Turns for runtime use
	// 	// Group PlayerTurnInfos by player, maintaining sorted order by turn number
	// 	// Since Turns are stored by index (turn 1 at index 0, turn 2 at index 1, turn 3 at index 2),
	// 	// we use turn.TurnNumber - 1 as the index to maintain sorted order in playerTurnInfos
	// 	playerTurnInfoMap := make(map[string][]*dao.PlayerTurnInfo)
	// 	for _, turn := range g.currentRound.round.Turns {
	// 		if turn == nil {
	// 			continue
	// 		}
	// 		idx := int(turn.TurnNumber) - 1
	// 		for _, playerTurnInfo := range turn.PlayerTurnInfos {
	// 			key := playerTurnInfo.TemporaryAddress
	// 			// Initialize slice if needed
	// 			if playerTurnInfoMap[key] == nil {
	// 				playerTurnInfoMap[key] = make([]*dao.PlayerTurnInfo, 0, 3)
	// 			}
	// 			// Ensure slice is large enough and store at correct index to maintain sorted order
	// 			for len(playerTurnInfoMap[key]) <= idx {
	// 				playerTurnInfoMap[key] = append(playerTurnInfoMap[key], nil)
	// 			}
	// 			playerTurnInfoMap[key][idx] = playerTurnInfo
	// 		}
	// 	}

	// 	// Assign reconstructed playerTurnInfos to gamePlayers
	// 	for key, turnInfos := range playerTurnInfoMap {
	// 		player, err := g.getGamePlayer(key)
	// 		if err != nil {
	// 			// should never happen
	// 			log.Fatalf("getGamePlayer failed, err: %v", err)
	// 		}
	// 		player.playerTurnInfos = turnInfos
	// 		// Calculate current HP and lost HP from submitted cards
	// 		submittedCards := player.getSubmittedCards()
	// 		if len(submittedCards) != 0 {
	// 			player.currentHP = currentHpFromCards(submittedCards)
	// 		}
	// 		player.totalLostHP = int64(player.getLostHP())
	// 	}

	// 	// Check if round is completed (CompleteReason is set or IsLastRound is true)
	// 	if g.currentRound.round.IsLastRound || g.currentRound.round.CompleteReason != proto.RoundCompleteReason_ROUND_COMPLETE_NORMAL {
	// 		g.currentRound.turnStatus = proto.TurnStatus_TURN_ROUND_COMPLETED
	// 		g.setupNewRound()
	// 	} else {
	// 		// Determine status from turns
	// 		if len(g.currentRound.round.Turns) == 0 {
	// 			g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION
	// 		} else {
	// 			// Default to waiting commitments if turns exist
	// 			g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_COMMITMENTS
	// 		}
	// 		g.sendTimerEventByCurrentRound()
	// 	}
	// } else {
	// 	g.setupNewRound()
	// }

	// return g
}

func (g *Game) saveGame() error {
	err := db.SaveGame(g.gameInfo)
	if err != nil {
		log.Errorw("saveGame failed", "err", err, "game id", g.gameInfo.ID)
		return err
	}
	return nil
}

func (g *Game) saveRound(round *dao.Round) error {
	err := db.SaveRound(round)
	if err != nil {
		log.Errorw("saveRound failed", "err", err, "game id", g.gameInfo.ID, "round num", round.RoundNumber)
		return err
	}
	return nil
}

// pushStateToContractCreating is used for continue games to immediately start contract creation
// It marks all players as ready and initiates contract creation
func (g *Game) pushStateToContractCreating() error {
	g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
	allPlayers := make([]types.PlayerAddress, 0, len(g.currentRound.gamePlayers))
	for _, player := range g.currentRound.gamePlayers {
		allPlayers = append(allPlayers, player.PlayerAddress())
		// Mark player as ready for current turn
		player.currentTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_READY
	}
	if err := g.sendContractCreation(allPlayers); err != nil {
		g.handleGameAbortInternalError()
		return err
	}
	g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN
	return nil
}

// setupNewTurn sends event to chain manager to setup a new turn
// Note: For the first turn of the first round, this is not needed as the contract creation handles it
func (g *Game) setupNewTurn() error {
	// RoomContract check removed - always uses RoomV2 contract address
	turnNumber := g.currentRound.getCurrentTurnNumber()
	log.Infow("setup new turn", "game id", g.gameInfo.ID, "round number", g.currentRound.round.RoundNumber, "turn number", turnNumber)
	err := g.sendTurnReady()
	if err != nil {
		return err
	}
	g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN
	return nil
}

// incrementTurnNumber increments the turn number for the current round
func (g *Game) incrementTurnNumber() {
	g.currentRound.turnNumber++
}

func (g *Game) stopGame() {
	log.Infow("stop game", "game id", g.gameInfo.ID)
	g.workerMangerService.CloseWorker(g.workerID())
	g.wg.Done()
}

func (g *Game) createSelf() {
	g.workerMangerService.SpwanWorker(g.ctx, g.workerID(), types.WORKER_TYPE_GAME, g)
}

func (g *Game) workerID() string {
	return fmt.Sprint(g.gameInfo.ID)
}

// WorkerID returns the worker ID for this game (exported version)
func (g *Game) WorkerID() string {
	return g.workerID()
}

func (g *Game) setupNewRound() {
	roundNum := uint32(1)
	if g.currentRound.round != nil {
		roundNum = g.currentRound.round.RoundNumber + 1
	}
	newRound := &dao.Round{
		GameID:      g.gameInfo.ID,
		RoundNumber: roundNum,
		Turns:       make([]*dao.Turn, 0, 3), // Pre-allocate for 3 turns
	}
	g.currentRound.round = newRound // Update the embedded Round's reference
	g.currentRound.turnNumber = 1   // Start with turn 1 for each new round
	g.currentRound.createNewTurn()
	g.gameInfo.Rounds = append(g.gameInfo.Rounds, newRound)
	g.sendTimerEventByCurrentRound()
}

func (g *Game) sendEventsToAllPlayers(events ...*types.Event) {
	for _, player := range g.currentRound.gamePlayers {
		for _, event := range events {
			g.workerMangerService.SendEvent(player.String(), event)
		}
	}
}

func (g *Game) getGamePlayer(tempAddr string) (*gamePlayer, error) {
	player, ok := g.currentRound.gamePlayers[strings.ToLower(tempAddr)]
	if !ok {
		return nil, fmt.Errorf("player %s not found", tempAddr)
	}
	return player, nil
}

func (g *Game) sendContractCreation(allPlayers []types.PlayerAddress) error {
	evt := &types.RequireGameCreationEvent{
		GameID:         g.gameInfo.ID,
		Players:        allPlayers,
		InitialHP:      g.gameInfo.InitialHP,
		RoundTimeout:   g.gameInfo.CommitmentSubmissionTimeout,
		MaxRoundNumber: g.gameInfo.MaxRounds,
	}
	g.txPoolEnqueuer.AddCreateRoom(evt)
	return nil
}

func (g *Game) sendTurnReady() error {
	evt := &types.RequireSetupNewTurnEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: uint32(g.currentRound.round.RoundNumber),
		TurnNumber:  g.currentRound.getCurrentTurnNumber(),
	}
	g.txPoolEnqueuer.AddSetTurnReady(evt)
	return nil
}

// completeGameAndNotify runs settlement, clears tx pool info for this game, then notifies the manager to remove the game from maps.
func (g *Game) completeGameAndNotify(evt *types.GameCompletedEvent) error {
	if g.gameResultSettler != nil {
		if err := g.gameResultSettler.GameResultSettlement(evt); err != nil {
			return err
		}
	}
	g.txPoolEnqueuer.ClearGameInfo(evt.GameID)
	return g.gameContextHandler.HandleGameCompletedEvent(evt)
}
