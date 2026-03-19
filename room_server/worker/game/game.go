package game

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// NewEphemeralGameForEvent creates a transient Game instance backed by the latest DB state.
// It is intended for synchronous event handling (e.g. from GameManager) rather than long-lived workers.
func NewEphemeralGameForEvent(
	ctx context.Context,
	workerMangerService *worker.WorkerManager,
	txPoolEnqueuer TxPoolEnqueuer,
	gameResultSettler GameResultSettler,
	gameInfo *dao.Game,
) *Game {
	return &Game{
		ctx:                 ctx,
		gameInfo:            gameInfo,
		currentRound:        buildRuntimeState(gameInfo),
		workerMangerService: workerMangerService,
		txPoolEnqueuer:      txPoolEnqueuer,
		gameResultSettler:   gameResultSettler,
	}
}

type Game struct {
	ctx                 context.Context
	gameInfo            *dao.Game
	currentRound        *round
	workerMangerService *worker.WorkerManager
	txPoolEnqueuer      TxPoolEnqueuer
	gameResultSettler   GameResultSettler
}

func NewGame(
	ctx context.Context,
	players []types.PlayerAddress,
	workerMangerService *worker.WorkerManager,
	txPoolEnqueuer TxPoolEnqueuer,
	gameResultSettler GameResultSettler,
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
		gameInfo: &dao.Game{
			Players:  daoPlayers,
			Type:     types.GameTypePVP,
			GameArgs: *gameArgs,
		},
		currentRound:        &round{round: nil, gamePlayers: gamePlayers},
		workerMangerService: workerMangerService,
		txPoolEnqueuer:      txPoolEnqueuer,
		gameResultSettler:   gameResultSettler,
	}
	game.setupNewRound()
	return game
}

func NewGameFromGameInfo(
	ctx context.Context,
	workerMangerService *worker.WorkerManager,
	gameInfo *dao.Game,
	txPoolEnqueuer TxPoolEnqueuer,
	gameResultSettler GameResultSettler) *Game {
	// Build runtime round and gamePlayers from fully loaded gameInfo
	currentRound := buildRuntimeState(gameInfo)

	g := &Game{
		ctx:                 ctx,
		gameInfo:            gameInfo,
		currentRound:        currentRound,
		workerMangerService: workerMangerService,
		txPoolEnqueuer:      txPoolEnqueuer,
		gameResultSettler:   gameResultSettler,
	}

	// Optional: expiry on recovery – skip and abort games that have clearly timed out
	if !gameInfo.CreatedAt.IsZero() {
		perRoundSeconds := gameInfo.ConfirmationTimeout +
			gameInfo.CommitmentSubmissionTimeout +
			gameInfo.CardSubmissionTimeout
		if perRoundSeconds > 0 && gameInfo.MaxRounds > 0 {
			expiryDuration := time.Duration(perRoundSeconds*gameInfo.MaxRounds) * time.Second
			if time.Since(gameInfo.CreatedAt) > expiryDuration {
				log.Infow("skipping expired game on recovery",
					"game id", gameInfo.ID,
					"status", gameInfo.Status,
					"created_at", gameInfo.CreatedAt,
					"expiry_duration", expiryDuration.Seconds())
				if gameInfo.Status == proto.GameStatus_GAME_INIT {
					if err := g.handleGameAbortInit(); err != nil {
						log.Errorf("expired game abort init failed, game: %d, err %s", gameInfo.ID, err)
					}
				} else {
					if err := g.handleGameAbortInternalError(); err != nil {
						log.Errorf("expired game abort internal failed, game: %d, err %s", gameInfo.ID, err)
					}
				}
				return nil
			}
		}
	}

	// Restore runtime state for current round and turn from DB
	if g.currentRound.round != nil {
		var currentTurn *dao.Turn
		if len(g.currentRound.round.Turns) > 0 {
			var maxTurnNum uint32
			for _, t := range g.currentRound.round.Turns {
				if t != nil && t.TurnNumber > maxTurnNum {
					maxTurnNum = t.TurnNumber
					currentTurn = t
				}
			}
			if currentTurn != nil && maxTurnNum > 0 {
				g.currentRound.turnNumber = maxTurnNum
			} else {
				g.currentRound.turnNumber = 1
			}
		} else {
			g.currentRound.turnNumber = 1
		}

		// Use persisted TurnStatus from DB when available; otherwise fall back to a safe default.
		r := g.currentRound.round
		switch {
		case r.IsLastRound || r.CompleteReason != proto.RoundCompleteReason_ROUND_COMPLETE_NORMAL:
			g.currentRound.setTurnStatus(proto.TurnStatus_TURN_ROUND_COMPLETED)
		case currentTurn != nil && currentTurn.TurnStatus != 0:
			g.currentRound.setTurnStatus(proto.TurnStatus(currentTurn.TurnStatus))
		default:
			g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION)
		}

		// Restore each player's currentTurnInfo from the current turn
		if currentTurn != nil {
			for _, pti := range currentTurn.PlayerTurnInfos {
				if pti == nil {
					continue
				}
				key := strings.ToLower(pti.TemporaryAddress)
				player, ok := g.currentRound.gamePlayers[key]
				if !ok {
					// fallback without lowercasing in case data was stored differently
					player, ok = g.currentRound.gamePlayers[pti.TemporaryAddress]
				}
				if !ok || player == nil {
					log.Errorf("recovery: player %s not found in gamePlayers for game %d", pti.TemporaryAddress, gameInfo.ID)
					continue
				}
				player.currentTurnInfo = pti
				// Optionally restore HP and multiplier from latest submitted card if available
				if pti.TurnSubmittedCard != nil && pti.TurnSubmittedCard.HealthAfter > 0 {
					player.currentHP = int64(pti.TurnSubmittedCard.HealthAfter)
					if pti.TurnSubmittedCard.MultiplierAfter > 0 {
						player.multiplier = pti.TurnSubmittedCard.MultiplierAfter
					}
				}
			}
		}
	}

	// For recovered games, potentially re-enqueue on-chain operations that were in-flight
	switch gameInfo.Status {
	case proto.GameStatus_GAME_INIT:
		// Game was matched but contract not yet created; re-push state to contract creation flow
		if err := g.pushStateToContractCreating(); err != nil {
			log.Errorf("recovered GAME_INIT pushStateToContractCreating failed, game: %d, err %s", gameInfo.ID, err)
		}
	case proto.GameStatus_GAME_RUNNING:
		if g.currentRound.getTurnStatus() == proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN {
			// Game was waiting for turn setup on-chain; re-send turn-ready
			if err := g.sendTurnReady(); err != nil {
				log.Errorf("recovered GAME_RUNNING sendTurnReady failed, game: %d, err %s", gameInfo.ID, err)
			}
		}
	}

	// Schedule timeout based on recovered (and possibly re-enqueued) state
	g.sendTimerEventByCurrentRound()

	return g
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
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN)
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
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN)
	return nil
}

// incrementTurnNumber increments the turn number for the current round
func (g *Game) incrementTurnNumber() {
	g.currentRound.turnNumber++
}

func (g *Game) stopGame() {
	log.Infow("stop game", "game id", g.gameInfo.ID)
	g.workerMangerService.CloseWorker(g.workerID())
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
	return nil
}
