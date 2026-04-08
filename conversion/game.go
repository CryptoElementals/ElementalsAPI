package conversion

import (
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

func DbGameInfoToProtoGameInfo(info *dao.Game) *proto.GameInfo {
	if info == nil {
		return nil
	}
	gameInfo := &proto.GameInfo{
		GameId:    info.ID,
		GameType:  proto.GameType(info.Type),
		Status:    proto.GameStatus(info.Status),
		InitialHp: int32(info.GameArgs.InitialHP),
	}
	// convert rounds (synthetic from flat turns)
	for _, round := range SyntheticRoundsFromTurns(info.Turns) {
		gameInfo.Rounds = append(gameInfo.Rounds, DbGameRoundToProtoGameRound(round, info))
	}
	// conver results
	gameResult := DbGameResultToProtoGameResult(info.GameResult)
	gameInfo.Result = gameResult
	return gameInfo
}

func DbGameResultToProtoGameResult(result *dao.GameResult) *proto.GameResult {
	if result == nil {
		return nil
	}
	return &proto.GameResult{
		Multiplier:             int32(result.Multiplier),
		WinnerPlayerId:         result.WinnerPlayerId,
		WinnerTemporaryAddress: result.WinnerTemporaryAddress,
		GameResultType:         result.GameResultType,
	}
}

func DbBattleRewardToProtoBattleReward(battleReward *dao.BattleReward) *proto.BattleReward {
	if battleReward == nil {
		return nil
	}
	return &proto.BattleReward{
		PlayerRewards: DbPlayerRewardsToProto(battleReward.PlayerRewards),
		SystemFee:     int32(battleReward.SystemFee),
	}
}

func DbPlayerRewardsToProto(playerReward []*dao.PlayerReward) []*proto.PlayerReward {
	if len(playerReward) == 0 {
		return nil
	}
	var playerRewards []*proto.PlayerReward
	for _, playerReward := range playerReward {
		playerRewards = append(playerRewards, &proto.PlayerReward{
			PlayerId:               playerReward.PlayerId,
			TemporaryAddress:       playerReward.TemporaryAddress,
			TokenChange:            int32(playerReward.TokenChange),
			PointChange:            int32(playerReward.PointChange),
			Offline:                playerReward.IsOffline,
			Surrendered:            playerReward.Surrendered,
			PlayerGameResultStatus: playerReward.PlayerGameResultStatus,
		})
	}
	return playerRewards
}

func DbGamePlayerToProtoPlayerAddress(player *dao.GamePlayerInfo) *proto.PlayerAddress {
	if player == nil {
		return nil
	}
	return &proto.PlayerAddress{
		Id:               player.PlayerId,
		TemporaryAddress: player.TemporaryAddress,
	}
}

func DbGameRoundToProtoGameRound(round *RoundView, game *dao.Game) *proto.Round {
	if round == nil {
		return nil
	}
	// Convert Turns to PlayerRoundInfos for proto (backward compatibility)
	// Aggregate all PlayerTurnInfos from all turns
	playerRoundInfoMap := make(map[string]*proto.PlayerRoundInfo)
	for _, turn := range round.Turns {
		for _, playerTurnInfo := range turn.PlayerTurnInfos {
			key := playerTurnInfo.TemporaryAddress
			if _, exists := playerRoundInfoMap[key]; !exists {
				playerRoundInfoMap[key] = &proto.PlayerRoundInfo{
					PlayerAddress: &proto.PlayerAddress{
						Id:               playerTurnInfo.PlayerID,
						TemporaryAddress: playerTurnInfo.TemporaryAddress,
					},
					PlayerReady: playerTurnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_READY ||
						playerTurnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED ||
						playerTurnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED,
					SubmittedCards: make([]*proto.RoundSubmittedCard, 0),
				}
			}
			// Convert TurnSubmittedCard to proto RoundSubmittedCard
			if playerTurnInfo.TurnSubmittedCard != nil {
				roundCard := TurnSubmittedCardToProtoRoundSubmittedCard(playerTurnInfo.TurnSubmittedCard, turn.TurnNumber)
				playerRoundInfoMap[key].SubmittedCards = append(playerRoundInfoMap[key].SubmittedCards, roundCard)
			}
		}
	}

	// Convert map to slice
	playerRoundInfos := make([]*proto.PlayerRoundInfo, 0, len(playerRoundInfoMap))
	for _, pri := range playerRoundInfoMap {
		playerRoundInfos = append(playerRoundInfos, pri)
	}

	// Determine status from CompleteReason
	status := proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION
	if round.CompleteReason != proto.RoundCompleteReason_ROUND_COMPLETE_NORMAL {
		status = proto.TurnStatus_TURN_ROUND_COMPLETED
	} else if len(round.Turns) > 0 {
		status = proto.TurnStatus_TURN_WAITTING_COMMITMENTS
	}

	return &proto.Round{
		Number:           int32(round.RoundNumber),
		Status:           status,
		PlayerRoundInfos: playerRoundInfos,
	}
}

// TurnSubmittedCardToProtoRoundSubmittedCard converts TurnSubmittedCard to proto RoundSubmittedCard
func TurnSubmittedCardToProtoRoundSubmittedCard(card *dao.TurnSubmittedCard, turnNumber uint32) *proto.RoundSubmittedCard {
	if card == nil {
		return nil
	}
	return &proto.RoundSubmittedCard{
		PlayerHealthBefore:  card.HealthBefore,
		PlayerHealthEnd:     card.HealthAfter,
		Description:         card.Description,
		ElementRelation:     card.ElementRelation,
		SubmittedCardId:     card.CardID,
		SubmittedCommitment: card.CommitmentHash,
		Salt:                card.Salt,
	}
}

func DbRoundToRoundResult(round *RoundView, game *dao.Game) *proto.RoundResult {
	if round == nil {
		return nil
	}
	// Convert Turns to PlayerRoundStats for proto
	// Aggregate all PlayerTurnInfos from all turns
	playerRoundStatMap := make(map[string]*proto.PlayerRoundStat)
	var totalLostHP map[string]int32 = make(map[string]int32)

	for _, turn := range round.Turns {
		for _, playerTurnInfo := range turn.PlayerTurnInfos {
			key := playerTurnInfo.TemporaryAddress
			if _, exists := playerRoundStatMap[key]; !exists {
				playerRoundStatMap[key] = &proto.PlayerRoundStat{
					PlayerId:         playerTurnInfo.PlayerID,
					TemporaryAddress: playerTurnInfo.TemporaryAddress,
					CardStats:        make([]*proto.PlayerCardStat, 0),
				}
				totalLostHP[key] = 0
			}
			// Convert TurnSubmittedCard to PlayerCardStat
			if playerTurnInfo.TurnSubmittedCard != nil {
				cardStat := TurnSubmittedCardToProtoPlayerCardStat(len(playerRoundStatMap[key].CardStats), playerTurnInfo.TurnSubmittedCard)
				playerRoundStatMap[key].CardStats = append(playerRoundStatMap[key].CardStats, cardStat)
				// Calculate lost HP
				if playerTurnInfo.TurnSubmittedCard.HealthBefore > playerTurnInfo.TurnSubmittedCard.HealthAfter {
					totalLostHP[key] += int32(playerTurnInfo.TurnSubmittedCard.HealthBefore - playerTurnInfo.TurnSubmittedCard.HealthAfter)
				}
			}
		}
	}

	// Set LostHP for each player
	for key, stat := range playerRoundStatMap {
		stat.LostHP = totalLostHP[key]
	}

	// Convert map to slice
	playerRoundStats := make([]*proto.PlayerRoundStat, 0, len(playerRoundStatMap))
	for _, stat := range playerRoundStatMap {
		playerRoundStats = append(playerRoundStats, stat)
	}

	// Get round end time from latest turn if available
	var roundEndTime uint64
	if len(round.Turns) > 0 {
		latestTurn := round.Turns[len(round.Turns)-1]
		if latestTurn.TurnStartAt > 0 {
			roundEndTime = uint64(latestTurn.TurnStartAt)
		}
	}

	var maxR uint32
	if game != nil {
		maxR = dao.MaxRoundNumberFromTurns(game.Turns)
	}
	isGameOver := game != nil &&
		(game.Status == proto.GameStatus_GAME_END || game.Status == proto.GameStatus_GAME_ABORTED) &&
		maxR > 0 && round.RoundNumber == maxR

	return &proto.RoundResult{
		Players:      playerRoundStats,
		RoundNumber:  round.RoundNumber,
		IsGameOver:   isGameOver,
		RoundEndTime: roundEndTime,
	}
}

// TurnSubmittedCardToProtoPlayerCardStat converts TurnSubmittedCard to proto PlayerCardStat
func TurnSubmittedCardToProtoPlayerCardStat(cardNumber int, card *dao.TurnSubmittedCard) *proto.PlayerCardStat {
	if card == nil {
		return nil
	}
	return &proto.PlayerCardStat{
		CardNumber:      int32(cardNumber),
		CardID:          int32(card.CardID),
		HPBefore:        int32(card.HealthBefore),
		HPAfter:         int32(card.HealthAfter),
		Description:     card.Description,
		ElementRelation: card.ElementRelation,
	}
}

func DbGameToProtoGamePhase(game *dao.Game, currentRound *RoundView, turnNumber uint32, turnStartAt int64) *proto.GamePhase {
	if game == nil || currentRound == nil {
		return nil
	}

	// Find current turn
	var currentTurn *dao.Turn
	for _, turn := range currentRound.Turns {
		if turn.TurnNumber == turnNumber {
			currentTurn = turn
			break
		}
	}

	// Get player count from turns or estimate
	playerCount := 0
	if currentTurn != nil {
		playerCount = len(currentTurn.PlayerTurnInfos)
	} else if len(currentRound.Turns) > 0 {
		playerCount = len(currentRound.Turns[0].PlayerTurnInfos)
	}

	gamePhase := &proto.GamePhase{
		GameType:    proto.GameType(game.Type),
		GameID:      game.ID,
		RoundNumber: uint32(currentRound.RoundNumber),
		TurnNumber:  turnNumber,
		TurnStartAt: turnStartAt,
		Players:     make([]*proto.GamePhasePlayer, 0, playerCount),
	}

	initialHP := uint32(game.GameArgs.InitialHP)

	// Build players from current turn's PlayerTurnInfos
	if currentTurn != nil {
		for _, playerTurnInfo := range currentTurn.PlayerTurnInfos {
			addr := types.NewPlayerAddress(
				playerTurnInfo.PlayerID,
				playerTurnInfo.TemporaryAddress,
			).ToProto()

			// Get turn status from PlayerTurnInfo
			turnStatus := playerTurnInfo.PlayerStatus
			var commitment *[]byte
			var card *uint32

			// Get card info from TurnSubmittedCard if available
			if playerTurnInfo.TurnSubmittedCard != nil {
				if playerTurnInfo.TurnSubmittedCard.CardID != 0 {
					cardVal := playerTurnInfo.TurnSubmittedCard.CardID
					card = &cardVal
				}
				if len(playerTurnInfo.TurnSubmittedCard.CommitmentHash) > 0 {
					commitmentVal := playerTurnInfo.TurnSubmittedCard.CommitmentHash
					commitment = &commitmentVal
				}
			}

			// Current HP: after battle use HealthAfter; before resolution the stub only has HealthBefore (HealthAfter is still zero).
			var currentHP uint32
			if c := playerTurnInfo.TurnSubmittedCard; c != nil {
				if c.Description != "" || c.HealthAfter != 0 {
					currentHP = c.HealthAfter
				} else {
					currentHP = c.HealthBefore
				}
			} else {
				currentHP = initialHP
			}

			player := &proto.GamePhasePlayer{
				Address:    addr,
				TurnStatus: turnStatus,
				CurrentHP:  currentHP,
			}

			if commitment != nil {
				player.Commitment = *commitment
			}
			if card != nil {
				player.Card = card
			}

			gamePhase.Players = append(gamePhase.Players, player)
		}
	} else {
		// No turn found, return empty game phase with initial state
		// This can happen if turn hasn't been created yet
	}

	return gamePhase
}
