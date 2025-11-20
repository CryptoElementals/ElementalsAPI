package game

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type gamePlayer struct {
	player          *dao.GamePlayerInfo
	playerTurnInfos []*dao.PlayerTurnInfo
	totalLostHP     int64
	currentHP       int64
}

func (p *gamePlayer) PlayerAddress() types.PlayerAddress {
	addr := types.PlayerAddress{}
	addr.FromDao(*p.player)
	return addr
}

func (p *gamePlayer) String() string {
	return fmt.Sprintf("%d_%s", p.player.PlayerId, p.player.TemporaryAddress)
}

// getSubmittedCards returns all submitted cards from playerTurnInfos as TurnSubmittedCard
func (p *gamePlayer) getSubmittedCards() []*dao.TurnSubmittedCard {
	cards := make([]*dao.TurnSubmittedCard, 0)
	for _, turnInfo := range p.playerTurnInfos {
		if turnInfo.TurnSubmittedCard != nil {
			cards = append(cards, turnInfo.TurnSubmittedCard)
		}
	}
	return cards
}

// getPlayerTurnInfoForTurn returns PlayerTurnInfo for a specific turn number
// Note: This assumes playerTurnInfos are ordered by turn number, which may not always be true
// A better approach would be to match by TurnID, but we need access to the Turn records
func (p *gamePlayer) getPlayerTurnInfoForTurn(turnNumber uint32) *dao.PlayerTurnInfo {
	// Try to find by index first (assuming ordered)
	if len(p.playerTurnInfos) >= int(turnNumber) {
		return p.playerTurnInfos[turnNumber-1]
	}
	// If not found by index, return the latest one (might be for current turn)
	if len(p.playerTurnInfos) > 0 {
		return p.playerTurnInfos[len(p.playerTurnInfos)-1]
	}
	return nil
}

// isPlayerReady checks if player is ready for current turn
func (p *gamePlayer) isPlayerReady() bool {
	// Check the latest turn info's status
	if len(p.playerTurnInfos) == 0 {
		return false
	}
	latest := p.playerTurnInfos[len(p.playerTurnInfos)-1]
	return latest.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_READY ||
		latest.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED ||
		latest.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED
}

// isSurrendered checks if player has surrendered
func (p *gamePlayer) isSurrendered() bool {
	// Check if any turn info indicates surrender
	for _, turnInfo := range p.playerTurnInfos {
		if turnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN {
			// This might indicate surrender, but we need to check the actual status
			// For now, we'll need to track this separately or check game state
		}
	}
	return false // TODO: Need to track surrender status properly
}

// getLostHP calculates lost HP from submitted cards
func (p *gamePlayer) getLostHP() int32 {
	lostHP := int32(0)
	for _, card := range p.getSubmittedCards() {
		if card.HealthBefore > card.HealthAfter {
			lostHP += int32(card.HealthBefore - card.HealthAfter)
		}
	}
	return lostHP
}

type Game struct {
	ctx                 context.Context
	gameInfo            *dao.Game
	currentRound        *Round
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
		currentRound:        &Round{round: nil, gamePlayers: gamePlayers, battleStates: nil},
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
		currentRound:        &Round{round: nil, gamePlayers: gamePlayers, battleStates: nil},
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
				g.currentRound.round = r
			}
		}
		// Initialize game state from Turns
		// Find the latest turn to determine current turn number
		if len(g.currentRound.round.Turns) > 0 {
			latestTurn := g.currentRound.round.Turns[len(g.currentRound.round.Turns)-1]
			g.currentRound.turnNumber = latestTurn.TurnNumber + 1
			if g.currentRound.turnNumber > 3 {
				g.currentRound.turnNumber = 1 // Round completed, will setup new round
			}
		} else {
			g.currentRound.turnNumber = 1
		}

		// Reconstruct playerTurnInfos from Turns for runtime use
		// Group PlayerTurnInfos by player
		playerTurnInfoMap := make(map[string][]*dao.PlayerTurnInfo)
		for _, turn := range g.currentRound.round.Turns {
			for _, playerTurnInfo := range turn.PlayerTurnInfos {
				key := playerTurnInfo.TemporaryAddress
				playerTurnInfoMap[key] = append(playerTurnInfoMap[key], playerTurnInfo)
			}
		}

		// Assign reconstructed playerTurnInfos to gamePlayers
		for key, turnInfos := range playerTurnInfoMap {
			player, err := g.getGamePlayer(key)
			if err != nil {
				// should never happen
				log.Fatalf("getGamePlayer failed, err: %v", err)
			}
			player.playerTurnInfos = turnInfos
			// Calculate current HP and lost HP from submitted cards
			submittedCards := player.getSubmittedCards()
			if len(submittedCards) != 0 {
				player.currentHP = currentHpFromCards(submittedCards)
			}
			player.totalLostHP = int64(player.getLostHP())
		}

		// Determine round status from CompleteReason
		if shouldTerminate {
			terminateGame()
			return nil
		}
		// Check if round is completed (CompleteReason is set or IsLastRound is true)
		if g.currentRound.round.IsLastRound || g.currentRound.round.CompleteReason != proto.RoundCompleteReason_ROUND_COMPLETE_NORMAL {
			g.currentRound.turnStatus = proto.TurnStatus_TURN_ROUND_COMPLETED
			g.setupNewRound()
		} else {
			// Determine status from turns
			if len(g.currentRound.round.Turns) == 0 {
				g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION
			} else {
				// Default to waiting commitments if turns exist
				g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_COMMITMENTS
			}
			g.sendTimerEventByCurrentRound()
		}
	} else {
		g.setupNewRound()
	}

	return g
}

