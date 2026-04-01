// Package turnament implements tournament bracket scheduling and match starts (separate from PVP matchmaking queue).
package turnament

import (
	"context"
	"errors"
	"math/bits"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const tickInterval = 2 * time.Second

// GameCreator starts tournament matches the same way as PVP (HandleGameMatchedEvent with GameTypeTournament).
type GameCreator interface {
	HandleGameMatchedEvent(evt *types.GameMatchedEvent) (uint, error)
}

type coordinator struct {
	ctx         context.Context
	cancel      context.CancelFunc
	gameCreator GameCreator
}

func newCoordinator(parent context.Context, gameCreator GameCreator) *coordinator {
	ctx, cancel := context.WithCancel(parent)
	return &coordinator{
		ctx:         ctx,
		cancel:      cancel,
		gameCreator: gameCreator,
	}
}

func (tc *coordinator) start() {
	go tc.loop()
}

func (tc *coordinator) stop() {
	tc.cancel()
}

func (tc *coordinator) loop() {
	tick := time.NewTicker(tickInterval)
	defer tick.Stop()
	for {
		select {
		case <-tc.ctx.Done():
			return
		case <-tick.C:
			tc.tick()
		}
	}
}

func (tc *coordinator) tick() {
	schedules, err := db.TournamentListEnabledSchedules()
	if err != nil {
		log.Errorw("tournament: list schedules", "err", err)
		return
	}
	for i := range schedules {
		s := &schedules[i]
		if dao.TournamentBracketCapacity(s.BracketExponent) == 0 {
			log.Warnw("tournament: invalid bracket exponent on schedule", "schedule_id", s.ID, "exp", s.BracketExponent)
			continue
		}
		if err := tc.ensureOpenTournament(s); err != nil {
			log.Errorw("tournament: ensure open", "schedule_id", s.ID, "err", err)
		}
	}
	toLock, err := db.TournamentListOpenPastScheduled(time.Now())
	if err != nil {
		log.Errorw("tournament: list to lock", "err", err)
		return
	}
	for i := range toLock {
		t := &toLock[i]
		if err := tc.lockTournament(t); err != nil {
			log.Errorw("tournament: lock", "tournament_id", t.ID, "err", err)
		}
	}
}

func scheduleInstanceStart(s *dao.TournamentSchedule, instanceIndex uint32) time.Time {
	step := time.Duration(s.IntervalSeconds) * time.Second * time.Duration(instanceIndex)
	return s.FirstStartAt.Add(step)
}

func (tc *coordinator) ensureOpenTournament(sched *dao.TournamentSchedule) error {
	latest, err := db.TournamentGetLatestByScheduleID(sched.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		capacity := dao.TournamentBracketCapacity(sched.BracketExponent)
		t := &dao.Tournament{
			TournamentScheduleID: sched.ID,
			InstanceIndex:        0,
			ScheduledStartAt:     scheduleInstanceStart(sched, 0),
			BracketExponent:      sched.BracketExponent,
			MaxParticipants:      capacity,
			BracketSlots:         0,
			Status:               dao.TournamentStatusOpen,
		}
		return db.TournamentCreate(t)
	}
	if err != nil {
		return err
	}
	switch latest.Status {
	case dao.TournamentStatusOpen, dao.TournamentStatusLocked, dao.TournamentStatusInProgress:
		return nil
	case dao.TournamentStatusCompleted, dao.TournamentStatusCancelled:
		next := &dao.Tournament{
			TournamentScheduleID: sched.ID,
			InstanceIndex:        latest.InstanceIndex + 1,
			ScheduledStartAt:     scheduleInstanceStart(sched, latest.InstanceIndex+1),
			BracketExponent:      sched.BracketExponent,
			MaxParticipants:      dao.TournamentBracketCapacity(sched.BracketExponent),
			BracketSlots:         0,
			Status:               dao.TournamentStatusOpen,
		}
		return db.TournamentCreate(next)
	default:
		return nil
	}
}

func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func finalRoundNumber(bracketSlots uint32) uint32 {
	if bracketSlots <= 1 {
		return 0
	}
	return uint32(bits.Len32(bracketSlots) - 1)
}

func (tc *coordinator) lockTournament(t *dao.Tournament) error {
	var afterTournamentID uint
	var afterRound uint32

	err := db.Get().Transaction(func(tx *gorm.DB) error {
		var cur dao.Tournament
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cur, t.ID).Error; err != nil {
			return err
		}
		if cur.Status != dao.TournamentStatusOpen {
			return nil
		}

		entries, err := db.TournamentListQueuedEntries(tx, cur.ID)
		if err != nil {
			return err
		}

		now := time.Now()
		cur.LockedAt = &now

		if len(entries) == 0 {
			cur.Status = dao.TournamentStatusCancelled
			if err := tx.Save(&cur).Error; err != nil {
				return err
			}
			return nil
		}

		if len(entries) == 1 {
			cur.BracketSlots = 1
			cur.Status = dao.TournamentStatusCompleted
			cur.CompletedAt = &now
			if err := tx.Save(&cur).Error; err != nil {
				return err
			}
			e := &entries[0]
			e.Status = dao.TournamentEntryStatusWinner
			e.SeedPosition = 0
			if err := tx.Save(e).Error; err != nil {
				return err
			}
			return nil
		}

		capN := int(cur.MaxParticipants)
		inBracket := entries
		if len(entries) > capN {
			for i := capN; i < len(entries); i++ {
				ent := &entries[i]
				ent.Status = dao.TournamentEntryStatusKickedOverflow
				ent.SeedPosition = 0
				if err := tx.Save(ent).Error; err != nil {
					return err
				}
			}
			inBracket = entries[:capN]
		}

		k := len(inBracket)
		Neff := nextPow2(k)
		if Neff > capN {
			Neff = capN
		}
		NeffU := uint32(Neff)

		cur.BracketSlots = NeffU
		cur.Status = dao.TournamentStatusInProgress
		if err := tx.Save(&cur).Error; err != nil {
			return err
		}

		slots := make([]*dao.TournamentEntry, Neff)
		for i := 0; i < k; i++ {
			e := &inBracket[i]
			e.Status = dao.TournamentEntryStatusInBracket
			e.SeedPosition = uint32(i)
			if err := tx.Save(e).Error; err != nil {
				return err
			}
			slots[i] = e
		}

		if err := tc.insertRoundMatches(tx, &cur, 1, slots); err != nil {
			return err
		}
		if err := tc.applyByesInRoundTx(tx, cur.ID, 1); err != nil {
			return err
		}

		afterTournamentID = cur.ID
		afterRound = 1
		return nil
	})
	if err != nil {
		return err
	}
	if afterTournamentID != 0 {
		tc.afterRoundReady(afterTournamentID, afterRound)
	}
	return nil
}

