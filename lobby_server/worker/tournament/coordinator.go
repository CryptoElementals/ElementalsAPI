// Package tournament implements tournament bracket scheduling and match starts (separate from PVP matchmaking queue).
package tournament

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CryptoElementals/common/bot_manager"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/snowflake"
	"github.com/CryptoElementals/common/timer"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const tickInterval = 500 * time.Millisecond
const recurringTickInterval = 3 * time.Second
const nextRoundStartDelay = 5 * time.Second
const disabledCreationLogInterval = 1 * time.Minute
const coordinatorTickEventType = "tournament:coordinator:tick"
const nextRoundStartEventType = "tournament:next-round-start"
const maxBotsPerTickPerTournament = 10

type coordinatorTickEvent struct{}

func (e *coordinatorTickEvent) EventType() string { return coordinatorTickEventType }
func (e *coordinatorTickEvent) Marshal() []byte   { return []byte("{}") }
func (e *coordinatorTickEvent) Unmarshal(_ []byte) error {
	return nil
}
func (e *coordinatorTickEvent) String() string { return coordinatorTickEventType }

type nextRoundStartEvent struct {
	MatchIDs    []uint `json:"match_ids"`
	TournamentID string `json:"tournament_id"`
	NextRound   uint32 `json:"next_round"`
}

func (e *nextRoundStartEvent) EventType() string { return nextRoundStartEventType }
func (e *nextRoundStartEvent) Marshal() []byte {
	b, _ := json.Marshal(e)
	return b
}
func (e *nextRoundStartEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}
func (e *nextRoundStartEvent) String() string {
	return fmt.Sprintf("%s(tournament_id=%s,next_round=%d,match_ids=%d)", nextRoundStartEventType, e.TournamentID, e.NextRound, len(e.MatchIDs))
}

// GameCreator starts tournament matches via RoomWorkerService.CreateGameAndRun.
type GameCreator interface {
	CreateGameAndRun(players []types.PlayerAddress, gameType proto.GameType, completedMatchID int64) (int64, error)
}

type coordinator struct {
	ctx                context.Context
	cancel             context.CancelFunc
	lobbyPublisher     pubsub.Publisher
	rosterPublisher    pubsub.Publisher
	botStore           *bot_manager.RedisStore
	joinTournamentFunc func(tournamentID string, req *proto.PlayerAddress) error
	gameCreator        GameCreator
	entryFee           int32
	minPlayersRequired uint32
	intervalSeconds    uint32
	beforeStartSeconds uint32

	botFreshness                time.Duration
	botFillWindow               time.Duration
	botFillInterval             time.Duration
	botFillLastJoinByTournament sync.Map // map[tournamentID]unixMilli

	tournamentCreationEnabled atomic.Bool
	lastDisabledLogUnixSec    atomic.Int64
}

func newCoordinator(parent context.Context, lobbyPublisher pubsub.Publisher, rosterPublisher pubsub.Publisher, botStore *bot_manager.RedisStore, gameCreator GameCreator, entryFee int32, minPlayersRequired uint32, intervalSeconds uint32, beforeStartSeconds uint32, botFillWindowSeconds uint32, botFillIntervalSeconds uint32, botFreshnessSec int64) *coordinator {
	ctx, cancel := context.WithCancel(parent)
	tc := &coordinator{
		ctx:                ctx,
		cancel:             cancel,
		lobbyPublisher:     lobbyPublisher,
		rosterPublisher:    rosterPublisher,
		botStore:           botStore,
		gameCreator:        gameCreator,
		entryFee:           entryFee,
		minPlayersRequired: minPlayersRequired,
		intervalSeconds:    intervalSeconds,
		beforeStartSeconds: beforeStartSeconds,
		botFreshness:       time.Duration(botFreshnessSec) * time.Second,
		botFillWindow:      time.Duration(botFillWindowSeconds) * time.Second,
		botFillInterval:    time.Duration(botFillIntervalSeconds) * time.Second,
	}
	tc.tournamentCreationEnabled.Store(true)
	return tc
}

