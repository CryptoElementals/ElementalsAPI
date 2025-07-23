package user

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

// API标签常量
const (
	GET_CARDS_LABEL = "GetCards"
)

// 请求结构体
type GetAllCardsRequest struct {
	api.BaseRequest
}

// 响应用卡牌结构体
// 只包含业务需要的字段
type CardInfo struct {
	CardID             int    `json:"card_id"`
	ElementType        string `json:"element_type"`
	Level              string `json:"level"`
	LifeForce          int    `json:"life_force"`
	Attack             int    `json:"attack"`
	Defense            int    `json:"defense"`
	NormalImageURL     string `json:"normal_image_url"`
	ActiveImageURL     string `json:"active_image_url"`
	BackgroundImageURL string `json:"background_image_url"`
	IconURL            string `json:"icon_url"`
	Description        string `json:"description"`
	Name               string `json:"name"`
}

// 响应结构体
type GetAllCardsResponse struct {
	api.BaseResponse
	Cards []CardInfo `json:"Cards"`
}

// 任务结构体
type GetAllCardsTask struct {
	Request  *GetAllCardsRequest
	Response *GetAllCardsResponse
}

// 任务创建函数
func NewGetAllCardsTask(data *map[string]interface{}) (api.Task, error) {
	req := &GetAllCardsRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.RequestUUID = (*data)["RequestUUID"].(string)

	validate := validator.New()
	err = validate.Struct(req)
	if err != nil {
		return nil, err
	}

	return &GetAllCardsTask{
		Request: req,
		Response: &GetAllCardsResponse{
			BaseResponse: api.BaseResponse{
				Action:      GET_CARDS_LABEL + "Response",
				RequestUUID: req.RequestUUID,
			},
		},
	}, nil
}

// 任务执行函数
func (t *GetAllCardsTask) Run(c *gin.Context) (api.Response, error) {
	// 获取所有卡牌
	dbCards, err := db.GetAllCards()
	if err != nil {
		return nil, errors.OperateDbFailed("获取卡牌列表失败")
	}

	// 转换为响应格式
	var cards []CardInfo
	for _, card := range dbCards {
		cards = append(cards, CardInfo{
			CardID:             card.CardID,
			ElementType:        card.ElementType,
			Level:              card.Level,
			LifeForce:          card.LifeForce,
			Attack:             card.Attack,
			Defense:            card.Defense,
			NormalImageURL:     card.NormalImageURL,
			ActiveImageURL:     card.ActiveImageURL,
			BackgroundImageURL: card.BackgroundImageURL,
			IconURL:            card.IconURL,
			Description:        card.Description,
			Name:               card.Name,
		})
	}

	t.Response.Cards = cards
	return t.Response, nil
}
