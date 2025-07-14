package db

import (
	dao "github.com/CryptoElementals/common/models"
)

// GetCardByID 根据卡牌ID获取卡牌信息
func GetCardByID(cardID int) (*dao.Card, error) {
	var card dao.Card
	err := Get().Where("card_id = ?", cardID).First(&card).Error
	if err != nil {
		return nil, err
	}
	return &card, nil
}

// GetCardsByElementType 根据五行属性获取所有卡牌
func GetCardsByElementType(elementType string) ([]dao.Card, error) {
	var cards []dao.Card
	err := Get().Where("element_type = ?", elementType).Find(&cards).Error
	return cards, err
}

// GetCardsByLevel 根据等级获取所有卡牌
func GetCardsByLevel(level string) ([]dao.Card, error) {
	var cards []dao.Card
	err := Get().Where("level = ?", level).Find(&cards).Error
	return cards, err
}

// GetAllCards 获取所有卡牌
func GetAllCards() ([]dao.Card, error) {
	var cards []dao.Card
	err := Get().Order("card_id").Find(&cards).Error
	return cards, err
}

// CreateCard 创建新卡牌
func CreateCard(card *dao.Card) error {
	return Get().Create(card).Error
}

// UpdateCard 更新卡牌信息
func UpdateCard(card *dao.Card) error {
	return Get().Save(card).Error
}

// DeleteCard 删除卡牌
func DeleteCard(cardID int) error {
	return Get().Where("card_id = ?", cardID).Delete(&dao.Card{}).Error
}

// GetCardCount 获取卡牌总数
func GetCardCount() (int64, error) {
	var count int64
	err := Get().Model(&dao.Card{}).Count(&count).Error
	return count, err
}

// GetCardsByIDs 根据卡牌ID列表批量获取卡牌
func GetCardsByIDs(cardIDs []int) ([]dao.Card, error) {
	var cards []dao.Card
	err := Get().Where("card_id IN ?", cardIDs).Find(&cards).Error
	return cards, err
}

// SearchCards 搜索卡牌（支持名称、描述模糊搜索）
func SearchCards(keyword string) ([]dao.Card, error) {
	var cards []dao.Card
	err := Get().Where("name LIKE ? OR description LIKE ?",
		"%"+keyword+"%", "%"+keyword+"%").Find(&cards).Error
	return cards, err
}
