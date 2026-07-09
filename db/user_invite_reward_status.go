package db

import (
	"errors"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
)

// InviteeInitialRewardStatus is the invitee's initial invite reward row state.
type InviteeInitialRewardStatus struct {
	HasReward bool
	Claimed   bool
	Point     int32
}

// GetInviteeInitialRewardStatus returns invitee initial reward availability.
func GetInviteeInitialRewardStatus(inviteePlayerID int64) (InviteeInitialRewardStatus, error) {
	var row dao.UserInviteeReward
	err := Get().Where("invitee_player_id = ?", inviteePlayerID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return InviteeInitialRewardStatus{}, nil
	}
	if err != nil {
		return InviteeInitialRewardStatus{}, err
	}
	claimed := row.ClaimedAt != nil
	return InviteeInitialRewardStatus{
		HasReward: true,
		Claimed:   claimed,
		Point:     row.Point,
	}, nil
}
