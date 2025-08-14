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
	"github.com/CryptoElementals/common/room_server/battle"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type timerEvent struct {
	currentGameStatus  proto.GameStatus
	currentRound       uint32
	currentRoundStatus proto.RoundStatus
}

type gamePlayer struct {
	player      *dao.GamePlayerInfo
	roundPlayer *dao.PlayerRoundInfo
	totalLostHP int64
	currentHP   int64
	addr        *types.PlayerAddress
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
	gamePlayers         map[string]*gamePlayer
	currentRound        *dao.Round
	workerMangerService *worker.WorkerManager
	chainSvc            ContractClient
	gameContextHandler  GameHandler
}

func NewGame(
	ctx context.Context,
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
			addr:      &player,
		}
	}
	game := &Game{
		ctx: ctx,
		gameInfo: &dao.Game{
			Players:  daoPlayers,
			Type:     types.GameTypePVP,
			GameArgs: *gameArgs,
		},
		gamePlayers:         gamePlayers,
		workerMangerService: workerMangerService,
		chainSvc:            chainSvc,
		gameContextHandler:  gameContinuer,
	}
	game.setupNewRound()
	return game
}

func NewGameFromGameInfo(
	ctx context.Context,
	workerMangerService *worker.WorkerManager,
	gameContinuer GameHandler,
	gameInfo *dao.Game,
	chainSvc ContractClient) *Game {
	g := &Game{
		ctx:                 ctx,
		gameInfo:            gameInfo,
		gamePlayers:         make(map[string]*gamePlayer),
		workerMangerService: workerMangerService,
		chainSvc:            chainSvc,
		gameContextHandler:  gameContinuer,
	}

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
		addrKey := types.NewPlayerAddress(playerInfo.WalletAddress, playerInfo.TemporaryAddress)
		g.setGamePlayer(playerInfo.TemporaryAddress, &gamePlayer{
			player:    playerInfo,
			currentHP: g.gameInfo.InitialHP,
			addr:      addrKey,
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
				g.currentRound = r
			}
		}
		for _, roundPlayer := range g.currentRound.PlayerRoundInfos {
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
		if g.currentRound.Status == proto.RoundStatus_ROUND_COMPLETED {
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
	}
	for _, round := range g.gameInfo.Rounds {
		if round.RoundNumber == (roundNum) {
			return conversion.DbRoundToRoundResult(round), gameRes
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
	return conversion.DbGameToProtoGamePhase(g.gameInfo, g.currentRound)
}

func (g *Game) saveGame() error {
	err := db.SaveGame(g.gameInfo)
	if err != nil {
		log.Errorf("SaveGame failed, err: %v", err)
		return err
	}
	return nil
}

func (g *Game) savePlayerRoundInfo(roundPlayer *dao.PlayerRoundInfo) error {
	err := db.SavePlayerRoundInfo(roundPlayer)
	if err != nil {
		return err
	}
	return nil
}

func (g *Game) saveRound(round *dao.Round) error {
	err := db.SaveRound(round)
	if err != nil {
		return err
	}
	return nil
}

func (g *Game) Handle(ctx context.Context, event *types.Event) error {
	g.lock.Lock()
	defer g.lock.Unlock()
	if timerEvt, err := types.AssertInterface[*timerEvent](event); err == nil {
		g.handleTimerEvent(timerEvt)
		return nil
	}
	switch g.gameInfo.Status {
	case proto.GameStatus_GAME_INIT, proto.GameStatus_GAME_RUNNING:
		err := g.handleRound(event)
		if err != nil {
			log.Errorf("handleRound failed, err: %v", err)
			return err
		}
		return nil
	case proto.GameStatus_GAME_END:
		return errors.New("game has ended")
	}
	return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
}

func (g *Game) handleRound(event *types.Event) error {
	currentRound := g.currentRound
	if surrentEvt, err := types.AssertInterface[*types.SurrenderEvent](event); err == nil {
		return g.handleSurrenderEvent(surrentEvt)
	}

	switch currentRound.Status {
	case proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION:
		return g.handleWaittingRoundPlayersConfirmed(event)
	case proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN:
		return g.handleGameStateWaittingSetupOnChain(event)
	case proto.RoundStatus_ROUND_WAITTING_COMMITMENTS:
		return g.handleGameStateWaittingCommitments(event)
	case proto.RoundStatus_ROUND_WAITTING_CARDS:
		return g.handleGameStateCardSubmitted(event)
	}
	return nil
}

func (g *Game) pushStateToContractCreating() error {
	g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
	allPlayers := make([]types.PlayerAddress, 0, len(g.gamePlayers))
	for _, player := range g.gamePlayers {
		allPlayers = append(allPlayers, player.PlayerAddress())
		player.roundPlayer.PlayerReady = true
	}
	err := g.sendContractCreation(allPlayers)
	if err != nil {
		return err
	}
	g.currentRound.Status = proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN
	return nil
}

func (g *Game) handleSurrenderEvent(event *types.SurrenderEvent) error {
	p, err := g.getGamePlayer(event.Address.TemporaryAddress)
	if err != nil {
		return err
	}
	p.roundPlayer.Surrendered = true
	err = g.savePlayerRoundInfo(p.roundPlayer)
	if err != nil {
		return err
	}
	return g.handleRoundEnd(proto.RoundCompleteReason_ROUND_COMPLETE_PLAYER_SURRENDER)
}

func (g *Game) handleWaittingRoundPlayersConfirmed(event *types.Event) error {
	evt, err := types.AssertInterface[*types.PlayerReadyEvent](event)
	if err != nil {
		return err
	}
	// stale events
	if evt.RoundNumber != g.currentRound.RoundNumber {
		return nil
	}
	player, err := g.getGamePlayer(evt.PlayerAddress.TemporaryAddress)
	if err != nil {
		log.Errorf("getGamePlayer failed, err: %v", err)
		return err
	}
	player.roundPlayer.PlayerReady = true
	// check if all players ready
	allPlayersReady := true
	for _, player := range g.gamePlayers {
		if !player.roundPlayer.PlayerReady {
			allPlayersReady = false
		}
	}
	g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), &types.RoundPartialReadyEvent{
		GameID:       g.gameInfo.ID,
		RoundNumber:  uint32(g.currentRound.RoundNumber),
		ReadyAddress: player.PlayerAddress(),
	}))
	// for the first round we don't record any thing until both players confirmed
	if !allPlayersReady {
		return g.savePlayerRoundInfo(player.roundPlayer)
	}
	allPlayers := make([]types.PlayerAddress, 0, len(g.gamePlayers))
	for _, player := range g.gamePlayers {
		allPlayers = append(allPlayers, player.PlayerAddress())
	}
	// the first round, we need to create contract
	if g.currentRound.RoundNumber == 1 {
		g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
		err := g.sendContractCreation(allPlayers)
		if err != nil {
			return err
		}
	} else {
		if g.gameInfo.RoomContract == "" {
			return errors.New("room contract empty, need RequireContractCreationEvent but got RequireSetupNewRoundEvent")
		}
		// otherwise we need to setup new round on chain
		err := g.sendRoundReady()
		if err != nil {
			return err
		}
	}
	g.currentRound.Status = proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN
	err = g.saveGame()
	if err != nil {
		return err
	}
	g.sendTimerEventByCurrentRound()
	return nil
}

