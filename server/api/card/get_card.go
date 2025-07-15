package card

import (
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

// API标签常量
const (
	GET_CARD_BY_ID_LABEL = "GET_CARD_BY_ID"
)

// 请求结构体
type GetCardByIDRequest struct {
	api.BaseRequest
	CardIDs []int `mapstructure:"CardIDs" validate:"required,dive,required"`
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
type GetCardByIDResponse struct {
	api.BaseResponse
	Cards []CardInfo `json:"Cards"`
}

// 任务结构体
type GetCardByIDTask struct {
	Request  *GetCardByIDRequest
	Response *GetCardByIDResponse
}

// 任务创建函数
func NewGetCardByIDTask(data *map[string]interface{}) (api.Task, error) {
	req := &GetCardByIDRequest{}
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

	return &GetCardByIDTask{
		Request: req,
		Response: &GetCardByIDResponse{
			BaseResponse: api.BaseResponse{
				Action:      GET_CARD_BY_ID_LABEL + "Response",
				RequestUUID: req.RequestUUID,
			},
		},
	}, nil
}

// 任务执行函数
func (t *GetCardByIDTask) Run(c *gin.Context) (api.Response, error) {
	var cards []CardInfo
	for _, id := range t.Request.CardIDs {
		card, err := db.GetCardByID(int(id))
		if err != nil {
			return nil, errors.UserNotFound(fmt.Sprintf("卡牌ID %d 不存在", id))
		}
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

// RegisterCardApis 注册卡牌相关API
func RegisterCardApis() {
	api.Register(GET_CARD_BY_ID_LABEL, NewGetCardByIDTask, api.NOAUTH)
}
