package lobbyserver

import (
	"context"
	"errors"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func topFourRewardTokensFromConfig() []int32 {
	p := config.LSGConf.TournamentCfg.TopFourPrizeTokens
	out := make([]int32, 4)
	for i := 0; i < 4 && i < len(p); i++ {
		out[i] = p[i]
	}
	return out
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
		TopFourRewardTokens:         topFourRewardTokensFromConfig(),
		PrizePoolTokens:             pool,
		SecondsUntilScheduledStart:  secs,
	}, nil
}