func (tc *coordinator) insertRoundMatches(tx *gorm.DB, t *dao.Tournament, roundNumber uint32, slots []*dao.TournamentEntry) error {
	n := len(slots)
	if n%2 != 0 {
		return errors.New("tournament: odd bracket slot count")
	}
	for m := 0; m < n/2; m++ {
		match := &dao.TournamentMatch{
			TournamentID: t.ID,
			RoundNumber:  roundNumber,
			MatchIndex:   uint32(m),
		}
		if slots[2*m] != nil {
			id := slots[2*m].ID
			match.EntryAID = &id
		}
		if slots[2*m+1] != nil {
			id := slots[2*m+1].ID
			match.EntryBID = &id
		}
		if err := db.TournamentCreateMatch(tx, match); err != nil {
			return err
		}
	}
	return nil
}

func (tc *coordinator) applyByesInRoundTx(tx *gorm.DB, tournamentID uint, round uint32) error {
	matches, err := db.TournamentListMatchesForRound(tx, tournamentID, round)
	if err != nil {
		return err
	}
	for i := range matches {
		m := &matches[i]
		if m.WinnerEntryID != nil {
			continue
		}
		hasA := m.EntryAID != nil
		hasB := m.EntryBID != nil
		if hasA && !hasB {
			m.WinnerEntryID = m.EntryAID
			if err := db.TournamentSaveMatch(tx, m); err != nil {
				return err
			}
		} else if !hasA && hasB {
			m.WinnerEntryID = m.EntryBID
			if err := db.TournamentSaveMatch(tx, m); err != nil {
				return err
			}
		}
	}
	return nil
}

