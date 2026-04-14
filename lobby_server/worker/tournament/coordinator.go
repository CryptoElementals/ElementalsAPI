// Package tournament implements tournament bracket scheduling and match starts (separate from PVP matchmaking queue).
package tournament

import (
	"context"
	"errors"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/lobby_server/bot_manager"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const tickInterval = 1 * time.Second

// GameCreator starts tournament matches via RoomWorkerService.CreateGameAndRun.
type GameCreator interface {
	CreateGameAndRun(players []types.PlayerAddress, gameType uint, completedMatchID int64) (int64, error)
}

type coordinator struct {
	ctx                context.Context
	cancel             context.CancelFunc
	publisher          pubsub.Publisher
	botStore           *bot_manager.RedisStore
	gameCreator        GameCreator
	entryFee           int32
	minPlayersRequired uint32
	intervalSeconds    uint32
	beforeStartSeconds uint32
}

func newCoordinator(parent context.Context, publisher pubsub.Publisher, botStore *bot_manager.RedisStore, gameCreator GameCreator, entryFee int32, minPlayersRequired uint32, intervalSeconds uint32, beforeStartSeconds uint32) *coordinator {
	ctx, cancel := context.WithCancel(parent)
	return &coordinator{
		ctx:                ctx,
		cancel:             cancel,
		publisher:          publisher,
		botStore:           botStore,
		gameCreator:        gameCreator,
		entryFee:           entryFee,
		minPlayersRequired: minPlayersRequired,
		intervalSeconds:    intervalSeconds,
		beforeStartSeconds: beforeStartSeconds,
	}
}

func (tc *coordinator) start() {
	log.Debugw("tournament coordinator start")
	go tc.loop()
}

func (tc *coordinator) stop() {
	tc.cancel()
}

func (tc *coordinator) loop() {
	tick := time.NewTicker(tickInterval)
	defer tick.Stop()
	log.Debugw("tournament coordinator loop start")
	for {
		select {
		case <-tc.ctx.Done():
			return
		case <-tick.C:
			tc.tick()
		}
	}
}

// 1. 超过整数倍单元时间(支持配置，如1小时, 10分钟)，创建下一个tournament
func (tc *coordinator) tick() {
	now := time.Now().UTC()

	if err := tc.ensureNextTournaments(now); err != nil {
		log.Errorw("tournament: ensure next tournaments", "err", err)
		return
	}

	//2. 待匹配players
	// Grace window: if scheduler runs a little late (e.g. process restart at +1s), still begin this slot.
	tournamentToBegin, err := db.TournamentGetLatestRegistrationOpenWithinStartGrace(now, 10*time.Second)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			//log.Debugw("tournament: no tournament to begin in grace window", "now", now)
			return
		}
		log.Errorw("tournament: get latest tournament to begin", "err", err)
		return
	}
	if err := tc.beginTournament(tournamentToBegin); err != nil {
		log.Errorw("tournament: begin", "tournament_id", tournamentToBegin.ID, "err", err)
	}
}

func (tc *coordinator) ensureNextTournaments(now time.Time) error {
	unitDuration := time.Duration(tc.intervalSeconds) * time.Second
	nextSlot := now.Truncate(unitDuration).Add(unitDuration)

	latest, err := db.TournamentGetLatestByScheduledStart()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// First bootstrap: always create tournament at the next unit boundary.
		return tc.createTournamentIfNotExists(nextSlot)
	}
	if err != nil {
		return err
	}

	if latest.ScheduledStartAt.Before(nextSlot) {
		return tc.createTournamentIfNotExists(nextSlot)
	}

	return nil
}