func (g *Game) handleGameStateWaittingSetupOnChain(event *types.Event) error {
	defer g.sendTimerEventByCurrentRound()
	if g.currentRound.RoundNumber == 1 {
		return g.handleRoomContractCreated(event)
	}
	return g.handleNewRoundSetupOnChain(event)
}

func (g *Game) handleRoomContractCreated(event *types.Event) error {
	evt, err := types.AssertInterface[*types.RoomContractCreated](event)
	if err != nil {
		return err
	}
	g.gameInfo.RoomContract = evt.RoomContractAddress
	// just skip setup new round on chain for first round
	g.currentRound.Status = proto.RoundStatus_ROUND_WAITTING_COMMITMENTS
	g.currentRound.SetupOnChainAt = evt.TimeStamp
	err = g.saveGame()
	if err != nil {
		return err
	}
	gameReadyEvt := types.NewEvent(g.workerID(), &types.GameReadyEvent{
		GameID:          g.gameInfo.ID,
		ContractAddress: evt.RoomContractAddress,
	})
	roundReadyEvt := types.NewEvent(g.workerID(), &types.RoundReadyEvent{
		GameID:         g.gameInfo.ID,
		RoundNumber:    g.currentRound.RoundNumber,
		RoundStartedAt: evt.TimeStamp,
		RoundTimeout:   g.gameInfo.RoundTimeout,
	})
	g.sendEventsToAllPlayers(gameReadyEvt, roundReadyEvt)
	return nil
}

