package battle

// cardData 卡牌数据
type cardData struct {
	level     string
	lifeForce int
	attack    int
	defense   int
}

// cardDataTable 卡牌属性查表
var cardDataTable = map[string]map[int]cardData{
	"J": {
		0: {level: "normal", lifeForce: 500, attack: 1000, defense: 500},
		1: {level: "rare", lifeForce: 90, attack: 17, defense: 6},
		2: {level: "epic", lifeForce: 100, attack: 19, defense: 7},
		3: {level: "legendary", lifeForce: 110, attack: 21, defense: 8},
	},
	"M": {
		0: {level: "normal", lifeForce: 500, attack: 1000, defense: 500},
		1: {level: "rare", lifeForce: 110, attack: 22, defense: 11},
		2: {level: "epic", lifeForce: 120, attack: 24, defense: 12},
		3: {level: "legendary", lifeForce: 130, attack: 26, defense: 13},
	},
	"S": {
		0: {level: "normal", lifeForce: 500, attack: 1000, defense: 500},
		1: {level: "rare", lifeForce: 100, attack: 20, defense: 9},
		2: {level: "epic", lifeForce: 110, attack: 22, defense: 10},
		3: {level: "legendary", lifeForce: 120, attack: 24, defense: 11},
	},
	"H": {
		0: {level: "normal", lifeForce: 500, attack: 1000, defense: 500},
		1: {level: "rare", lifeForce: 130, attack: 27, defense: 16},
		2: {level: "epic", lifeForce: 140, attack: 29, defense: 17},
		3: {level: "legendary", lifeForce: 150, attack: 31, defense: 18},
	},
	"T": {
		0: {level: "normal", lifeForce: 500, attack: 1000, defense: 500},
		1: {level: "rare", lifeForce: 120, attack: 24, defense: 13},
		2: {level: "epic", lifeForce: 130, attack: 26, defense: 14},
		3: {level: "legendary", lifeForce: 140, attack: 28, defense: 15},
	},
}

// CardFactory 卡牌工厂
type CardFactory struct{}

// NewCardFactory 创建新的卡牌工厂
func NewCardFactory() *CardFactory {
	return &CardFactory{}
}

// CreateCard 根据卡牌字符串创建完整的卡牌对象
// 格式: "J0", "M1", "S2", "H3", "T4" 等
func (cf *CardFactory) CreateCard(cardStr string) Card {
	if len(cardStr) == 0 {
		// 如果卡牌字符串为空，抛出错误
		panic("卡牌字符串不能为空")
	}

	symbol := string(cardStr[0])
	var subType int

	if len(cardStr) == 1 {
		// 如果只有Symbol没有SubType，补充为0
		subType = 0
	} else {
		// 有SubType的情况
		subType = int(cardStr[1] - '0') // 将字符转换为数字
		// 确保小型号在有效范围内
		if subType < 0 || subType > 3 {
			subType = 0
		}
	}

	cardData := cf.getCardData(symbol, subType)
	return Card{
		Symbol:    symbol,
		SubType:   subType,
		Level:     cardData.level,
		LifeForce: cardData.lifeForce,
		Attack:    cardData.attack,
		Defense:   cardData.defense,
	}
}

// getCardData 获取卡牌完整数据（查表实现）
func (cf *CardFactory) getCardData(symbol string, subType int) cardData {
	typeTable, ok := cardDataTable[symbol]
	if !ok {
		// 类型无效，返回默认
		typeTable = cardDataTable["J"]
	}
	data, ok := typeTable[subType]
	if !ok {
		// 小型号无效，返回0号
		data = typeTable[0]
	}
	return data
}

// ParseCards 解析卡牌字符串为Card对象
func (cf *CardFactory) ParseCards(cardStrings []string) []Card {
	cards := make([]Card, len(cardStrings))
	for i, cardStr := range cardStrings {
		cards[i] = cf.CreateCard(cardStr)
	}
	return cards
}