func (tc *coordinator) createTournamentIfNotExists(at time.Time) error {
	if _, err := db.TournamentGetByScheduledStart(at); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	t := &dao.Tournament{
		TournamentID:         strconv.FormatInt(dao.GenerateSnowflakeID(), 10),
		Status:               dao.TournamentStatusRegistrationOpen,
		ScheduledStartAt:     at,
		ScheduledEndDeadline: at.Add(time.Duration(tc.intervalSeconds) * time.Second),
		RegistrationDeadline: at.Add(-time.Duration(tc.beforeStartSeconds) * time.Second),
		EntryFee:             tc.entryFee,
	}
	return db.TournamentCreate(t)
}

func (tc *coordinator) beginTournament(t *dao.Tournament) error {
	var overflowPlayers []types.PlayerAddress
	var newMatchIDs []uint
	var createdRound bool

	err := db.Get().Transaction(func(tx *gorm.DB) error {
		var cur dao.Tournament
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cur, t.ID).Error; err != nil {
			return err
		}
		if cur.Status != dao.TournamentStatusRegistrationOpen {
			return nil
		}
		// idempotent: round1 exists means already matched.
		if _, rerr := db.TournamentGetRound(tx, cur.TournamentID, 1); rerr == nil {
			return nil
		} else if !errors.Is(rerr, gorm.ErrRecordNotFound) {
			return rerr
		}

		queued, err := db.TournamentListParticipantsByStatus(tx, cur.TournamentID, dao.TournamentParticipantStatusQueued)
		if err != nil {
			return err
		}

		total := len(queued)
		capN := minInt(floorPow2(total), 8192)
		if capN < int(tc.minPlayersRequired) {
			cur.Status = dao.TournamentStatusCancelled
			if err := tx.Save(&cur).Error; err != nil {
				return err
			}

			log.Infow("tournament: cancel tournament because not enough players",
				"tournament_id", cur.TournamentID, "total players", total, "minPlayersRequired", tc.minPlayersRequired)
			for i := range queued {
				p := &queued[i]
				p.Status = dao.TournamentParticipantStatusKickedNotEnough
				if err := tx.Save(p).Error; err != nil {
					return err
				}
				overflowPlayers = append(overflowPlayers, types.PlayerAddress{Id: p.PlayerID, TemporaryAddress: p.TempAddress})
			}
			return nil
		}

		cur.Status = dao.TournamentStatusInProgress
		if err := tx.Save(&cur).Error; err != nil {
			return err
		}

		round := &dao.TournamentRound{
			TournamentID: cur.TournamentID,
			RoundNo:      1,
			Status:       dao.TournamentRoundStatusMatched,
		}
		if err := db.TournamentCreateRound(tx, round); err != nil {
			return err
		}
		createdRound = true

		// In-bracket and overflow marking.
		for i := range queued {
			p := &queued[i]
			if i < capN {
				p.Status = dao.TournamentParticipantStatusInProgress
			} else {
				p.Status = dao.TournamentParticipantStatusKickedOverflow
				overflowPlayers = append(overflowPlayers, types.PlayerAddress{Id: p.PlayerID, TemporaryAddress: p.TempAddress})
			}
			if err := tx.Save(p).Error; err != nil {
				return err
			}
		}

		// Pair for round1 (no bye because cap is power of 2).
		firstRoundMatchNo := uint32(1)
		for i := 0; i < capN; i += 2 {
			match := &dao.TournamentMatch{
				TournamentID:       cur.TournamentID,
				RoundNo:            firstRoundMatchNo,
				MatchNo:            uint32(i/2 + 1),
				Player1ID:          queued[i].PlayerID,
				Player1TempAddress: queued[i].TempAddress,
				Player2ID:          queued[i+1].PlayerID,
				Player2TempAddress: queued[i+1].TempAddress,
				Status:             dao.TournamentMatchStatusMatched,
			}
			if err := db.TournamentCreateMatch(tx, match); err != nil {
				return err
			}
			log.Infow("tournament: create match", "TournamentID", match.TournamentID, "round_no", match.RoundNo,
				"match_no", match.MatchNo, "match_id", match.ID, "player1_id", match.Player1ID, "player2_id", match.Player2ID)
			newMatchIDs = append(newMatchIDs, match.ID)
		}
		return nil
	})
	if err != nil {
		log.Errorw("tournament: create tournament first round failed", "tournament_id", t.TournamentID, "err", err)
		return err
	}

	for _, p := range overflowPlayers {
		if err := db.UnlockUserToken(tc.ctx, p.Id, p.TemporaryAddress, true); err != nil {
			log.Warnw("tournament: unlock overflow token failed", "player_id", p.Id, "temp_address", p.TemporaryAddress, "err", err)
		}
	}

	anyPlaying := tc.startGamesForNewMatches(newMatchIDs)

	if createdRound && t.TournamentID != "" {
		if r, rerr := db.TournamentGetRound(db.Get(), t.TournamentID, 1); rerr == nil {
			if anyPlaying {
				r.Status = dao.TournamentRoundStatusPlaying
			}
			_ = db.TournamentSaveRound(db.Get(), r)
		}
	}
	return nil
}

