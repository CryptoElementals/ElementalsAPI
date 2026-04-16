package lobbyserver

import (
	"context"
	"errors"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func topFourRewardTokensFromTierTable(participantCount int64) ([]int32, error) {
	totalPlayerCount := int32(64)
	switch {
	case participantCount < 128:
		totalPlayerCount = 64
	case participantCount < 256:
		totalPlayerCount = 128
	case participantCount < 512:
		totalPlayerCount = 256
	case participantCount < 1024:
		totalPlayerCount = 512
	case participantCount < 2048:
		totalPlayerCount = 1024
	case participantCount < 4096:
		totalPlayerCount = 2048
	case participantCount < 8192:
		totalPlayerCount = 4096
	default:
		totalPlayerCount = 8192
	}

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

func minPlayersRequiredFromConfig() int32 {
	v := int32(config.LSGConf.TournamentCfg.MinPlayersRequired)
	if v <= 0 {
		return 64
	}
	return v
}

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
	}, nil
}