func (tc *coordinator) afterRoundReady(tournamentID uint, round uint32) {
	tc.startPendingGamesForRound(tournamentID, round)
	if err := tc.tryAdvanceAfterRound(tournamentID, round); err != nil {
		log.Errorw("tournament: advance after round", "tournament_id", tournamentID, "round", round, "err", err)
	}
}

func (tc *coordinator) startPendingGamesForRound(tournamentID uint, round uint32) {
	matches, err := db.TournamentListMatchesForRound(db.Get(), tournamentID, round)
	if err != nil {
		log.Errorw("tournament: list matches for games", "err", err)
		return
	}
	for i := range matches {
		m := &matches[i]
		if m.GameID != nil || m.WinnerEntryID != nil {
			continue
		}
		if m.EntryAID == nil || m.EntryBID == nil {
			continue
		}
		if err := tc.startMatchGame(m); err != nil {
			log.Errorw("tournament: start match game", "match_id", m.ID, "err", err)
		}
	}
}

func (tc *coordinator) startMatchGame(m *dao.TournamentMatch) error {
	var cur dao.TournamentMatch
	if err := db.Get().First(&cur, m.ID).Error; err != nil {
		return err
	}
	if cur.GameID != nil || cur.WinnerEntryID != nil {
		return nil
	}
	if cur.EntryAID == nil || cur.EntryBID == nil {
		return nil
	}

	ea, err := db.TournamentLoadEntry(db.Get(), *cur.EntryAID)
	if err != nil {
		return err
	}
	eb, err := db.TournamentLoadEntry(db.Get(), *cur.EntryBID)
	if err != nil {
		return err
	}

	pa := types.NewPlayerAddress(ea.PlayerId, ea.TemporaryAddress)
	pb := types.NewPlayerAddress(eb.PlayerId, eb.TemporaryAddress)
	evt := &types.GameMatchedEvent{
		Players:  []types.PlayerAddress{*pa, *pb},
		GameType: types.GameTypeTournament,
	}
	gid, err := tc.gameCreator.HandleGameMatchedEvent(evt)
	if err != nil {
		return err
	}
	gidU := uint(gid)
	cur.GameID = &gidU
	if err := db.TournamentSaveMatch(db.Get(), &cur); err != nil {
		return err
	}
	for _, p := range evt.Players {
		if err := db.SetLockedTokenGameID(tc.ctx, p.Id, p.TemporaryAddress, gid); err != nil {
			log.Errorw("tournament: SetLockedTokenGameID", "player", p.String(), "err", err)
			return err
		}
	}
	return nil
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
	if m.WinnerEntryID != nil {
		return nil
	}

	mFull, err := db.TournamentLoadMatchByID(db.Get(), m.ID)
	if err != nil {
		return err
	}
	winnerID, err := pickTournamentWinner(mFull, gameInfo)
	if err != nil {
		log.Errorw("tournament: pick winner", "match_id", m.ID, "err", err)
		return err
	}
	mFull.WinnerEntryID = &winnerID
	if err := db.TournamentSaveMatch(db.Get(), mFull); err != nil {
		return err
	}

	var loserID uint
	if mFull.EntryAID != nil && *mFull.EntryAID != winnerID {
		loserID = *mFull.EntryAID
	} else if mFull.EntryBID != nil && *mFull.EntryBID != winnerID {
		loserID = *mFull.EntryBID
	}
	if loserID != 0 {
		if le, err := db.TournamentLoadEntry(db.Get(), loserID); err == nil {
			le.Status = dao.TournamentEntryStatusEliminated
			le.EliminatedRound = mFull.RoundNumber
			_ = db.TournamentSaveEntry(db.Get(), le)
		}
	}

	return tc.tryAdvanceAfterRound(mFull.TournamentID, mFull.RoundNumber)
}

