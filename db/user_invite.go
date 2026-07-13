package db

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/internal/invite"
	"github.com/CryptoElementals/common/internal/playerlevel"
	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const inviteCodeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const inviteCodeLength = 8

// InviteApplyOutcome is the result of an invite bind attempt (business outcome, not DB error).
type InviteApplyOutcome struct {
	WarnCode cmnErrors.WarnCode
	Message  string
}

func inviteOutcome(entry cmnErrors.WarnEntry) InviteApplyOutcome {
	return InviteApplyOutcome{
		WarnCode: entry.Code,
		Message:  entry.Message,
	}
}

// GenerateUniqueInviteCode generates a unique 8-char invite code within tx.
func GenerateUniqueInviteCode(tx *gorm.DB) (string, error) {
	for i := 0; i < 32; i++ {
		code, err := randomInviteCode()
		if err != nil {
			return "", err
		}
		var count int64
		if err := tx.Model(&dao.UserInviteCode{}).Where("invite_code = ?", code).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return code, nil
		}
	}
	return "", errors.New("failed to generate unique invite code")
}

func randomInviteCode() (string, error) {
	b := make([]byte, inviteCodeLength)
	max := big.NewInt(int64(len(inviteCodeAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = inviteCodeAlphabet[n.Int64()]
	}
	return string(b), nil
}

// EnsureInviteCode creates an invite code row for normal users when missing.
func EnsureInviteCode(playerID int64) error {
	profile, err := GetUserProfileByPlayerIDInt(playerID)
	if err != nil {
		return err
	}
	if EffectiveServerType(profile) != dao.ServerTypeNormal {
		return nil
	}
	return Get().Transaction(func(tx *gorm.DB) error {
		var existing dao.UserInviteCode
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("player_id = ?", playerID).First(&existing).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		code, err := GenerateUniqueInviteCode(tx)
		if err != nil {
			return err
		}
		return tx.Create(&dao.UserInviteCode{
			PlayerID:    playerID,
			InviteCode:  code,
			InviteCount: 0,
		}).Error
	})
}

// GetInviteCodeByPlayerID returns the invite code row for a player.
func GetInviteCodeByPlayerID(playerID int64) (*dao.UserInviteCode, error) {
	var row dao.UserInviteCode
	err := Get().Where("player_id = ?", playerID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetInviterByInviteCode looks up the inviter row by invite code.
func GetInviterByInviteCode(code string) (*dao.UserInviteCode, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return nil, nil
	}
	var row dao.UserInviteCode
	err := Get().Where("invite_code = ?", code).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// CheckInviteCodePreLogin validates an invite code before login (read-only).
func CheckInviteCodePreLogin(inviteCode string, maxInviteCount int) (InviteApplyOutcome, error) {
	code := strings.TrimSpace(strings.ToUpper(inviteCode))
	if code == "" {
		return inviteOutcome(cmnErrors.WarnInviteInvalidCode), nil
	}
	inviterRow, err := GetInviterByInviteCode(code)
	if err != nil {
		return InviteApplyOutcome{}, err
	}
	if inviterRow == nil {
		return inviteOutcome(cmnErrors.WarnInviteInvalidCode), nil
	}
	profile, err := GetUserProfileByPlayerIDInt(inviterRow.PlayerID)
	if err != nil {
		return InviteApplyOutcome{}, err
	}
	if profile == nil || EffectiveServerType(profile) != dao.ServerTypeNormal {
		return inviteOutcome(cmnErrors.WarnInviteInvalidCode), nil
	}
	if inviterRow.InviteCount >= maxInviteCount {
		return inviteOutcome(cmnErrors.WarnInviteLimitReached), nil
	}
	return inviteOutcome(cmnErrors.WarnOK), nil
}

// ApplyInviteReferral binds invitee to inviter when allowed.
func ApplyInviteReferral(inviteePlayerID int64, inviteCode string, rewardPoints int32, maxInviteCount int) (InviteApplyOutcome, error) {
	code := strings.TrimSpace(strings.ToUpper(inviteCode))
	if code == "" {
		return inviteOutcome(cmnErrors.WarnOK), nil
	}

	var outcome InviteApplyOutcome
	err := Get().Transaction(func(tx *gorm.DB) error {
		inviterRow, err := getInviterRowForUpdate(tx, code)
		if err != nil {
			return err
		}
		if inviterRow == nil {
			outcome = inviteOutcome(cmnErrors.WarnInviteInvalidCode)
			return nil
		}
		inviterProfile, err := GetUserProfileByPlayerIDWithDB(fmt.Sprintf("%d", inviterRow.PlayerID), tx)
		if err != nil {
			return err
		}
		if EffectiveServerType(inviterProfile) != dao.ServerTypeNormal {
			outcome = inviteOutcome(cmnErrors.WarnInviteInvalidCode)
			return nil
		}
		if inviterRow.PlayerID == inviteePlayerID {
			outcome = inviteOutcome(cmnErrors.WarnInviteSelfInvite)
			return nil
		}
		var existing dao.UserInviteRelation
		err = tx.Where("invitee_player_id = ?", inviteePlayerID).First(&existing).Error
		if err == nil {
			outcome = inviteOutcome(cmnErrors.WarnOK)
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if inviterRow.InviteCount >= maxInviteCount {
			outcome = inviteOutcome(cmnErrors.WarnInviteLimitReached)
			return nil
		}
		relation := dao.UserInviteRelation{
			InviterPlayerID: inviterRow.PlayerID,
			InviteePlayerID: inviteePlayerID,
			InviteCode:      code,
		}
		if err := tx.Create(&relation).Error; err != nil {
			return err
		}
		if err := tx.Create(&dao.UserInviteeReward{
			RelationID:      relation.ID,
			InviteePlayerID: inviteePlayerID,
			Point:           rewardPoints,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&dao.UserInviteCode{}).
			Where("player_id = ?", inviterRow.PlayerID).
			Update("invite_count", gorm.Expr("invite_count + 1")).Error; err != nil {
			return err
		}
		outcome = inviteOutcome(cmnErrors.WarnOK)
		return nil
	})
	return outcome, err
}

func getInviterRowForUpdate(tx *gorm.DB, code string) (*dao.UserInviteCode, error) {
	var row dao.UserInviteCode
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("invite_code = ?", code).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ProcessInviteeLevelMilestonesTx creates inviter milestone rows when invitee level crosses thresholds.
func ProcessInviteeLevelMilestonesTx(tx *gorm.DB, inviteePlayerID int64, pointsBefore, pointsAfter int32) error {
	if pointsAfter <= pointsBefore {
		return nil
	}
	var relation dao.UserInviteRelation
	err := tx.Where("invitee_player_id = ?", inviteePlayerID).First(&relation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	oldLevel := playerlevel.CalculateLevel(int(pointsBefore))
	newLevel := playerlevel.CalculateLevel(int(pointsAfter))
	for _, m := range invite.DefaultInviterMilestones {
		if oldLevel < m.Level && newLevel >= m.Level {
			row := dao.UserInviterReward{
				RelationID:      relation.ID,
				InviterPlayerID: relation.InviterPlayerID,
				InviteePlayerID: inviteePlayerID,
				MilestoneLevel:  m.Level,
				Point:           m.Point,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// GetUnclaimedInviteeReward returns the unclaimed invitee reward row, if any.
func GetUnclaimedInviteeReward(inviteePlayerID int64) (*dao.UserInviteeReward, error) {
	var row dao.UserInviteeReward
	err := Get().Where("invitee_player_id = ? AND claimed_at IS NULL", inviteePlayerID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// MarkInviteeRewardClaimedTx marks invitee reward claimed and processes milestones.
func MarkInviteeRewardClaimedTx(tx *gorm.DB, inviteePlayerID int64, rewardPoints int32) (bool, error) {
	var row dao.UserInviteeReward
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("invitee_player_id = ? AND claimed_at IS NULL", inviteePlayerID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	if err := tx.Model(&row).Update("claimed_at", now).Error; err != nil {
		return false, err
	}
	if err := ProcessInviteeLevelMilestonesTx(tx, inviteePlayerID, 0, rewardPoints); err != nil {
		return false, err
	}
	return true, nil
}

// GetInviterByInviteePlayerID returns the invite relation for an invitee.
func GetInviterByInviteePlayerID(inviteePlayerID int64) (*dao.UserInviteRelation, error) {
	var relation dao.UserInviteRelation
	err := Get().Where("invitee_player_id = ?", inviteePlayerID).First(&relation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &relation, nil
}

// InviteRelationSummary is a joined invite relation for list APIs.
type InviteRelationSummary struct {
	InviteePlayerID int64
	InviteeName     string
	InvitedAt       time.Time
	InviteCode      string
}

// ListUserInviteRelationsByInviter lists invite relations for an inviter with pagination.
func ListUserInviteRelationsByInviter(inviterPlayerID int64, limit, offset int) ([]InviteRelationSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var rows []struct {
		InviteePlayerID int64
		InviteeName     string
		InvitedAt       time.Time
		InviteCode      string
	}
	err := Get().Table("user_invite_relations r").
		Select("r.invitee_player_id, p.name AS invitee_name, r.created_at AS invited_at, r.invite_code").
		Joins("JOIN user_profiles p ON p.player_id = r.invitee_player_id").
		Where("r.inviter_player_id = ?", inviterPlayerID).
		Order("r.created_at DESC").
		Limit(limit).Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]InviteRelationSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, InviteRelationSummary{
			InviteePlayerID: r.InviteePlayerID,
			InviteeName:     r.InviteeName,
			InvitedAt:       r.InvitedAt,
			InviteCode:      r.InviteCode,
		})
	}
	return out, nil
}

// ListInviterMilestoneRewardsForInvitee returns milestone reward rows for one invitee under an inviter.
func ListInviterMilestoneRewardsForInvitee(inviterPlayerID, inviteePlayerID int64) ([]invite.MilestoneRewardState, error) {
	var rows []dao.UserInviterReward
	err := Get().Where("inviter_player_id = ? AND invitee_player_id = ?", inviterPlayerID, inviteePlayerID).
		Order("milestone_level").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]invite.MilestoneRewardState, 0, len(rows))
	for _, row := range rows {
		out = append(out, invite.MilestoneRewardState{
			MilestoneLevel: row.MilestoneLevel,
			Claimed:        row.ClaimedAt != nil,
		})
	}
	return out, nil
}

// InviterRewardSummary is a row for ListInviterRewards.
type InviterRewardSummary struct {
	RewardID        uint
	InviteePlayerID int64
	InviteeName     string
	MilestoneLevel  int
	Point           int32
	Claimed         bool
	Claimable       bool
}

// ListInviterRewards lists inviter milestone reward rows.
func ListInviterRewards(inviterPlayerID int64, optionalInviteePlayerID int64) ([]InviterRewardSummary, error) {
	q := Get().Table("user_inviter_rewards r").
		Select(`r.id AS reward_id, r.invitee_player_id, p.name AS invitee_name,
			r.milestone_level, r.point, r.claimed_at IS NOT NULL AS claimed,
			r.claimed_at IS NULL AS claimable`).
		Joins("JOIN user_profiles p ON p.player_id = r.invitee_player_id").
		Where("r.inviter_player_id = ?", inviterPlayerID)
	if optionalInviteePlayerID > 0 {
		q = q.Where("r.invitee_player_id = ?", optionalInviteePlayerID)
	}
	var rows []InviterRewardSummary
	if err := q.Order("r.invitee_player_id, r.milestone_level").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// MarkInviterRewardClaimedTx marks a single inviter milestone reward as claimed.
func MarkInviterRewardClaimedTx(tx *gorm.DB, inviterPlayerID int64, rewardID uint) (*dao.UserInviterReward, error) {
	var row dao.UserInviterReward
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND inviter_player_id = ? AND claimed_at IS NULL", rewardID, inviterPlayerID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := tx.Model(&row).Update("claimed_at", now).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// GetInviterRewardByID returns an inviter reward row by id.
func GetInviterRewardByID(rewardID uint) (*dao.UserInviterReward, error) {
	var row dao.UserInviterReward
	err := Get().Where("id = ?", rewardID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