// startGamesForNewMatches calls room worker for each match and persists game_id + playing status.
func (tc *coordinator) startGamesForNewMatches(matchIDs []uint) bool {
	anyPlaying := false
	for _, matchID := range matchIDs {
		var m dao.TournamentMatch
		if err := db.Get().First(&m, matchID).Error; err != nil {
			log.Errorw("tournament: load match failed", "match_id", matchID, "err", err)
			continue
		}
		if m.Player1ID == 0 || m.Player2ID == 0 || m.Player1TempAddress == "" || m.Player2TempAddress == "" {
			log.Errorw("tournament: match player1 or player2 is empty", "match_id", matchID,
				"Player1ID", m.Player1ID, "Player1TempAddress", m.Player1TempAddress, "Player2ID", m.Player2ID, "Player2TempAddress", m.Player2TempAddress)
			continue
		}
		players := []types.PlayerAddress{
			{Id: m.Player1ID, TemporaryAddress: m.Player1TempAddress},
			{Id: m.Player2ID, TemporaryAddress: m.Player2TempAddress},
		}
		gameID, gerr := tc.gameCreator.CreateGameAndRun(players, types.GameTypeTournament, 0)
		if gerr != nil {
			log.Errorw("tournament: create game failed", "match_id", matchID, "err", gerr)
			continue
		}
		log.Infow("tournament: create game success", "TournamentID", m.TournamentID, "round_no", m.RoundNo,
			"match_no", m.MatchNo, "match_id", m.ID, "game_id", gameID, "players", types.ToJsonLoggable(players))
		m.GameID = &gameID
		m.Status = dao.TournamentMatchStatusPlaying
		if err := db.TournamentSaveMatch(db.Get(), &m); err != nil {
			log.Errorw("tournament: save game id failed", "match_id", matchID, "err", err)
			continue
		}
		anyPlaying = true
	}
	return anyPlaying
}

