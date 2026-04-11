// Package tournament implements tournament bracket scheduling and match starts (separate from PVP matchmaking queue).
package tournament

import (
	"context"
	"errors"
	"math/bits"
	"strconv"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
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
	gameCreator        GameCreator
	entryFee           int32
	minPlayersRequired uint32
	intervalSeconds    uint32
	beforeStartSeconds uint32
}

func newCoordinator(parent context.Context, gameCreator GameCreator, entryFee int32, minPlayersRequired uint32, intervalSeconds uint32, beforeStartSeconds uint32) *coordinator {
	ctx, cancel := context.WithCancel(parent)
	return &coordinator{
		ctx:                ctx,
		cancel:             cancel,
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
	unitDuration := time.Duration(tc.intervalSeconds) * time.Second
	//log.Debugw("tournament: tick", "unitDuration", unitDuration)

	if err := tc.ensureNextTournaments(now); err != nil {
		log.Errorw("tournament: ensure next tournaments", "err", err)
		return
	}

	//2. 待匹配players
	currentSlotTime := now.Truncate(unitDuration)
	tournamentToBegin, err := db.TournamentGetLatestRegistrationOpenWithSlot(currentSlotTime)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			//log.Debugw("tournament: no tournament to begin at current slot", "currentSlotTime", currentSlotTime)
			return
		}
		log.Errorw("tournament: get latest tournament to begin", "err", err)
		return
	}
	if err := tc.beginTournament(tournamentToBegin); err != nil { //这个逻辑待看。。。。。
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

	anyPlaying := false
	for _, matchID := range newMatchIDs {
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
		gameID, gerr := tc.gameCreator.CreateGameAndRun(players, types.GameTypeTournament, 0) //todo: if failed, how to handle?
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

func (tc *coordinator) onGameCompleted(gameInfo *dao.Game) error {
	if gameInfo.Status == proto.GameStatus_GAME_ABORTED {
		return nil
	}

	m, err := db.TournamentGetMatchByGameID(gameInfo.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if m.WinnerPlayerID != nil && *m.WinnerPlayerID != 0 {
		return nil
	}
	if m.WinnerTempAddress != nil && *m.WinnerTempAddress != "" {
		return nil
	}
	return nil
}

func pickTournamentWinner(_ *dao.TournamentMatch, _ *dao.Game) (uint, error) {
	return 0, errors.New("not implemented")
}