func (g *Game) handleNewRoundSetupOnChain(event *types.Event) error {
	evt, err := types.AssertInterface[*types.NewRoundSetupComplete](event)
	if err != nil {
		return err
	}
	if evt.GameID != g.gameInfo.ID {
		return errors.New("invalid game id")
	}
	// stale event
	if evt.RoundNumber != uint32(g.currentRound.RoundNumber) {
		return nil
	}
	g.currentRound.Status = proto.RoundStatus_ROUND_WAITTING_COMMITMENTS
	g.currentRound.SetupOnChainAt = evt.TimeStamp
	err = g.saveRound(g.currentRound)
	if err != nil {
		return err
	}
	g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), &types.RoundReadyEvent{
		GameID:         g.gameInfo.ID,
		RoundNumber:    evt.RoundNumber,
		RoundStartedAt: evt.TimeStamp,
	}))
	return nil
}

func (g *Game) handleGameStateWaittingCommitments(event *types.Event) error {
	evt, err := types.AssertInterface[*types.PlayerCommitmentOnChain](event)
	if err != nil {
		return err
	}
	// stale events
	if evt.RoundNumber != g.currentRound.RoundNumber {
		return nil
	}
	player, err := g.getGamePlayer(evt.Address.TemporaryAddress)
	if err != nil {
		return err
	}
	player.roundPlayer.SubmittedCommitment = evt.Commitment
	// check if all player commitment on chain
	allCommitmentsOnChain := true
	for _, player := range g.gamePlayers {
		if len(player.roundPlayer.SubmittedCommitment) == 0 {
			allCommitmentsOnChain = false
			break
		}
	}
	if !allCommitmentsOnChain {
		return g.savePlayerRoundInfo(player.roundPlayer)
	}
	g.currentRound.Status = proto.RoundStatus_ROUND_WAITTING_CARDS
	err = g.saveRound(g.currentRound)
	if err != nil {
		return err
	}
	// all player commitment on chain, send EVENT_TYPE_COMMITMENTS_ON_CHAIN to players
	commitmentsOnChainEvt := types.NewEvent(g.workerID(), &types.CommitmentsOnChainEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: evt.RoundNumber,
	})
	g.sendEventsToAllPlayers(commitmentsOnChainEvt)
	g.sendTimerEventByCurrentRound()
	return nil
}

func (g *Game) handleGameStateCardSubmitted(event *types.Event) error {
	// set player cards and player status
	evt, err := types.AssertInterface[*types.PlayerCardsOnChain](event)
	if err != nil {
		return err
	}
	// stale events
	if evt.RoundNumber != g.currentRound.RoundNumber {
		return nil
	}
	player, err := g.getGamePlayer(evt.Address.TemporaryAddress)
	if err != nil {
		return err
	}
	for i, card := range evt.Cards {
		player.roundPlayer.SubmittedCards = append(player.roundPlayer.SubmittedCards, &dao.RoundSubmittedCard{
			CardID:     card,
			CardNumber: uint32(i + 1),
		})
	}
	// check if all player cards on chain
	allCardsOnChain := true
	for _, player := range g.gamePlayers {
		if len(player.roundPlayer.SubmittedCards) == 0 {
			allCardsOnChain = false
			break
		}
	}
	err = g.savePlayerRoundInfo(player.roundPlayer)
	if err != nil {
		return err
	}
	if !allCardsOnChain {
		return nil
	}
	g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), &types.CardsOnChainEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: evt.RoundNumber,
	}))
	return g.handleRoundEnd(proto.RoundCompleteReason_ROUND_COMPLETE_NORMAL)
}