func pickTournamentWinner(m *dao.TournamentMatch, gameInfo *dao.Game) (uint, error) {
	gr := gameInfo.GameResult
	if gr == nil {
		return 0, errors.New("tournament: missing game result")
	}
	if m.EntryA == nil || m.EntryB == nil {
		return 0, errors.New("tournament: match entries not preloaded")
	}

	if gr.GameResultType == proto.GameResultType_GAME_TIE {
		if m.EntryA.JoinSequence <= m.EntryB.JoinSequence {
			return m.EntryA.ID, nil
		}
		return m.EntryB.ID, nil
	}

	wpa := strings.EqualFold(m.EntryA.TemporaryAddress, gr.WinnerTemporaryAddress) && m.EntryA.PlayerId == gr.WinnerPlayerId
	wpb := strings.EqualFold(m.EntryB.TemporaryAddress, gr.WinnerTemporaryAddress) && m.EntryB.PlayerId == gr.WinnerPlayerId
	if wpa {
		return m.EntryA.ID, nil
	}
	if wpb {
		return m.EntryB.ID, nil
	}

	if m.EntryA.JoinSequence <= m.EntryB.JoinSequence {
		return m.EntryA.ID, nil
	}
	return m.EntryB.ID, nil
}

func roundAllMatchesHaveWinner(tournamentID uint, round uint32) (bool, error) {
	matches, err := db.TournamentListMatchesForRound(db.Get(), tournamentID, round)
	if err != nil {
		return false, err
	}
	if len(matches) == 0 {
		return false, nil
	}
	for i := range matches {
		if matches[i].WinnerEntryID == nil {
			return false, nil
		}
	}
	return true, nil
}

func (tc *coordinator) tryAdvanceAfterRound(tournamentID uint, completedRound uint32) error {
	ok, err := roundAllMatchesHaveWinner(tournamentID, completedRound)
	if err != nil || !ok {
		return err
	}

	var tour dao.Tournament
	if err := db.Get().First(&tour, tournamentID).Error; err != nil {
		return err
	}

	lastRound := finalRoundNumber(tour.BracketSlots)
	if completedRound == lastRound {
		matches, err := db.TournamentListMatchesForRound(db.Get(), tournamentID, completedRound)
		if err != nil {
			return err
		}
		if len(matches) != 1 || matches[0].WinnerEntryID == nil {
			return errors.New("tournament: invalid final round")
		}
		wid := *matches[0].WinnerEntryID
		if w, err := db.TournamentLoadEntry(db.Get(), wid); err == nil {
			w.Status = dao.TournamentEntryStatusWinner
			_ = db.TournamentSaveEntry(db.Get(), w)
		}
		now := time.Now()
		tour.Status = dao.TournamentStatusCompleted
		tour.CompletedAt = &now
		if err := db.TournamentSave(&tour); err != nil {
			return err
		}
		if sched, err := db.TournamentGetSchedule(tour.TournamentScheduleID); err == nil {
			_ = tc.ensureOpenTournament(sched)
		}
		return nil
	}

	nextRound := completedRound + 1
	matches, err := db.TournamentListMatchesForRound(db.Get(), tournamentID, completedRound)
	if err != nil {
		return err
	}
	slots := make([]*dao.TournamentEntry, len(matches))
	for i := range matches {
		if matches[i].WinnerEntryID == nil {
			return errors.New("tournament: missing winner")
		}
		e, err := db.TournamentLoadEntry(db.Get(), *matches[i].WinnerEntryID)
		if err != nil {
			return err
		}
		slots[i] = e
	}

	if err := db.Get().Transaction(func(tx *gorm.DB) error {
		if err := tc.insertRoundMatches(tx, &tour, nextRound, slots); err != nil {
			return err
		}
		return tc.applyByesInRoundTx(tx, tournamentID, nextRound)
	}); err != nil {
		return err
	}

	tc.afterRoundReady(tournamentID, nextRound)
	return nil
}