func floorPow2(n int) int {
	if n < 1 {
		return 0
	}
	return 1 << (bits.Len(uint(n)) - 1)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// tournamentMatchOutcomeToPublish is filled inside the bracket transaction only on successful commit paths.
type tournamentMatchOutcomeToPublish struct {
	gameID                         int64
	tournamentID                   string
	roundNo                        uint32
	matchNo                        uint32
	winnerPID                      int64
	winnerTemp                     string
	loserPID                       int64
	loserTemp                      string
	tournamentFinished             bool
	roundFinished                  bool                // all matches in roundNo completed after this game
	cumulativeBattleReward         *proto.BattleReward // SUM(tournament_rewards) per player; PlayerRewards order winner, loser
	winnerCumulationIfNextMatchWin *proto.PlayerReward // set when tournament not finished
	nextRoundNo                    *uint32             // set when tournament not finished
}

func (tc *coordinator) publishTournamentMatchOutcome(o *tournamentMatchOutcomeToPublish) {
	if o == nil {
		return
	}
	winner := types.NewPlayerAddress(o.winnerPID, o.winnerTemp)
	loser := types.NewPlayerAddress(o.loserPID, o.loserTemp)
	w := winner.ToProto()
	l := loser.ToProto()
	evt := &proto.Event{
		Type: proto.EventType_TYPE_TOURNAMENT_MATCH_OUTCOME,
		Receivers: []*proto.PlayerAddress{
			w, l,
		},
		Event: &proto.Event_TournamentMatchOutcome{
			TournamentMatchOutcome: &proto.TournamentMatchOutcome{
				GameId:                         o.gameID,
				TournamentId:                   o.tournamentID,
				RoundNo:                        o.roundNo,
				MatchNo:                        o.matchNo,
				Winner:                         w,
				Loser:                          l,
				TournamentFinished:             o.tournamentFinished,
				RoundFinished:                  o.roundFinished,
				CumulativeBattleReward:         o.cumulativeBattleReward,
				WinnerCumulationIfNextMatchWin: o.winnerCumulationIfNextMatchWin,
				NextRoundNo:                    o.nextRoundNo,
			},
		},
	}
	if err := pubsub.Publish(tc.ctx, tc.publisher, pubsub.TopicLobby, evt); err != nil {
		log.Errorw("tournament: publish match outcome failed", "game_id", o.gameID, "err", err)
	}
}

func (tc *coordinator) publishTournamentRosterUpdate(tournamentID string) {
	if tc == nil || tc.publisher == nil || tournamentID == "" {
		return
	}
	evt := &proto.Event{
		Type: proto.EventType_TYPE_TOURNAMENT_ROSTER_UPDATE,
		Event: &proto.Event_TournamentRosterUpdate{
			TournamentRosterUpdate: &proto.TournamentRosterUpdate{TournamentID: tournamentID},
		},
	}
	if err := pubsub.Publish(tc.ctx, tc.publisher, pubsub.TopicTournamentRoster, evt); err != nil {
		log.Errorw("tournament: publish roster update failed", "tournament_id", tournamentID, "err", err)
	}
}

func (tc *coordinator) onGameCompleted(gameID int64) error {
	gr, err := db.LoadGameResultByGameID(gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if gr.GameResultType == proto.GameResultType_GAME_ABORTED {
		return nil
	}
	m0, err := db.TournamentGetMatchByGameID(gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		log.Errorw("tournament: get match by game id failed", "game_id", gameID, "err", err)
		return err
	}
	if m0.WinnerPlayerID != nil && *m0.WinnerPlayerID != 0 {
		return nil
	}
	if m0.WinnerTempAddress != nil && *m0.WinnerTempAddress != "" {
		return nil
	}

	var (
		unlockLosers   []types.PlayerAddress
		unlockChampion []types.PlayerAddress
		postNewIDs     []uint
		postNextRound  uint32
		postTournID    string
		publishOutcome *tournamentMatchOutcomeToPublish
		cumReward      *proto.BattleReward
	)

	err = db.Get().Transaction(func(tx *gorm.DB) error {
		var locked dao.TournamentMatch
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("game_id = ?", gameID).First(&locked).Error; err != nil {
			return err
		}
		if locked.WinnerPlayerID != nil && *locked.WinnerPlayerID != 0 {
			return nil
		}
		if locked.WinnerTempAddress != nil && *locked.WinnerTempAddress != "" {
			return nil
		}

		winPID, winTemp, err := pickTournamentWinnerTx(tx, &locked, gr)
		if err != nil {
			return err
		}
		winTempNorm := strings.ToLower(strings.TrimSpace(winTemp))
		wp := winPID
		wt := winTempNorm
		locked.WinnerPlayerID = &wp
		locked.WinnerTempAddress = &wt
		locked.Status = dao.TournamentMatchStatusCompleted //update match status to completed
		if err := tx.Save(&locked).Error; err != nil {
			return err
		}

		var loserPID int64
		var loserTemp string
		if winPID == locked.Player1ID && strings.EqualFold(winTempNorm, strings.TrimSpace(locked.Player1TempAddress)) {
			loserPID, loserTemp = locked.Player2ID, locked.Player2TempAddress
		} else if winPID == locked.Player2ID && strings.EqualFold(winTempNorm, strings.TrimSpace(locked.Player2TempAddress)) {
			loserPID, loserTemp = locked.Player1ID, locked.Player1TempAddress
		} else {
			if winPID == locked.Player1ID {
				loserPID, loserTemp = locked.Player2ID, locked.Player2TempAddress
			} else {
				loserPID, loserTemp = locked.Player1ID, locked.Player1TempAddress
			}
		}

		loserPart, err := db.TournamentGetParticipantByPlayerTx(tx, locked.TournamentID, loserPID, loserTemp)
		if err != nil {
			return fmt.Errorf("tournament loser participant: %w", err)
		}
		loserPart.Status = dao.TournamentParticipantStatusEliminated
		if err := tx.Save(loserPart).Error; err != nil {
			return err
		}
		loserTempNorm := strings.ToLower(strings.TrimSpace(loserTemp))
		unlockLosers = append(unlockLosers, types.PlayerAddress{
			Id:               loserPID,
			TemporaryAddress: loserTempNorm,
		})

		rwPID, rwTemp, _ := db.WinnerFromPlayerResultInfos(gr.PlayerResultInfos)
		log.Infow("tournament: game completed - room GameResult vs bracket winner (reward branch follows game_result_type)",
			"game_id", gameID,
			"tournament_id", locked.TournamentID,
			"round_no", locked.RoundNo,
			"match_no", locked.MatchNo,
			"game_result_type", gr.GameResultType.String(),
			"room_winner_player_id", rwPID,
			"room_winner_temp", rwTemp,
			"bracket_winner_player_id", winPID,
			"bracket_winner_temp", winTempNorm,
			"bracket_loser_player_id", loserPID,
			"bracket_loser_temp", loserTempNorm,
		)
		if _, rerr := db.TournamentApplyMatchRewardsTx(tx, gr.GameResultType, locked.TournamentID, locked.RoundNo, locked.MatchNo,
			locked.Player1ID, locked.Player1TempAddress,
			locked.Player2ID, locked.Player2TempAddress,
			winPID, winTempNorm, loserPID, loserTempNorm); rerr != nil {
			return rerr
		}
		var cerr error
		cumReward, cerr = db.TournamentCumulativeBattleRewardProtoTx(tx, locked.TournamentID, winPID, winTempNorm, loserPID, loserTempNorm)
		if cerr != nil {
			return cerr
		}

		roundMatches, err := db.TournamentListMatchesForRound(tx, locked.TournamentID, locked.RoundNo)
		if err != nil {
			return err
		}
		for _, rm := range roundMatches {
			if rm.Status != dao.TournamentMatchStatusCompleted { // one match not completed yet, then round not completed
				nextRN := locked.RoundNo + 1
				winNext, werr := winnerCumulationIfNextMatchWinProto(tx, locked.TournamentID, winPID, winTempNorm)
				if werr != nil {
					return werr
				}
				publishOutcome = &tournamentMatchOutcomeToPublish{
					gameID:                         gameID,
					tournamentID:                   locked.TournamentID,
					roundNo:                        locked.RoundNo,
					matchNo:                        locked.MatchNo,
					winnerPID:                      winPID,
					winnerTemp:                     winTempNorm,
					loserPID:                       loserPID,
					loserTemp:                      loserTempNorm,
					tournamentFinished:             false,
					roundFinished:                  false,
					cumulativeBattleReward:         cumReward,
					winnerCumulationIfNextMatchWin: winNext,
					nextRoundNo:                    &nextRN,
				}
				return nil
			}
		}

		roundRow, err := db.TournamentGetRound(tx, locked.TournamentID, locked.RoundNo)
		if err != nil {
			return err
		}
		roundRow.Status = dao.TournamentRoundStatusCompleted
		if err := db.TournamentSaveRound(tx, roundRow); err != nil {
			return err
		}

		survivors, err := db.TournamentListParticipantsByStatus(tx, locked.TournamentID, dao.TournamentParticipantStatusInProgress)
		if err != nil {
			return err
		}
		n := len(survivors)
		if n == 0 {
			return fmt.Errorf("tournament %s: no in_progress participants after round", locked.TournamentID)
		}
		if n == 1 {
			var tour dao.Tournament
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tournament_id = ?", locked.TournamentID).First(&tour).Error; err != nil {
				return err
			}
			tour.Status = dao.TournamentStatusFinished
			if err := tx.Save(&tour).Error; err != nil {
				return err
			}
			ch := &survivors[0]
			ch.Status = dao.TournamentParticipantStatusChampion
			if err := tx.Save(ch).Error; err != nil {
				return err
			}
			unlockChampion = append(unlockChampion, types.PlayerAddress{
				Id:               ch.PlayerID,
				TemporaryAddress: strings.ToLower(strings.TrimSpace(ch.TempAddress)),
			})
			publishOutcome = &tournamentMatchOutcomeToPublish{
				gameID:                 gameID,
				tournamentID:           locked.TournamentID,
				roundNo:                locked.RoundNo,
				matchNo:                locked.MatchNo,
				winnerPID:              winPID,
				winnerTemp:             winTempNorm,
				loserPID:               loserPID,
				loserTemp:              loserTempNorm,
				tournamentFinished:     true,
				roundFinished:          true,
				cumulativeBattleReward: cumReward,
			}
			return nil
		}
		if n%2 != 0 {
			return fmt.Errorf("tournament %s: odd in_progress count %d", locked.TournamentID, n)
		}

		nextNo := locked.RoundNo + 1
		newRound := &dao.TournamentRound{
			TournamentID: locked.TournamentID,
			RoundNo:      nextNo,
			Status:       dao.TournamentRoundStatusMatched,
		}
		if err := db.TournamentCreateRound(tx, newRound); err != nil {
			return err
		}
		for i := 0; i < n; i += 2 {
			a, b := survivors[i], survivors[i+1]
			match := &dao.TournamentMatch{
				TournamentID:       locked.TournamentID,
				RoundNo:            nextNo,
				MatchNo:            uint32(i/2 + 1),
				Player1ID:          a.PlayerID,
				Player1TempAddress: a.TempAddress,
				Player2ID:          b.PlayerID,
				Player2TempAddress: b.TempAddress,
				Status:             dao.TournamentMatchStatusMatched,
			}
			if err := db.TournamentCreateMatch(tx, match); err != nil {
				return err
			}
			postNewIDs = append(postNewIDs, match.ID)
		}
		postNextRound = nextNo
		postTournID = locked.TournamentID
		winNext, werr := winnerCumulationIfNextMatchWinProto(tx, locked.TournamentID, winPID, winTempNorm)
		if werr != nil {
			return werr
		}
		publishOutcome = &tournamentMatchOutcomeToPublish{
			gameID:                         gameID,
			tournamentID:                   locked.TournamentID,
			roundNo:                        locked.RoundNo,
			matchNo:                        locked.MatchNo,
			winnerPID:                      winPID,
			winnerTemp:                     winTempNorm,
			loserPID:                       loserPID,
			loserTemp:                      loserTempNorm,
			tournamentFinished:             false,
			roundFinished:                  true,
			cumulativeBattleReward:         cumReward,
			winnerCumulationIfNextMatchWin: winNext,
			nextRoundNo:                    &nextNo,
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, p := range unlockLosers {
		if err := db.UnlockUserToken(tc.ctx, p.Id, p.TemporaryAddress, true); err != nil {
			log.Warnw("tournament: unlock loser token failed", "player_id", p.Id, "temp_address", p.TemporaryAddress, "err", err)
		}
	}
	for _, p := range unlockChampion {
		if err := db.UnlockUserToken(tc.ctx, p.Id, p.TemporaryAddress, true); err != nil {
			log.Warnw("tournament: unlock champion token failed", "player_id", p.Id, "temp_address", p.TemporaryAddress, "err", err)
		}
	}

	if publishOutcome != nil {
		tc.publishTournamentMatchOutcome(publishOutcome)
	}

	if len(postNewIDs) > 0 {
		anyPlaying := tc.startGamesForNewMatches(postNewIDs)
		if anyPlaying && postTournID != "" {
			if r, rerr := db.TournamentGetRound(db.Get(), postTournID, postNextRound); rerr == nil {
				r.Status = dao.TournamentRoundStatusPlaying
				_ = db.TournamentSaveRound(db.Get(), r)
			}
		}
	}

	return nil
}

// winnerCumulationIfNextMatchWinProto returns winner cumulative token/point totals if they also win their next match (current sum + one match-win bonus).
func winnerCumulationIfNextMatchWinProto(tx *gorm.DB, tournamentID string, winPID int64, winTempNorm string) (*proto.PlayerReward, error) {
	wtok, wpt, err := db.TournamentSumPlayerRewardTotalsTx(tx, tournamentID, winPID, winTempNorm)
	if err != nil {
		return nil, err
	}
	dtok, dpt := db.TournamentOneMatchWinRewardDelta()
	return &proto.PlayerReward{
		PlayerId:         winPID,
		TemporaryAddress: winTempNorm,
		TokenChange:      wtok + dtok,
		PointChange:      wpt + dpt,
	}, nil
}

// pickTournamentWinnerTx resolves winner player id and temp address from game result (for bracket update).
// On GAME_TIE, the player who joined the tournament earlier (participants.created_at) advances; the other is eliminated.
func pickTournamentWinnerTx(tx *gorm.DB, m *dao.TournamentMatch, gr *dao.GameResult) (winnerPID int64, winnerTemp string, err error) {
	if gr == nil {
		return 0, "", errors.New("game result missing")
	}
	p1id, p1t := m.Player1ID, m.Player1TempAddress
	p2id, p2t := m.Player2ID, m.Player2TempAddress

	addrMatch := func(pid int64, temp string, wid int64, wtemp string) bool {
		return pid == wid && strings.EqualFold(strings.TrimSpace(temp), strings.TrimSpace(wtemp))
	}

	switch gr.GameResultType {
	case proto.GameResultType_GAME_TIE:
		p1Part, err := db.TournamentGetParticipantByPlayerTx(tx, m.TournamentID, p1id, p1t)
		if err != nil {
			return 0, "", fmt.Errorf("tournament tie: load participant: %w", err)
		}
		p2Part, err := db.TournamentGetParticipantByPlayerTx(tx, m.TournamentID, p2id, p2t)
		if err != nil {
			return 0, "", fmt.Errorf("tournament tie: load participant: %w", err)
		}
		if p1Part.CreatedAt.Before(p2Part.CreatedAt) {
			return p1id, p1t, nil
		}
		if p2Part.CreatedAt.Before(p1Part.CreatedAt) {
			return p2id, p2t, nil
		}
		if p1Part.ID <= p2Part.ID {
			return p1id, p1t, nil
		}
		return p2id, p2t, nil
	case proto.GameResultType_GAME_NORMAL, proto.GameResultType_GAME_KO:
		wid, wt, wok := db.WinnerFromPlayerResultInfos(gr.PlayerResultInfos)
		if !wok {
			wid, wt = 0, ""
		}
		if addrMatch(p1id, p1t, wid, wt) {
			return p1id, p1t, nil
		}
		if addrMatch(p2id, p2t, wid, wt) {
			return p2id, p2t, nil
		}
		if p1id <= p2id {
			return p1id, p1t, nil
		}
		return p2id, p2t, nil
	default:
		return 0, "", fmt.Errorf("unsupported game result type: %v", gr.GameResultType)
	}
}
