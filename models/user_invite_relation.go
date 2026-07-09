package dao

// UserInviteRelation records a successful inviter-invitee binding.
type UserInviteRelation struct {
	BaseModel
	InviterPlayerID int64  `gorm:"column:inviter_player_id;type:bigint;not null;index" json:"inviter_player_id"`
	InviteePlayerID int64  `gorm:"column:invitee_player_id;type:bigint;not null;uniqueIndex" json:"invitee_player_id"`
	InviteCode      string `gorm:"column:invite_code;type:varchar(16);not null" json:"invite_code"`
}

func (UserInviteRelation) TableName() string {
	return "user_invite_relations"
}
