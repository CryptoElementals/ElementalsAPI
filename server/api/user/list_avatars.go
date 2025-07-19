package user

import (
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const LIST_AVATARS_LABEL = "ListAvatars"

// ListAvatarsRequest 请求结构体
type ListAvatarsRequest struct {
	api.BaseRequest
	Address string `mapstructure:"Address" validate:"required"`
}

// ListAvatarsResponse 响应结构体
type ListAvatarsResponse struct {
	api.BaseResponse
	AvatarURLs []string `json:"AvatarURLs"`
}

type ListAvatarsTask struct {
	Request  *ListAvatarsRequest
	Response *ListAvatarsResponse
}

// 解码请求
func NewListAvatarsRequest(data *map[string]interface{}) (*ListAvatarsRequest, error) {
	req := &ListAvatarsRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewListAvatarsResponse(sessionId string) *ListAvatarsResponse {
	return &ListAvatarsResponse{
		BaseResponse: api.BaseResponse{
			Action:      LIST_AVATARS_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewListAvatarsTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewListAvatarsRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ListAvatarsTask{
		Request:  req,
		Response: NewListAvatarsResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *ListAvatarsTask) Run(c *gin.Context) (api.Response, error) {
	// 从请求参数中获取用户地址
	address := task.Request.Address
	if address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "User address cannot be empty"
		return task.Response, nil
	}

	// 应该改成读取us3，然后返回头像列表
	defaultAvatarURLs := []string{
		"https://us3.example.com/avatars/default_avatar_1.png",
		"https://us3.example.com/avatars/default_avatar_2.png",
		"https://us3.example.com/avatars/default_avatar_3.png",
		"https://us3.example.com/avatars/default_avatar_4.png",
		"https://us3.example.com/avatars/default_avatar_5.png",
		"https://us3.example.com/avatars/default_avatar_6.png",
		"https://us3.example.com/avatars/default_avatar_7.png",
		"https://us3.example.com/avatars/default_avatar_8.png",
		"https://us3.example.com/avatars/default_avatar_9.png",
		"https://us3.example.com/avatars/default_avatar_10.png",
	}

	// 设置响应数据
	task.Response.AvatarURLs = defaultAvatarURLs
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Avatar list retrieved successfully"

	return task.Response, nil
}
