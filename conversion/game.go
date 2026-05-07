package conversion

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

func DbGameResultToProtoGameResult(result *dao.GameResult) *proto.GameResult {
	if result == nil {
		return nil
	}
	wid, wt, _ := db.WinnerFromPlayerResultInfos(result.PlayerResultInfos)
	return &proto.GameResult{
		Multiplier:             int32(result.Multiplier),
		WinnerPlayerId:         wid,
		WinnerTemporaryAddress: wt,
		GameResultType:         result.GameResultType,
	}
}

// PlayerGameResultStatusPtrFromGameResult returns the persisted per-player outcome, or nil if absent.
func PlayerGameResultStatusPtrFromGameResult(gr *dao.GameResult, playerID int64) *proto.PlayerGameResultStatus {
	if gr == nil {
		return nil
	}
	if pri, ok := db.PlayerResultInfoByPlayerID(gr.PlayerResultInfos)[playerID]; ok && pri != nil {
		st := pri.PlayerGameResultStatus
		return &st
	}
	return nil
}

func DbBattleRewardToProtoBattleReward(br *dao.BattleRewardPVP, infos []*dao.PlayerResultInfo) *proto.BattleReward {
	if br == nil {
		return nil
	}
	return &proto.BattleReward{
		PlayerRewards: DbPlayerRewardsToProto(br.PlayerRewards, infos),
		SystemFee:     int32(br.SystemFee),
	}
}

// DbPlayerRewardsToProto joins wallet rows with PlayerResultInfos for addresses and outcome enums.
func DbPlayerRewardsToProto(playerReward []*dao.PlayerReward, infos []*dao.PlayerResultInfo) []*proto.PlayerReward {
	if len(playerReward) == 0 {
		return nil
	}
	byID := db.PlayerResultInfoByPlayerID(infos)
	var playerRewards []*proto.PlayerReward
	for _, pr := range playerReward {
		if pr == nil {
			continue
		}
		out := &proto.PlayerReward{
			PlayerId:    pr.PlayerId,
			TokenChange: int32(pr.TokenChange),
			PointChange: int32(pr.PointChange),
		}
		if pri, ok := byID[pr.PlayerId]; ok && pri != nil {
			out.TemporaryAddress = pri.TemporaryAddress
			st := pri.PlayerGameResultStatus
			out.PlayerGameResultStatus = st
			out.Offline = st == proto.PlayerGameResultStatus_PLAYER_OFFLINE
			out.Surrendered = st == proto.PlayerGameResultStatus_PLAYER_SURRENDER
		}
		playerRewards = append(playerRewards, out)
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
						playerTurnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED ||
						playerTurnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_ON_CHAIN ||
						playerTurnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_CARD_ON_CHAIN,
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
		game.Status == proto.GameStatus_GAME_END &&
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

// playerTurnInfoInTurn returns this player's row for the given persisted turn, if present.
func playerTurnInfoInTurn(turn *dao.Turn, playerID int64, temporaryAddress string) *dao.PlayerTurnInfo {
	if turn == nil {
		return nil
	}
	wantAddr := strings.ToLower(temporaryAddress)
	for _, pti := range turn.PlayerTurnInfos {
		if pti == nil {
			continue
		}
		if pti.PlayerID == playerID && strings.ToLower(pti.TemporaryAddress) == wantAddr {
			return pti
		}
	}
	return nil
}

// turnCardPlayingInfosForCurrentRound builds one TurnCardPlayingInfo per turn in the current round with
// TurnNumber <= activeTurn (compare TurnCardPlayingInfo.TurnNumber to GamePhase.TurnNumber on the client).
func turnCardPlayingInfosForCurrentRound(round *RoundView, activeTurn uint32, playerID int64, temporaryAddress string) []*proto.TurnCardPlayingInfo {
	if round == nil {
		return nil
	}
	var out []*proto.TurnCardPlayingInfo
	for _, turn := range round.Turns {
		if turn == nil || turn.TurnNumber > activeTurn {
			break
		}
		cpi := &proto.TurnCardPlayingInfo{TurnNumber: turn.TurnNumber}
		pti := playerTurnInfoInTurn(turn, playerID, temporaryAddress)
		if pti != nil && pti.TurnSubmittedCard != nil {
			c := pti.TurnSubmittedCard
			if len(c.CommitmentHash) > 0 {
				cpi.Commitment = append([]byte(nil), c.CommitmentHash...)
			}
			if c.CardID != 0 {
				cpi.Card = c.CardID
			}
		}
		out = append(out, cpi)
	}
	return out
}

// gamePhaseTimeoutSeconds picks the per-phase deadline from [dao.GameArgs] (seconds) for the single
// GamePhase.Timeout field: confirmation while waiting for battle confirm, card while waiting for cards.
func gamePhaseTimeoutSeconds(turnStatus proto.TurnStatus, args *dao.GameArgs) int64 {
	if args == nil {
		return 0
	}
	switch turnStatus {
	case proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION:
		return args.ConfirmationTimeout
	case proto.TurnStatus_TURN_WAITTING_COMMITMENTS:
		return args.CommitmentSubmissionTimeout
	default:
		return 0
	}
}

func DbGameToProtoGamePhase(game *dao.Game, currentRound *RoundView, turnNumber uint32, turnStartAt int64, caller types.PlayerAddress) *proto.GamePhase {
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
		GameType:         proto.GameType(game.Type),
		GameID:           game.ID,
		RoundNumber:      uint32(currentRound.RoundNumber),
		TurnNumber:       turnNumber,
		TurnStartAt:      turnStartAt,
		PlayerTurnStatus: proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN,
		Players:          make([]*proto.GamePhasePlayer, 0, playerCount),
	}

	if currentTurn != nil {
		gamePhase.TurnStatus = proto.TurnStatus(currentTurn.TurnStatus)
		gamePhase.Timeout = gamePhaseTimeoutSeconds(gamePhase.TurnStatus, game.GameArgs)
		// Only the SyncGamePhase caller's per-player status (not other players').
		if pti := playerTurnInfoInTurn(currentTurn, caller.Id, caller.TemporaryAddress); pti != nil {
			gamePhase.PlayerTurnStatus = pti.PlayerStatus
		}
	}

	initialHP := uint32(game.GameArgs.InitialHP)

	// Build players from current turn's PlayerTurnInfos
	if currentTurn != nil {
		for _, playerTurnInfo := range currentTurn.PlayerTurnInfos {
			addr := types.NewPlayerAddress(
				playerTurnInfo.PlayerID,
				playerTurnInfo.TemporaryAddress,
			).ToProto()

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
				Address:              addr,
				CurrentHP:            currentHP,
				TurnCardPlayingInfos: turnCardPlayingInfosForCurrentRound(currentRound, turnNumber, playerTurnInfo.PlayerID, playerTurnInfo.TemporaryAddress),
			}

			gamePhase.Players = append(gamePhase.Players, player)
		}
	} else {
		// No turn found, return empty game phase with initial state
		// This can happen if turn hasn't been created yet
	}

	return gamePhase
}
