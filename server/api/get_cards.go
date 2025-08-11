package api

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(GET_CARDS_LABEL, NewGetAllCardsTask, NOAUTH)
}

// 请求结构体
type GetAllCardsRequest struct {
	BaseRequest
}

// 响应用卡牌结构体
// 只包含业务需要的字段
type CardInfo struct {
	CardID             int    `json:"CardId"`
	ElementType        string `json:"ElementType"`
	Level              string `json:"Level"`
	LifeForce          int    `json:"LifeForce"`
	Attack             int    `json:"Attack"`
	Defense            int    `json:"Defense"`
	NormalImageURL     string `json:"NormalImageURL"`
	ActiveImageURL     string `json:"ActiveImageURL"`
	BackgroundImageURL string `json:"BackgroundImageURL"`
	IconURL            string `json:"IconURL"`
	Description        string `json:"Description"`
	Name               string `json:"Name"`
}

// 响应结构体
type GetAllCardsResponse struct {
	BaseResponse
	Cards []CardInfo `json:"Cards"`
}

// 任务结构体
type GetAllCardsTask struct {
	Request  *GetAllCardsRequest
	Response *GetAllCardsResponse
}

// 任务创建函数
func NewGetAllCardsTask(data *map[string]interface{}) (Task, error) {
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
			BaseResponse: BaseResponse{
				Action:      GET_CARDS_LABEL + "Response",
				RequestUUID: req.RequestUUID,
			},
		},
	}, nil
}

// 任务执行函数
func (t *GetAllCardsTask) Run(c *gin.Context) (Response, error) {
	// 获取所有卡牌
	dbCards, err := db.GetAllCards()
	if err != nil {
		return nil, errors.OperateDbFailed("获取卡牌列表失败")
	}

	// 转换为响应格式
	var cards []CardInfo
	for _, card := range dbCards {
		// 为卡牌的四个图片URL生成预签名URL
		normalImageURL := ""
		if card.NormalImageURL != "" {
			normalImageURL, _ = utils.GetPresignedImageURL(card.NormalImageURL)
		}

		activeImageURL := ""
		if card.ActiveImageURL != "" {
			activeImageURL, _ = utils.GetPresignedImageURL(card.ActiveImageURL)
		}

		backgroundImageURL := ""
		if card.BackgroundImageURL != "" {
			backgroundImageURL, _ = utils.GetPresignedImageURL(card.BackgroundImageURL)
		}

		iconURL := ""
		if card.IconURL != "" {
			iconURL, _ = utils.GetPresignedImageURL(card.IconURL)
		}

		cards = append(cards, CardInfo{
			CardID:             card.CardID,
			ElementType:        card.ElementType,
			Level:              card.Level,
			LifeForce:          card.LifeForce,
			Attack:             card.Attack,
			Defense:            card.Defense,
			NormalImageURL:     normalImageURL,
			ActiveImageURL:     activeImageURL,
			BackgroundImageURL: backgroundImageURL,
			IconURL:            iconURL,
			Description:        card.Description,
			Name:               card.Name,
		})
	}

	t.Response.Cards = cards
	return t.Response, nil
}
