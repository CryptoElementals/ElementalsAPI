package db

import (
	dao "github.com/CryptoElementals/common/models"
)

// CreateRoom 创建房间记录
func CreateRoom(room *dao.Room) error {
	return Get().Create(room).Error
}

// GetRoomByRoomID 根据RoomID获取房间记录
func GetRoomByRoomID(roomID string) (*dao.Room, error) {
	var room dao.Room
	err := Get().Where("room_id = ?", roomID).First(&room).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// GetRoomsByRoomID 根据RoomID获取所有房间记录（一个房间可能有多个玩家）
func GetRoomsByRoomID(roomID string) ([]dao.Room, error) {
	var rooms []dao.Room
	err := Get().Where("room_id = ?", roomID).Find(&rooms).Error
	return rooms, err
}

// GetRoomsByAddress 根据地址获取用户的房间记录
func GetRoomsByAddress(address string) ([]dao.Room, error) {
	var rooms []dao.Room
	err := Get().Where("address = ?", address).Find(&rooms).Error
	return rooms, err
}

// GetActiveRoomByAddress 根据地址获取用户当前活跃的房间记录
func GetActiveRoomByAddress(address string) (*dao.Room, error) {
	var room dao.Room
	err := Get().Where("address = ?", address).Order("created_at DESC").First(&room).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// UpdateRoomStage 更新房间阶段
func UpdateRoomStage(roomID string, address string, stage uint) error {
	return Get().Model(&dao.Room{}).Where("room_id = ? AND address = ?", roomID, address).Update("stage", stage).Error
}

// UpdateRoomCards 更新房间卡牌
func UpdateRoomCards(roomID string, address string, cards string) error {
	return Get().Model(&dao.Room{}).Where("room_id = ? AND address = ?", roomID, address).Update("cards", cards).Error
}

// UpdateRoomStageAndCards 同时更新房间阶段和卡牌
func UpdateRoomStageAndCards(roomID string, address string, stage uint, cards string) error {
	return Get().Model(&dao.Room{}).Where("room_id = ? AND address = ?", roomID, address).Updates(map[string]interface{}{
		"stage": stage,
		"cards": cards,
	}).Error
}

// UpdateRoomBattleState 更新房间对战状态（血量、倍率、阶段完成状态）
func UpdateRoomBattleState(roomID string, address string, playerHP int, multiplier float64, isStageOver bool) error {
	return Get().Model(&dao.Room{}).Where("room_id = ? AND address = ?", roomID, address).Updates(map[string]interface{}{
		"player_hp":     playerHP,
		"multiplier":    multiplier,
		"is_stage_over": isStageOver,
	}).Error
}

// UpdateRoomBattleStateByStage 更新指定stage的房间对战状态（血量、倍率、阶段完成状态）
func UpdateRoomBattleStateByStage(roomID string, address string, stage uint, playerHP int, multiplier float64, isStageOver bool) error {
	return Get().Model(&dao.Room{}).Where("room_id = ? AND address = ? AND stage = ?", roomID, address, stage).Updates(map[string]interface{}{
		"player_hp":     playerHP,
		"multiplier":    multiplier,
		"is_stage_over": isStageOver,
	}).Error
}

// UpdateRoomPlayerHP 更新玩家血量
func UpdateRoomPlayerHP(roomID string, address string, playerHP int) error {
	return Get().Model(&dao.Room{}).Where("room_id = ? AND address = ?", roomID, address).Update("player_hp", playerHP).Error
}

// UpdateRoomMultiplier 更新积分倍率
func UpdateRoomMultiplier(roomID string, address string, multiplier float64) error {
	return Get().Model(&dao.Room{}).Where("room_id = ? AND address = ?", roomID, address).Update("multiplier", multiplier).Error
}

// UpdateRoomStageOver 更新阶段完成状态
func UpdateRoomStageOver(roomID string, address string, isStageOver bool) error {
	return Get().Model(&dao.Room{}).Where("room_id = ? AND address = ?", roomID, address).Update("is_stage_over", isStageOver).Error
}

// GetRoomBattleState 获取房间对战状态
func GetRoomBattleState(roomID string) ([]dao.Room, error) {
	var rooms []dao.Room
	err := Get().Where("room_id = ?", roomID).Find(&rooms).Error
	return rooms, err
}

// GetRoomsByStage 根据阶段获取房间记录
func GetRoomsByStage(roomID string, stage uint) ([]dao.Room, error) {
	var rooms []dao.Room
	err := Get().Where("room_id = ? AND stage = ?", roomID, stage).Find(&rooms).Error
	return rooms, err
}

// DeleteRoomByRoomID 根据RoomID删除房间记录
func DeleteRoomByRoomID(roomID string) error {
	return Get().Where("room_id = ?", roomID).Delete(&dao.Room{}).Error
}

// DeleteRoomByRoomIDAndAddress 根据RoomID和地址删除特定玩家的房间记录
func DeleteRoomByRoomIDAndAddress(roomID string, address string) error {
	return Get().Where("room_id = ? AND address = ?", roomID, address).Delete(&dao.Room{}).Error
}

// GetRoomPlayers 获取房间中的所有玩家
func GetRoomPlayers(roomID string) ([]string, error) {
	var rooms []dao.Room
	err := Get().Where("room_id = ?", roomID).Find(&rooms).Error
	if err != nil {
		return nil, err
	}

	var addresses []string
	for _, room := range rooms {
		addresses = append(addresses, room.Address)
	}
	return addresses, nil
}

// GetRoomsForBattleProcessing 获取需要处理对战推演的房间记录
// 条件：is_stage_over为false的记录
func GetRoomsForBattleProcessing() ([]dao.Room, error) {
	var rooms []dao.Room
	err := Get().Where("is_stage_over = ?", false).Find(&rooms).Error
	return rooms, err
}