func (g *Game) saveGame() error {
	err := db.SaveGame(g.gameInfo)
	if err != nil {
		log.Errorw("saveGame failed", "err", err, "game id", g.gameInfo.ID)
	}
	return nil
}

// getOrUpdatePlayerTurnInfo gets or creates PlayerTurnInfo for current turn
func (g *Game) getOrUpdatePlayerTurnInfo(player *gamePlayer) *dao.PlayerTurnInfo {
	currentTurn := g.getOrCreateCurrentTurn()
	// Check if PlayerTurnInfo already exists in current turn
	for _, pti := range currentTurn.PlayerTurnInfos {
		if pti.PlayerID == player.player.PlayerId && pti.TemporaryAddress == player.player.TemporaryAddress {
			// Found matching PlayerTurnInfo in current turn
			// Update player's playerTurnInfos if not already there
			found := false
			for _, existing := range player.playerTurnInfos {
				if existing.ID == pti.ID {
					found = true
					return existing
				}
			}
			if !found {
				player.playerTurnInfos = append(player.playerTurnInfos, pti)
				return pti
			}
			return pti
		}
	}
	// Create new PlayerTurnInfo for current turn
	newPlayerTurnInfo := &dao.PlayerTurnInfo{
		TurnID:           currentTurn.ID,
		PlayerID:         player.player.PlayerId,
		TemporaryAddress: player.player.TemporaryAddress,
		PlayerStatus:     proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN,
	}
	player.playerTurnInfos = append(player.playerTurnInfos, newPlayerTurnInfo)
	currentTurn.PlayerTurnInfos = append(currentTurn.PlayerTurnInfos, newPlayerTurnInfo)
	return newPlayerTurnInfo
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
	allPlayers := make([]types.PlayerAddress, 0, len(g.currentRound.gamePlayers))
	for _, player := range g.currentRound.gamePlayers {
		allPlayers = append(allPlayers, player.PlayerAddress())
		// Mark player as ready for current turn
		playerTurnInfo := g.getOrUpdatePlayerTurnInfo(player)
		playerTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_READY
	}
	if err := g.sendContractCreation(allPlayers); err != nil {
		g.handleGameAbortInternalError()
		return err
	}
	g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN
	return nil
}

// getCurrentTurnNumber returns the current turn number (1-3) from the Round struct
func (g *Game) getCurrentTurnNumber() uint32 {
	if g.currentRound == nil || g.currentRound.turnNumber == 0 {
		return 1
	}
	return g.currentRound.turnNumber
}

// setupNewTurn sends event to chain manager to setup a new turn
// Note: For the first turn of the first round, this is not needed as the contract creation handles it
func (g *Game) setupNewTurn() error {
	// Skip for the first turn of the first round
	if g.currentRound.round.RoundNumber == 1 && g.getCurrentTurnNumber() == 1 {
		return nil
	}
	// RoomContract check removed - always uses RoomV2 contract address
	turnNumber := g.getCurrentTurnNumber()
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

// getOrCreateCurrentTurn gets or creates the current Turn record
func (g *Game) getOrCreateCurrentTurn() *dao.Turn {
	turnNumber := g.getCurrentTurnNumber()
	// Check if turn already exists
	for _, turn := range g.currentRound.round.Turns {
		if turn.TurnNumber == turnNumber {
			return turn
		}
	}
	// Create new turn
	newTurn := &dao.Turn{
		RoundID:         g.currentRound.round.ID, // Will be set when round is saved
		TurnNumber:      turnNumber,
		PlayerTurnInfos: make([]*dao.PlayerTurnInfo, 0),
	}
	return newTurn
}

// getCurrentTurn gets the current Turn record, returns nil if not found
func (g *Game) getCurrentTurn() *dao.Turn {
	turnNumber := g.getCurrentTurnNumber()
	for _, turn := range g.currentRound.round.Turns {
		if turn.TurnNumber == turnNumber {
			return turn
		}
	}
	return nil
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
		Turns:       make([]*dao.Turn, 0),
	}
	// Initialize playerTurnInfos for each player (empty at start of round)
	for _, player := range g.currentRound.gamePlayers {
		player.playerTurnInfos = make([]*dao.PlayerTurnInfo, 0)
	}
	g.currentRound.round = newRound // Update the embedded Round's reference
	g.currentRound.turnNumber = 1   // Start with turn 1 for each new round
	g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION
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

func (g *Game) setGamePlayer(tempAddr string, player *gamePlayer) {
	g.currentRound.gamePlayers[strings.ToLower(tempAddr)] = player
}

func (g *Game) getGamePlayer(tempAddr string) (*gamePlayer, error) {
	player, ok := g.currentRound.gamePlayers[strings.ToLower(tempAddr)]
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
		GameID:      g.gameInfo.ID,
		RoundNumber: uint32(g.currentRound.round.RoundNumber),
		TurnNumber:  g.getCurrentTurnNumber(),
	})
}

func (g *Game) abortedGameResult() *dao.GameResult {
	gameRes := &dao.GameResult{
		GameResultType: proto.GameResultType_GAME_ABORTED,
		BattleReward: &dao.BattleReward{
			PlayerRewards: []*dao.PlayerReward{},
		},
	}
	for _, player := range g.currentRound.gamePlayers {
		playerReward := &dao.PlayerReward{
			PlayerId:         player.player.PlayerId,
			TemporaryAddress: player.player.TemporaryAddress,
		}
		gameRes.BattleReward.PlayerRewards = append(gameRes.BattleReward.PlayerRewards, playerReward)
	}
	return gameRes
}

func currentHpFromCards(cards []*dao.TurnSubmittedCard) int64 {
	if len(cards) == 0 {
		return 0
	}
	// Get the last card (assuming they're ordered by turn)
	lastCard := cards[len(cards)-1]
	return int64(lastCard.HealthAfter)
}
