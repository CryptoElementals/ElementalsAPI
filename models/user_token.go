package dao

type UserToken struct {
	BaseModel
	PlayerId     int64 `gorm:"index;not null"`
	Points       int32 `gorm:"default:0"`
	TokenAmount  int32 `gorm:"default:0"`
	LockedTokens []*LockedUserToken
}

type LockedUserToken struct {
	BaseModel
	UserTokenID      uint   `gorm:"not null"`
	TemporaryAddress string `gorm:"index;not null"`
	GameID           uint   `gorm:"not null"`
	TokenAmount      int32  `gorm:"default:0"`
}
