package dao

import "time"

// UserInviterReward is a per-milestone reward row for the inviter.
type UserInviterReward struct {
	BaseModel
	RelationID      uint       `gorm:"column:relation_id;not null;index:idx_relation_milestone,unique" json:"relation_id"`
	InviterPlayerID int64      `gorm:"column:inviter_player_id;type:bigint;not null;index" json:"inviter_player_id"`
	InviteePlayerID int64      `gorm:"column:invitee_player_id;type:bigint;not null;index" json:"invitee_player_id"`
	MilestoneLevel  int        `gorm:"column:milestone_level;not null;index:idx_relation_milestone,unique" json:"milestone_level"`
	Point           int32      `gorm:"column:point;not null" json:"point"`
	ClaimedAt       *time.Time `gorm:"column:claimed_at" json:"claimed_at"`
}

func (UserInviterReward) TableName() string {
	return "user_inviter_rewards"
}