// can go into game end from any other status
func (g *Game) handleGameEnd() error {
	completeEvt := &types.GameCompletedEvent{
		GameID:   g.gameInfo.ID,
		GameInfo: g.gameInfo,
	}
	gameCompletedEvt := types.NewEvent(g.workerID(), completeEvt)
	g.currentRound.Status = proto.RoundStatus_ROUND_COMPLETED
	g.currentRound.IsLastRound = true
	g.gameInfo.Status = proto.GameStatus_GAME_END
	err := g.saveGame()
	if err != nil {
		return err
	}
	if err := g.gameContextHandler.HandleGameCompletedEvent(completeEvt); err != nil {
		return err
	}
	g.sendEventsToAllPlayers(gameCompletedEvt)
	g.stopWorker()
	return nil
}

// can go into game end from any other status
func (g *Game) handleGameAbortInit() error {
	log.Infow("game aborted", "game id", g.gameInfo.ID)
	if g.gameInfo.Status != proto.GameStatus_GAME_INIT {
		return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
	}
	completeEvt := &types.GameCompletedEvent{
		GameID:   g.gameInfo.ID,
		GameInfo: g.gameInfo,
	}
	gameCompletedEvt := types.NewEvent(g.workerID(), completeEvt)
	// we need a deterministic sequence for locks
	if err := g.gameContextHandler.HandleGameCompletedEvent(completeEvt); err != nil {
		return err
	}
	// we need a deterministic sequence for locks
	g.sendEventsToAllPlayers(gameCompletedEvt)
	g.stopWorker()
	return nil
}

func (g *Game) stopWorker() {
	g.workerMangerService.CloseWorker(g.workerID())
}

func (g *Game) createSelf() {
	g.workerMangerService.SpwanWorker(g.ctx, g.workerID(), types.WORKER_TYPE_GAME, g)
}

func (g *Game) workerID() string {
	return fmt.Sprint(g.gameInfo.ID)
}

func (g *Game) setupNewRound() {
	roundNum := uint32(1)
	if g.currentRound != nil {
		roundNum = g.currentRound.RoundNumber + 1
	}
	newRound := &dao.Round{
		GameID:      g.gameInfo.ID,
		RoundNumber: roundNum,
		Status:      proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION,
	}
	for _, player := range g.gamePlayers {
		playerRoundInfo := &dao.PlayerRoundInfo{
			WalletAddress:    player.player.WalletAddress,
			TemporaryAddress: player.player.TemporaryAddress,
			SubmittedCards:   make([]*dao.RoundSubmittedCard, 0),
		}
		newRound.PlayerRoundInfos = append(newRound.PlayerRoundInfos, playerRoundInfo)
		player.roundPlayer = playerRoundInfo
	}
	g.currentRound = newRound
	g.gameInfo.Rounds = append(g.gameInfo.Rounds, newRound)
	g.sendTimerEventByCurrentRound()
}

func (g *Game) sendEventsToAllPlayers(events ...*types.Event) {
	for _, player := range g.gamePlayers {
		for _, event := range events {
			g.workerMangerService.SendEvent(player.String(), event)
		}
	}
}