func (tc *coordinator) start() {
	log.Debugw("tournament coordinator start")
	evt := &coordinatorTickEvent{}
	if err := timer.RegisterHandler(timer.ScopeLobby, evt, func(_ timer.TimerEvent) error {
		select {
		case <-tc.ctx.Done():
			return nil
		default:
		}
		tc.tick()
		return nil
	}); err != nil {
		log.Fatalw("tournament coordinator register timer handler failed", "err", err)
		return
	}
	if err := timer.RegisterTournamentRecurring(recurringTickInterval, evt); err != nil {
		log.Fatalw("tournament coordinator register recurring failed", "err", err)
		return
	}
	nextEvt := &nextRoundStartEvent{}
	if err := timer.RegisterHandler(timer.ScopeLobby, nextEvt, func(event timer.TimerEvent) error {
		select {
		case <-tc.ctx.Done():
			return nil
		default:
		}
		e, ok := event.(*nextRoundStartEvent)
		if !ok {
			return fmt.Errorf("unexpected timer event type %T for %s", event, nextRoundStartEventType)
		}
		tc.handleNextRoundStart(e.MatchIDs, e.TournamentID, e.NextRound)
		return nil
	}); err != nil {
		log.Fatalw("tournament coordinator register next-round handler failed", "err", err)
		return
	}
}

func (tc *coordinator) stop() {
	if err := timer.UnregisterTournamentRecurring(); err != nil {
		log.Errorw("tournament coordinator unregister recurring failed", "err", err)
	}
	tc.cancel()
}

// 1. 超过整数倍单元时间(支持配置，如1小时, 10分钟)，创建下一个tournament
func (tc *coordinator) tick() {
	now := time.Now().UTC()

	if err := tc.fillRegistrationOpenTournamentsWithBots(now); err != nil {
		log.Errorw("tournament: fill bots before begin failed", "err", err)
	}

	//1. 之前的比赛待匹配players
	// Grace window: if scheduler runs a little late (e.g. process restart at +1s), still begin this slot.
	tournamentToBegin, err := db.TournamentGetLatestRegistrationOpenWithinStartGrace(now, 10*time.Second)
	if err != nil {
		//if errors.Is(err, gorm.ErrRecordNotFound) {
		//	log.Debugw("tournament: no tournament to begin in grace window", "now", now)
		//} else {
		//	log.Errorw("tournament: get latest tournament to begin", "err", err)
		//}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorw("tournament: get latest tournament to begin", "err", err)
		}
	} else {
		if err := tc.beginTournament(tournamentToBegin); err != nil {
			log.Errorw("tournament: begin", "tournament_id", tournamentToBegin.ID, "err", err)
		}
	}

	//2. 创建下一个tournament
	if tc.tournamentCreationEnabled.Load() {
		if err := tc.ensureNextTournaments(now); err != nil {
			log.Errorw("tournament: ensure next tournaments", "err", err)
			return
		}
	} else {
		tc.logTournamentCreationDisabled(now)
	}

}

func (tc *coordinator) setTournamentCreationEnabled(enabled bool) {
	tc.tournamentCreationEnabled.Store(enabled)
	if enabled {
		tc.lastDisabledLogUnixSec.Store(0)
		return
	}
	log.Infow("tournament: creation disabled by control")
}

func (tc *coordinator) isTournamentCreationEnabled() bool {
	return tc.tournamentCreationEnabled.Load()
}

func (tc *coordinator) logTournamentCreationDisabled(now time.Time) {
	nowSec := now.Unix()
	lastSec := tc.lastDisabledLogUnixSec.Load()
	if lastSec != 0 && nowSec-lastSec < int64(disabledCreationLogInterval/time.Second) {
		return
	}
	tc.lastDisabledLogUnixSec.Store(nowSec)
	log.Infow("tournament: creation is disabled")
}

func (tc *coordinator) fillRegistrationOpenTournamentsWithBots(now time.Time) error {
	if tc.botStore == nil {
		return nil
	}
	if tc.botFillWindow <= 0 || tc.botFillInterval <= 0 {
		return nil
	}
	rows, err := db.TournamentListRegistrationOpenInBotFillWindow(now, tc.botFillWindow)
	if err != nil {
		return err
	}
	for _, t := range rows {
		if err := tc.fillTournamentWithBotsIfNeeded(now, t.TournamentID); err != nil {
			log.Warnw("tournament: fill bots for tournament failed", "tournament_id", t.TournamentID, "err", err)
		}
	}
	return nil
}

