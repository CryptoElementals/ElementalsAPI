package system

import (
	"fmt"

	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const LIST_AVATARS_LABEL = "ListAvatars"

// ListAvatarsRequest 请求结构体
type ListAvatarsRequest struct {
	api.BaseRequest
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

	// 从S3获取头像列表
	avatarURLs, err := utils.GetAvatarURLs()
	if err != nil {
		// 获取失败时返回错误
		task.Response.BaseResponse.RetCode = -1
		task.Response.BaseResponse.Message = fmt.Sprintf("Failed to retrieve avatar list: %v", err)
		return task.Response, err
	}

	// 设置响应数据
	task.Response.AvatarURLs = avatarURLs
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Avatar list retrieved successfully"

	return task.Response, nil
}
