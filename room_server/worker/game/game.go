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
	// Battle state fields (used during battle execution)
	multiplier uint32       // Calculated from totalLostHP
	status     playerStatus // Runtime player status during battle
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
	// Count non-nil cards first to pre-allocate
	count := 0
	for _, turnInfo := range p.playerTurnInfos {
		if turnInfo.TurnSubmittedCard != nil {
			count++
		}
	}
	if count == 0 {
		return nil
	}
	cards := make([]*dao.TurnSubmittedCard, 0, count)
	for _, turnInfo := range p.playerTurnInfos {
		if turnInfo.TurnSubmittedCard != nil {
			cards = append(cards, turnInfo.TurnSubmittedCard)
		}
	}
	return cards
}

func (g *gamePlayer) getLastSubmittedCard() *dao.TurnSubmittedCard {
	lastSubmittedCard := g.playerTurnInfos[len(g.playerTurnInfos)-1].TurnSubmittedCard
	if lastSubmittedCard == nil {
		return nil
	}
	return lastSubmittedCard
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
	currentRound        *round
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
			player:     daoPlayer,
			currentHP:  gameArgs.InitialHP,
			multiplier: 1,
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
		currentRound:        &round{round: nil, gamePlayers: gamePlayers},
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
			// Check if game is over
			isGameComplete := gameInfo.GameResult != nil
			err := g.completeRoundAndCheckGameEnd(proto.RoundCompleteReason_ROUND_COMPLETE_SERVER_INTERNAL_TIMEOUT, isGameComplete)
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
		// Since Turns are stored by index (turn 1 at index 0, turn 2 at index 1, turn 3 at index 2),
		// we can find the latest turn by checking the highest index with a non-nil entry
		latestTurnNumber := uint32(0)
		for i := len(g.currentRound.round.Turns) - 1; i >= 0; i-- {
			if g.currentRound.round.Turns[i] != nil {
				latestTurnNumber = g.currentRound.round.Turns[i].TurnNumber
				break
			}
		}
		if latestTurnNumber > 0 {
			g.currentRound.turnNumber = latestTurnNumber + 1
			if g.currentRound.turnNumber > 3 {
				g.currentRound.turnNumber = 1 // Round completed, will setup new round
			}
		} else {
			g.currentRound.turnNumber = 1
		}

		// Reconstruct playerTurnInfos from Turns for runtime use
		// Group PlayerTurnInfos by player, maintaining sorted order by turn number
		// Since Turns are stored by index (turn 1 at index 0, turn 2 at index 1, turn 3 at index 2),
		// we use turn.TurnNumber - 1 as the index to maintain sorted order in playerTurnInfos
		playerTurnInfoMap := make(map[string][]*dao.PlayerTurnInfo)
		for _, turn := range g.currentRound.round.Turns {
			if turn == nil {
				continue
			}
			idx := int(turn.TurnNumber) - 1
			for _, playerTurnInfo := range turn.PlayerTurnInfos {
				key := playerTurnInfo.TemporaryAddress
				// Initialize slice if needed
				if playerTurnInfoMap[key] == nil {
					playerTurnInfoMap[key] = make([]*dao.PlayerTurnInfo, 0, 3)
				}
				// Ensure slice is large enough and store at correct index to maintain sorted order
				for len(playerTurnInfoMap[key]) <= idx {
					playerTurnInfoMap[key] = append(playerTurnInfoMap[key], nil)
				}
				playerTurnInfoMap[key][idx] = playerTurnInfo
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
		return err
	}
	return nil
}

// getPlayerTurnInfo gets PlayerTurnInfo for the current turn, returns nil if not found
// Uses index-based access: turnNumber 1 -> index 0, turnNumber 2 -> index 1, turnNumber 3 -> index 2
// Assumes playerTurnInfos slice is sorted by turn number
func (g *Game) getPlayerTurnInfo(player *gamePlayer) *dao.PlayerTurnInfo {
	turnNumber := g.currentRound.getCurrentTurnNumber()
	pti := player.playerTurnInfos[int(turnNumber)-1]
	return pti
}

// createPlayerTurnInfo creates a new PlayerTurnInfo for the current turn
// Ensures it's stored at the correct index to maintain sorted order by turn number
func (g *Game) createPlayerTurnInfo(player *gamePlayer) *dao.PlayerTurnInfo {
	currentTurn := g.currentRound.getCurrentTurn()
	// Create new PlayerTurnInfo for current turn
	newPlayerTurnInfo := &dao.PlayerTurnInfo{
		TurnID:           currentTurn.ID,
		PlayerID:         player.player.PlayerId,
		TemporaryAddress: player.player.TemporaryAddress,
		PlayerStatus:     proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN,
	}

	player.playerTurnInfos = append(player.playerTurnInfos, newPlayerTurnInfo)
	currentTurn.PlayerTurnInfos = player.playerTurnInfos
	return newPlayerTurnInfo
}

// getOrUpdatePlayerTurnInfo gets or creates PlayerTurnInfo for current turn
func (g *Game) getOrUpdatePlayerTurnInfo(player *gamePlayer) *dao.PlayerTurnInfo {
	// Try to get existing PlayerTurnInfo first
	if pti := g.getPlayerTurnInfo(player); pti != nil {
		return pti
	}
	// Create new PlayerTurnInfo if it doesn't exist
	return g.createPlayerTurnInfo(player)
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

// setupNewTurn sends event to chain manager to setup a new turn
// Note: For the first turn of the first round, this is not needed as the contract creation handles it
func (g *Game) setupNewTurn() error {
	// Skip for the first turn of the first round
	if g.currentRound.round.RoundNumber == 1 && g.currentRound.getCurrentTurnNumber() == 1 {
		return nil
	}
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
	// Initialize playerTurnInfos for each player (empty at start of round)
	for _, player := range g.currentRound.gamePlayers {
		player.playerTurnInfos = make([]*dao.PlayerTurnInfo, 0)
	}
	g.currentRound.round = newRound // Update the embedded Round's reference
	g.currentRound.turnNumber = 1   // Start with turn 1 for each new round
	g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION

	// Create turn 1 record immediately when new round is set up
	// This will also initialize playerTurnInfos for all players
	turn1 := g.currentRound.createNewTurn()
	turn1.RoundID = newRound.ID

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
	return g.chainSvc.CreateRoomContract(&types.RequireGameCreationEvent{
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
		TurnNumber:  g.currentRound.getCurrentTurnNumber(),
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