func (tc *coordinator) fillTournamentWithBotsIfNeeded(now time.Time, tournamentID string) error {
	if tournamentID == "" {
		return nil
	}
	need := 0
	var remainingToDeadline time.Duration
	fillProgress := 1.0 // 0=start of fill window, 1=deadline
	err := db.Get().Transaction(func(tx *gorm.DB) error {
		var t dao.Tournament
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tournament_id = ?", tournamentID).
			First(&t).Error; err != nil {
			return err
		}
		if t.Status != dao.TournamentStatusRegistrationOpen {
			tc.botFillLastJoinByTournament.Delete(tournamentID)
			return nil
		}
		fillStart := t.RegistrationDeadline.Add(-tc.botFillWindow)
		if now.Before(fillStart) || !now.Before(t.RegistrationDeadline) {
			tc.botFillLastJoinByTournament.Delete(tournamentID)
			return nil
		}
		queued, err := db.TournamentListParticipantsByStatus(tx, t.TournamentID, dao.TournamentParticipantStatusQueued)
		if err != nil {
			return err
		}
		target := int(tc.minPlayersRequired)
		if target%2 != 0 {
			target++
		}
		if len(queued) >= target {
			tc.botFillLastJoinByTournament.Delete(tournamentID)
			return nil
		}
		need = target - len(queued)
		remainingToDeadline = t.RegistrationDeadline.Sub(now)
		window := t.RegistrationDeadline.Sub(fillStart)
		if window > 0 {
			elapsed := now.Sub(fillStart)
			if elapsed < 0 {
				elapsed = 0
			}
			if elapsed > window {
				elapsed = window
			}
			fillProgress = float64(elapsed) / float64(window)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if need <= 0 {
		return nil
	}
	effectiveInterval := tc.predictiveBotFillInterval(need, remainingToDeadline)
	if !tc.shouldAddBotNow(tournamentID, now, effectiveInterval) {
		return nil
	}

	if tc.joinTournamentFunc == nil {
		return fmt.Errorf("nil tournament join function")
	}
	burst := minInt(need, maxBotsPerTickPerTournament)
	// Adaptive burst:
	// - when there is enough time, add fewer bots (1~4);
	// - when behind schedule, raise burst up to maxBotsPerTickPerTournament.
	if effectiveInterval > 0 {
		adaptive := int((recurringTickInterval + effectiveInterval - 1) / effectiveInterval) // ceil(period/effective)
		if adaptive < 1 {
			adaptive = 1
		}
		if adaptive < burst {
			burst = adaptive
		}
	}
	// Time-progress cap (front-slow, back-fast):
	// prefer real players early, then ramp up bot burst near deadline.
	stageCap := 1 + int(fillProgress*4) // 1..5
	if stageCap < 1 {
		stageCap = 1
	}
	if stageCap > maxBotsPerTickPerTournament {
		stageCap = maxBotsPerTickPerTournament
	}
	if stageCap < burst {
		burst = stageCap
	}
	for i := 0; i < burst; i++ {
		bot, err := tc.botStore.PopFreshIdleBotForMatch(now.UnixMilli(), tc.botFreshness.Milliseconds())
		if err != nil {
			return err
		}
		if bot == nil {
			return nil
		}
		if err := tc.joinTournamentFunc(tournamentID, &proto.PlayerAddress{
			Id:               bot.Id,
			TemporaryAddress: bot.TemporaryAddress,
		}); err != nil {
			_, rerr := tc.botStore.ReleaseInGameBot(tc.ctx, *bot)
			if rerr != nil {
				log.Warnw("tournament: release bot after failed join failed", "bot", bot.String(), "err", rerr)
			}
			log.Warnw("tournament: bot join tournament failed", "tournament_id", tournamentID, "bot", bot.String(), "err", err)
			return nil
		}
		log.Debugw("tournament: bot join tournament success",
			"tournament_id", tournamentID,
			"bot", bot.String(),
			"need_before_join", need,
			"join_index_in_tick", i+1,
			"join_burst_limit", burst,
			"fill_progress", fillProgress,
			"remaining_to_deadline_ms", remainingToDeadline.Milliseconds(),
			"effective_interval_ms", effectiveInterval.Milliseconds())
	}
	return nil
}

func (tc *coordinator) shouldAddBotNow(tournamentID string, now time.Time, interval time.Duration) bool {
	if interval <= 0 {
		interval = time.Second
	}
	nowMs := now.UnixMilli()
	if v, ok := tc.botFillLastJoinByTournament.Load(tournamentID); ok {
		lastMs, _ := v.(int64)
		if nowMs-lastMs < interval.Milliseconds() {
			return false
		}
	}
	tc.botFillLastJoinByTournament.Store(tournamentID, nowMs)
	return true
}

// predictiveBotFillInterval computes a faster cadence when the current shortage
// cannot be covered in time with the base interval.
func (tc *coordinator) predictiveBotFillInterval(need int, remainingToDeadline time.Duration) time.Duration {
	base := tc.botFillInterval
	if base <= 0 {
		base = time.Second
	}
	if need <= 0 || remainingToDeadline <= 0 {
		return base
	}
	// Last-slot protection: when only 1 seat is missing and deadline is very near,
	// tighten cadence to tick interval so we don't miss the final fill chance.
	if need == 1 && remainingToDeadline < 5*time.Second {
		if tickInterval < base {
			return tickInterval
		}
		return base
	}

	// Average pace needed to fill all missing slots before deadline.
	target := remainingToDeadline / time.Duration(need)
	if target < time.Second {
		target = time.Second
	}
	if target < base {
		return target
	}
	return base
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
		TournamentID:         strconv.FormatInt(snowflake.GenerateID(), 10),
		Status:               dao.TournamentStatusRegistrationOpen,
		ScheduledStartAt:     at,
		ScheduledEndDeadline: at.Add(time.Duration(tc.intervalSeconds) * time.Second),
		RegistrationDeadline: at.Add(-time.Duration(tc.beforeStartSeconds) * time.Second),
		EntryFee:             tc.entryFee,
	}
	if err := db.TournamentCreate(t); err != nil {
		return err
	}
	if err := tc.publishTournamentRosterUpdate(t.TournamentID); err != nil {
		return err
	}
	return nil
}

func (tc *coordinator) beginTournament(t *dao.Tournament) error {
	var newMatchIDs []uint
	var createdRound bool
	var releaseCandidates []types.PlayerAddress

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
				if err := db.RefundUserTokenForTournamentEntryTx(tx, p.PlayerID, cur.EntryFee); err != nil {
					return err
				}
				if err := db.RecordTournamentEntryLedgerTx(
					tx,
					cur.TournamentID,
					p.PlayerID,
					p.TempAddress,
					cur.EntryFee,
					dao.TournamentEntryLedgerDirectionEntryRefund,
					string(dao.TournamentParticipantStatusKickedNotEnough),
				); err != nil {
					return err
				}
				releaseCandidates = append(releaseCandidates, types.PlayerAddress{Id: p.PlayerID, TemporaryAddress: p.TempAddress})
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
				if err := db.RefundUserTokenForTournamentEntryTx(tx, p.PlayerID, cur.EntryFee); err != nil {
					return err
				}
				if err := db.RecordTournamentEntryLedgerTx(
					tx,
					cur.TournamentID,
					p.PlayerID,
					p.TempAddress,
					cur.EntryFee,
					dao.TournamentEntryLedgerDirectionEntryRefund,
					string(dao.TournamentParticipantStatusKickedOverflow),
				); err != nil {
					return err
				}
				releaseCandidates = append(releaseCandidates, types.PlayerAddress{Id: p.PlayerID, TemporaryAddress: p.TempAddress})
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
	tc.releaseBotsIfNeeded(tc.filterBotsForRelease(releaseCandidates))

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
		gameID, gerr := tc.gameCreator.CreateGameAndRun(players, proto.GameType_TOURNAMENT, 0)
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
	winnerRank                     uint32
	loserPID                       int64
	loserTemp                      string
	loserRank                      uint32
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
	rankedWinner := &proto.RankedPlayer{
		Address: winner.ToProto(),
		Rank:    o.winnerRank,
	}
	rankedLoser := &proto.RankedPlayer{
		Address: loser.ToProto(),
		Rank:    o.loserRank,
	}
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
				Winner:                         rankedWinner,
				Loser:                          rankedLoser,
				TournamentFinished:             o.tournamentFinished,
				RoundFinished:                  o.roundFinished,
				CumulativeBattleReward:         o.cumulativeBattleReward,
				WinnerCumulationIfNextMatchWin: o.winnerCumulationIfNextMatchWin,
				NextRoundNo:                    o.nextRoundNo,
			},
		},
	}
	if err := pubsub.Publish(tc.ctx, tc.lobbyPublisher, evt); err != nil {
		log.Errorw("tournament: publish match outcome failed", "game_id", o.gameID, "err", err)
	}
}

