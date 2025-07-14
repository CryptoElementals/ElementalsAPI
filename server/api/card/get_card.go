package card

import (
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services/battle"
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
	CardID int `mapstructure:"CardID" validate:"required"`
}

// 响应结构体
type GetCardByIDResponse struct {
	api.BaseResponse
	Card *battle.Card `json:"Card"`
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
	cardFactory := battle.NewCardFactory()
	card, err := cardFactory.GetCard(t.Request.CardID)
	if err != nil {
		return nil, errors.UserNotFound("卡牌不存在")
	}

	// 转换为battle.Card格式
	battleCard := battle.Card{
		ID:          card.ID,
		ElementType: card.ElementType,
		Level:       card.Level,
		LifeForce:   card.LifeForce,
		Attack:      card.Attack,
		Defense:     card.Defense,
	}

	t.Response.Card = &battleCard
	return t.Response, nil
}

// RegisterCardApis 注册卡牌相关API
func RegisterCardApis() {
	api.Register(GET_CARD_BY_ID_LABEL, NewGetCardByIDTask, api.NOAUTH)
}
