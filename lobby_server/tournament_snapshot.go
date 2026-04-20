package lobbyserver

import (
	"context"
	"errors"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// GetLatestRegistrationOpenTournamentSnapshot returns the latest registration-open tournament with future deadline
// plus pool and player registration. Non-open tournaments are intentionally not returned.
func (s *GRPCServices) GetLatestRegistrationOpenTournamentSnapshot(ctx context.Context, req *proto.PlayerAddress) (*proto.GetLatestRegistrationOpenTournamentSnapshotResponse, error) {
	_ = ctx
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil player address")
	}
	now := time.Now().UTC()
	t, err := db.TournamentGetLatestRegistrationOpenBeforeStart(now)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &proto.GetLatestRegistrationOpenTournamentSnapshotResponse{HasTournament: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "tournament: %v", err)
	}

	return buildTournamentSnapshotResponse(t, req, now)
}

// GetLatestInProgressTournamentSnapshot returns the latest in-progress tournament snapshot.
func (s *GRPCServices) GetLatestInProgressTournamentSnapshot(ctx context.Context, req *proto.PlayerAddress) (*proto.GetLatestRegistrationOpenTournamentSnapshotResponse, error) {
	_ = ctx
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil player address")
	}
	now := time.Now().UTC()
	t, err := db.TournamentGetLatestInProgress()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &proto.GetLatestRegistrationOpenTournamentSnapshotResponse{HasTournament: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "tournament: %v", err)
	}

	return buildTournamentSnapshotResponse(t, req, now)
}

func buildTournamentSnapshotResponse(t *dao.Tournament, req *proto.PlayerAddress, now time.Time) (*proto.GetLatestRegistrationOpenTournamentSnapshotResponse, error) {
	count, err := db.TournamentCountParticipantsForPool(t.TournamentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "participant count: %v", err)
	}

	entryFee := t.EntryFee
	pool := int64(entryFee) * count

	var playerReg bool
	var partStatus string
	p, perr := db.TournamentGetParticipantByPlayer(t.TournamentID, req.Id, req.TemporaryAddress)
	if perr == nil {
		playerReg = true
		partStatus = string(p.Status)
	} else if !errors.Is(perr, gorm.ErrRecordNotFound) {
		return nil, status.Errorf(codes.Internal, "participant lookup: %v", perr)
	}

	secs := int64(t.ScheduledStartAt.Sub(now).Seconds())
	if secs < 0 {
		secs = 0
	}
	topFourRewardTokens, err := topFourRewardTokensFromTierTable(count)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "tier reward config: %v", err)
	}
	mapRoundConfig, err := readRoundConfigFromTierTable(count)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "round config: %v", err)
	}
	mapRoundConfigProto := roundConfigMapToProto(mapRoundConfig)

	return &proto.GetLatestRegistrationOpenTournamentSnapshotResponse{
		HasTournament:               true,
		TournamentID:                t.TournamentID,
		TournamentStatus:            string(t.Status),
		ScheduledStartUnixSec:       t.ScheduledStartAt.Unix(),
		RegistrationDeadlineUnixSec: t.RegistrationDeadline.Unix(),
		EntryFee:                    entryFee,
		ParticipantCount:            count,
		PlayerRegistered:            playerReg,
		PlayerParticipantStatus:     partStatus,
		TopFourRewardTokens:         topFourRewardTokens,
		PrizePoolTokens:             pool,
		SecondsUntilScheduledStart:  secs,
		MinPlayersRequired:          minPlayersRequiredFromConfig(),
		MapRoundConfig:              mapRoundConfigProto,
	}, nil
}

type roundConfig struct {
	TotalPlayerCount          int32
	TokenChange               int32
	PointChange               int32
	RemainingParticipantCount int32
}

func (r roundConfig) toProto() *proto.TournamentRoundConfig {
	return &proto.TournamentRoundConfig{
		TotalPlayerCount:          r.TotalPlayerCount,
		TokenChange:               r.TokenChange,
		PointChange:               r.PointChange,
		RemainingParticipantCount: r.RemainingParticipantCount,
	}
}

func roundConfigMapToProto(m map[int32]roundConfig) map[int32]*proto.TournamentRoundConfig {
	out := make(map[int32]*proto.TournamentRoundConfig, len(m))
	for tierNo, cfg := range m {
		out[tierNo] = cfg.toProto()
	}
	return out
}

func tournamentTierBracketSize(participantCount int64) int32 {
	switch {
	case participantCount < 128:
		return 64
	case participantCount < 256:
		return 128
	case participantCount < 512:
		return 256
	case participantCount < 1024:
		return 512
	case participantCount < 2048:
		return 1024
	case participantCount < 4096:
		return 2048
	case participantCount < 8192:
		return 4096
	default:
		return 8192
	}
}

func topFourRewardTokensFromTierTable(participantCount int64) ([]int32, error) {
	totalPlayerCount := tournamentTierBracketSize(participantCount)

	var rows []dao.TournamentTierRewardConfig
	if err := db.Get().
		Where("total_player_count = ?", totalPlayerCount).
		Order("tier_no desc").
		Limit(3).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	// [highest, second-highest, second-highest, third-highest]
	out := make([]int32, 4)
	if len(rows) > 0 {
		out[0] = rows[0].RewardToken
	}
	if len(rows) > 1 {
		out[1] = rows[1].RewardToken
		out[2] = rows[1].RewardToken
	}
	if len(rows) > 2 {
		out[3] = rows[2].RewardToken
	}
	return out, nil
}

func readRoundConfigFromTierTable(participantCount int64) (map[int32]roundConfig, error) {
	totalPlayerCount := tournamentTierBracketSize(participantCount)
	rows, err := db.TournamentTierRewardConfigListByBracketSize(totalPlayerCount)
	if err != nil {
		return nil, err
	}
	out := make(map[int32]roundConfig, len(rows))
	for _, row := range rows {
		remainingParticipantCount := row.TotalPlayerCount // 当前 round 剩余玩家数
		if row.TierNo > 1 {
			remainingParticipantCount = row.TotalPlayerCount >> (row.TierNo - 1)
		}
		if remainingParticipantCount < 1 {
			log.Errorf("remaining participant count is less than 1: %d", remainingParticipantCount)
			remainingParticipantCount = 1
		}

		out[row.TierNo] = roundConfig{
			TotalPlayerCount:          row.TotalPlayerCount,
			TokenChange:               row.RewardToken,
			PointChange:               row.Point,
			RemainingParticipantCount: remainingParticipantCount,
		}
	}
	return out, nil
}

func minPlayersRequiredFromConfig() int32 {
	v := int32(config.LSGConf.TournamentCfg.MinPlayersRequired)
	if v <= 0 {
		return 64
	}
	return v
}
