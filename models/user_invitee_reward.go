package dao

import "time"

// UserInviteeReward is the invitee's initial invite reward row (one per relation).
type UserInviteeReward struct {
	BaseModel
	RelationID      uint       `gorm:"column:relation_id;not null;uniqueIndex" json:"relation_id"`
	InviteePlayerID int64      `gorm:"column:invitee_player_id;type:bigint;not null;index" json:"invitee_player_id"`
	Point           int32      `gorm:"column:point;not null;default:0" json:"point"`
	ClaimedAt       *time.Time `gorm:"column:claimed_at" json:"claimed_at"`
}

func (UserInviteeReward) TableName() string {
	return "user_invitee_rewards"
}
