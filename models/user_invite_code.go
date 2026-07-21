package dao

import "time"

// UserInviteCode stores a normal user's invite code and successful invite count.
type UserInviteCode struct {
	PlayerID        int64     `gorm:"column:player_id;type:bigint;primaryKey" json:"player_id"`
	InviteCode      string    `gorm:"column:invite_code;type:varchar(16);not null;uniqueIndex" json:"invite_code"`
	InviteCount     int       `gorm:"column:invite_count;not null;default:0" json:"invite_count"`
	MaxInviteCount  int       `gorm:"column:max_invite_count;not null;default:0" json:"max_invite_count"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserInviteCode) TableName() string {
	return "user_invite_codes"
}