func (tc *coordinator) publishTournamentRosterUpdate(tournamentID string) error {
	if tc == nil {
		return fmt.Errorf("nil coordinator")
	}
	if tc.rosterPublisher == nil {
		return fmt.Errorf("nil roster publisher")
	}
	if tournamentID == "" {
		return fmt.Errorf("empty tournament id")
	}
	evt := &proto.Event{
		Type: proto.EventType_TYPE_TOURNAMENT_ROSTER_UPDATE,
		Event: &proto.Event_TournamentRosterUpdate{
			TournamentRosterUpdate: &proto.TournamentRosterUpdate{TournamentID: tournamentID},
		},
	}
	if err := pubsub.Publish(tc.ctx, tc.rosterPublisher, evt); err != nil {
		log.Errorw("tournament: publish roster update failed", "tournament_id", tournamentID, "err", err)
		return err
	}
	return nil
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
		releaseCandidates []types.PlayerAddress
		postNewIDs        []uint
		postNextRound     uint32
		postTournID       string
		publishOutcome    *tournamentMatchOutcomeToPublish
		cumReward         *proto.BattleReward
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
		releaseCandidates = append(releaseCandidates, types.PlayerAddress{
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
		// Pay out when a player is eliminated.
		if rerr := db.TournamentApplyMatchRewardsTx(tx, locked.TournamentID, locked.RoundNo, locked.MatchNo,
			winPID, winTempNorm); rerr != nil {
			return rerr
		}
		// Loser leaves tournament now; settle loser's latest snapshot reward once.
		if rerr := db.TournamentSettlePlayerRewardToWalletTx(tx, locked.TournamentID, loserPID, loserTempNorm); rerr != nil {
			return rerr
		}

		roundMatches, err := db.TournamentListMatchesForRound(tx, locked.TournamentID, locked.RoundNo)
		if err != nil {
			return err
		}
		currentRoundPlayers := uint32(len(roundMatches) * 2)
		if currentRoundPlayers == 0 {
			return fmt.Errorf("tournament %s round %d has no players", locked.TournamentID, locked.RoundNo)
		}
		winnerRank := currentRoundPlayers / 2
		if winnerRank == 0 {
			winnerRank = 1
		}
		loserRank := currentRoundPlayers

		var cerr error
		cumReward, cerr = db.TournamentCumulativeBattleRewardProtoTx(tx, locked.TournamentID,
			winPID, winTempNorm, loserPID, loserTempNorm)
		if cerr != nil {
			return cerr
		}

		for _, rm := range roundMatches {
			if rm.Status != dao.TournamentMatchStatusCompleted { // one match not completed yet, then round not completed
				nextRN := locked.RoundNo + 1
				winNext, werr := winnerCumulationReward(tx, locked.TournamentID, winPID, winTempNorm, nextRN)
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
					winnerRank:                     winnerRank,
					loserPID:                       loserPID,
					loserTemp:                      loserTempNorm,
					loserRank:                      loserRank,
					tournamentFinished:             false,
					roundFinished:                  false,
					cumulativeBattleReward:         cumReward,
					winnerCumulationIfNextMatchWin: winNext,
					nextRoundNo:                    &nextRN,
				}
				return nil // return to wait for the match to be finished
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
			// Champion gets paid when tournament is finalized.
			if rerr := db.TournamentSettlePlayerRewardToWalletTx(tx, locked.TournamentID, winPID, winTempNorm); rerr != nil {
				return rerr
			}
			cumReward, cerr = db.TournamentCumulativeBattleRewardProtoTx(tx, locked.TournamentID,
				winPID, winTempNorm, loserPID, loserTempNorm)
			if cerr != nil {
				return cerr
			}
			releaseCandidates = append(releaseCandidates, types.PlayerAddress{
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
				winnerRank:             winnerRank,
				loserPID:               loserPID,
				loserTemp:              loserTempNorm,
				loserRank:              loserRank,
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
		winNext, werr := winnerCumulationReward(tx, locked.TournamentID, winPID, winTempNorm, nextNo)
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
			winnerRank:                     winnerRank,
			loserPID:                       loserPID,
			loserTemp:                      loserTempNorm,
			loserRank:                      loserRank,
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
	tc.releaseBotsIfNeeded(tc.filterBotsForRelease(releaseCandidates))

	if publishOutcome != nil {
		tc.publishTournamentMatchOutcome(publishOutcome)
	}

	if len(postNewIDs) > 0 {
		if err := tc.scheduleNextRoundStart(postNewIDs, postTournID, postNextRound); err != nil {
			log.Errorw("tournament: schedule next round start failed", "tournament_id", postTournID, "next_round", postNextRound, "err", err)
			return err
		}
	}

	return nil
}

func (tc *coordinator) scheduleNextRoundStart(matchIDs []uint, tournID string, nextRound uint32) error {
	if len(matchIDs) == 0 || tournID == "" || nextRound == 0 {
		return nil
	}
	evt := &nextRoundStartEvent{
		MatchIDs:    matchIDs,
		TournamentID: tournID,
		NextRound:   nextRound,
	}
	return timer.ProcessIn(timer.ScopeLobby, nextRoundStartDelay, evt, false)
}

func (tc *coordinator) handleNextRoundStart(matchIDs []uint, tournID string, nextRound uint32) {
	anyPlaying := tc.startGamesForNewMatches(matchIDs)
	if !anyPlaying || tournID == "" {
		return
	}
	if r, err := db.TournamentGetRound(db.Get(), tournID, nextRound); err == nil {
		r.Status = dao.TournamentRoundStatusPlaying
		_ = db.TournamentSaveRound(db.Get(), r)
	}
}

func (tc *coordinator) filterBotsForRelease(addrs []types.PlayerAddress) []types.PlayerAddress {
	if tc == nil || tc.botStore == nil || len(addrs) == 0 {
		return nil
	}
	bots := make([]types.PlayerAddress, 0, len(addrs))
	for _, addr := range addrs {
		isBot, err := tc.botStore.IsBot(addr)
		if err != nil {
			log.Warnw("tournament: bot check before collect failed", "player", addr.String(), "err", err)
			continue
		}
		if isBot {
			bots = append(bots, addr)
		}
	}
	return bots
}

func (tc *coordinator) releaseBotsIfNeeded(addrs []types.PlayerAddress) {
	if tc == nil || tc.botStore == nil || len(addrs) == 0 {
		return
	}
	for _, addr := range addrs {
		ok, err := tc.botStore.ReleaseInGameBot(tc.ctx, addr)
		if err != nil {
			log.Warnw("tournament: release bot failed", "player", addr.String(), "err", err)
			continue
		}
		if ok {
			log.Debugw("tournament: bot released", "player", addr.String())
		}
	}
}

// winnerCumulationReward returns winner cumulative token/point totals if they also win their next match (current sum + one match-win bonus).
func winnerCumulationReward(tx *gorm.DB, tournamentID string, winPID int64, winTempNorm string, roundNo uint32) (*proto.PlayerReward, error) {

	totalParticipants, err := db.TournamentCountParticipantsForPool(tournamentID)
	if err != nil {
		return nil, err
	}

	t, p, err := db.TournamentRoundReward(tx, int32(totalParticipants), roundNo)
	if err != nil {
		return nil, err
	}
	return &proto.PlayerReward{
		PlayerId:         winPID,
		TemporaryAddress: winTempNorm,
		TokenChange:      t,
		PointChange:      p,
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
			log.Errorf("tournament: winner from player result infos failed: %v, %v, %v, judge tournament %v winner by comparing player ids: %v, %v",
				gr.GameID, gr.GameResultType, gr.PlayerResultInfos, m.TournamentID, p1id, p2id)
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