func (g *Game) handleRoundEnd(reason proto.RoundCompleteReason) error {
	g.currentRound.CompleteReason = reason
	g.currentRound.RoundEndTime = time.Now().Unix()
	e := battle.NewBattleEngine()
	input := conversion.DbRoundToProtoRoundInput(g.currentRound)
	for _, p := range input.Players {
		player, err := g.getGamePlayer(p.TemporaryAddress)
		if err != nil {
			return err
		}
		p.LostHP = int32(player.totalLostHP)
		p.HP = int32(player.currentHP)
	}
	roundResult, gameResult, err := e.ExecuteRoundProto(input)
	if err != nil {
		log.Errorf("ExecuteRoundProto failed, err: %v", err)
		return err
	}
	g.applyRoundResultToCurrentRound(roundResult)
	if roundResult.IsGameOver {
		g.gameInfo.GameResult = conversion.ProtoGameResultToDbGameResult(gameResult)
		return g.handleGameEnd()
	}
	g.currentRound.Status = proto.RoundStatus_ROUND_COMPLETED
	roundCompletedEvt := types.NewEvent(g.workerID(), &types.RoundCompletedEvent{
		GameID:    g.gameInfo.ID,
		RoundInfo: g.currentRound,
	})
	g.setupNewRound()
	err = g.saveGame()
	if err != nil {
		return err
	}
	g.sendEventsToAllPlayers(roundCompletedEvt)
	return nil
}

func (g *Game) applyRoundResultToCurrentRound(roundResult *proto.RoundResult) {
	for _, p := range roundResult.Players {
		player, err := g.getGamePlayer(p.TemporaryAddress)
		if err != nil {
			// should never happen
			log.Fatalf("getGamePlayer failed, err: %v", err)
			continue
		}
		player.roundPlayer.LostHP = p.LostHP
		player.totalLostHP = int64(p.LostHP)
		for i, card := range p.CardStats {
			for _, sc := range player.roundPlayer.SubmittedCards {
				if sc.CardNumber == uint32(card.CardNumber) {
					sc := player.roundPlayer.SubmittedCards[i]
					sc.HealthBefore = uint32(card.HPBefore)
					sc.HealthAfter = uint32(card.HPAfter)
					sc.MultiplierBefore = uint32(card.MultiplierBefore)
					sc.MultiplierAfter = uint32(card.MultiplierAfter)
					sc.Description = card.Description
					sc.ElementRelation = card.ElementRelation
					sc.CardEffects = conversion.ProtoBattleEffectsToDbCardEffects(card.Effects)
				}
			}
		}
		if len(player.roundPlayer.SubmittedCards) != 0 {
			player.currentHP = currentHpFromCards(player.roundPlayer.SubmittedCards)
		}
	}
}

func (g *Game) timeoutFromCurentRound() time.Duration {
	if g.gameInfo.Status == proto.GameStatus_GAME_END {
		return 0
	}
	timeoutDuration := int64(0)
	switch g.gameInfo.Status {
	case proto.GameStatus_GAME_INIT:
		// game waitting confirmed for the first round
		timeoutDuration = g.gameInfo.GameArgs.GameMatchTimeout + g.gameInfo.GameArgs.GameMatchTimeoutRedundancy
	case proto.GameStatus_GAME_RUNNING:
		switch g.currentRound.Status {
		case proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION,
			proto.RoundStatus_ROUND_COMPLETED,
			proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN:
			// waitting for confimation
			timeoutDuration = g.gameInfo.GameArgs.RoundConfirmTimeout + g.gameInfo.GameArgs.RoundConfirmTimeoutRedundancy
		case proto.RoundStatus_ROUND_WAITTING_COMMITMENTS, proto.RoundStatus_ROUND_WAITTING_CARDS:
			// round submitting cards
			timeoutDuration = g.gameInfo.GameArgs.RoundTimeout + g.gameInfo.GameArgs.RoundTimeoutRedundancy
		}
	case proto.GameStatus_GAME_END:
		return 0
	}

	timeout := time.Second * time.Duration(timeoutDuration)
	if g.currentRound.SetupOnChainAt != 0 {
		timeout -= time.Since(time.Unix(g.currentRound.SetupOnChainAt, 0))
	}
	return timeout
}

func (g *Game) sendTimerEventByCurrentRound() {
	timeout := g.timeoutFromCurentRound()
	if timeout == 0 {
		return
	}
	timerEvent := &timerEvent{
		currentGameStatus:  g.gameInfo.Status,
		currentRound:       g.currentRound.RoundNumber,
		currentRoundStatus: g.currentRound.Status,
	}
	log.Debugw("send timer event",
		"game id", g.gameInfo.ID,
		"round", timerEvent.currentRound,
		"round status", timerEvent.currentRoundStatus,
		"timeout", timeout.Seconds(),
	)
	time.AfterFunc(timeout, func() {
		g.workerMangerService.SendEvent(g.workerID(), types.NewEvent(g.workerID(), timerEvent))
	})
}

func (g *Game) handleTimerEvent(event *timerEvent) {

	if g.gameInfo.Status == proto.GameStatus_GAME_END {
		return
	}
	// stale event
	if g.currentRound.RoundNumber != event.currentRound {
		return
	}
	// status changed go ahead
	if g.currentRound.Status != event.currentRoundStatus {
		return
	}
	log.Infow("timer event triggered",
		"game id", g.gameInfo.ID,
		"round", g.currentRound.RoundNumber,
		"round status", g.currentRound.Status,
		"game status", g.gameInfo.Status)
	// game init only exists at the very beginning, once both players confirms, it turns to game running
	if g.gameInfo.Status == proto.GameStatus_GAME_INIT {
		err := g.handleGameAbortInit()
		if err != nil {
			log.Errorf("abort game failed, err: %s", err.Error())
		}
		return
	}
	switch g.currentRound.Status {
	case proto.RoundStatus_ROUND_COMPLETED:
		// do nothing
	case proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN:
		log.Errorf("setup on chain timeout, current round: %d, gameid: %d", event.currentRound, g.gameInfo.ID)
		g.handleRoundEnd(proto.RoundCompleteReason_ROUND_COMPLETE_SERVER_CHAIN_TIMEOUT)
	case proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION:
		g.handleRoundEnd(proto.RoundCompleteReason_ROUND_COMPLETE_PLAYER_CONFIRMATION_TIMEOUT)
	case proto.RoundStatus_ROUND_WAITTING_COMMITMENTS:
		g.handleRoundEnd(proto.RoundCompleteReason_ROUND_COMPLETE_PLAYER_COMMITMENTS_TIMEOUT)
	case proto.RoundStatus_ROUND_WAITTING_CARDS:
		g.handleRoundEnd(proto.RoundCompleteReason_ROUND_COMPLETE_PLAYER_CARDS_TIMEOUT)
	}
}

func (g *Game) setGamePlayer(tempAddr string, player *gamePlayer) {
	g.gamePlayers[strings.ToLower(tempAddr)] = player
}

func (g *Game) getGamePlayer(tempAddr string) (*gamePlayer, error) {
	player, ok := g.gamePlayers[strings.ToLower(tempAddr)]
	if !ok {
		return nil, fmt.Errorf("player %s not found", tempAddr)
	}
	return player, nil
}

func (g *Game) sendContractCreation(allPlayers []types.PlayerAddress) error {
	err := g.chainSvc.CreateRoomContract(&types.RequireContractCreationEvent{
		GameID:         g.gameInfo.ID,
		Players:        allPlayers,
		InitialHP:      g.gameInfo.InitialHP,
		RoundTimeout:   g.gameInfo.RoundTimeout,
		MaxRoundNumber: g.gameInfo.MaxRounds,
	})
	if err != nil {
		return err
	}
	return nil
}

func (g *Game) sendRoundReady() error {
	err := g.chainSvc.SetRoundReady(&types.RequireSetupNewRoundEvent{
		GameID:          g.gameInfo.ID,
		ContractAddress: g.gameInfo.RoomContract,
		RoundNumber:     uint32(g.currentRound.RoundNumber),
	})
	if err != nil {
		return err
	}
	return nil
}

func currentHpFromCards(cards []*dao.RoundSubmittedCard) int64 {
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].CardNumber < cards[j].CardNumber
	})
	lastCard := cards[len(cards)-1]
	return int64(lastCard.HealthAfter)
}
