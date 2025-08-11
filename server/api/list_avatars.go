package api

import (
	"fmt"

	"github.com/CryptoElementals/common/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(LIST_AVATARS_LABEL, NewListAvatarsTask, NOAUTH)
}

// ListAvatarsRequest 请求结构体
type ListAvatarsRequest struct {
	BaseRequest
}

// ListAvatarsResponse 响应结构体
type ListAvatarsResponse struct {
	BaseResponse
	Avatars []AvatarData `json:"Avatars"`
}

// AvatarData 头像数据结构体
type AvatarData struct {
	AvatarName    string `json:"AvatarName"`
	AvatarURL     string `json:"AvatarURL"`
	BackgroundURL string `json:"BackgroundURL"`
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
		BaseResponse: BaseResponse{
			Action:      LIST_AVATARS_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewListAvatarsTask(data *map[string]interface{}) (Task, error) {
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

func (task *ListAvatarsTask) Run(c *gin.Context) (Response, error) {

	// 从S3获取头像列表
	avatarFiles, err := utils.ListAvatarFiles()
	if err != nil {
		// 获取失败时返回错误
		task.Response.BaseResponse.RetCode = -1
		task.Response.BaseResponse.Message = fmt.Sprintf("Failed to retrieve avatar list: %v", err)
		return task.Response, err
	}

	// 构建头像和背景图的配对
	var avatars []AvatarData
	for _, avatarFilename := range avatarFiles {
		// 生成头像预签名URL
		avatarURL, err := utils.GetPresignedImageURL(avatarFilename)
		if err != nil {
			continue // 跳过有问题的头像文件
		}

		// 根据头像文件名生成对应的背景文件名
		backgroundFilename := utils.GetBackgroundFilenameFromAvatarFilename(avatarFilename)

		// 生成背景图预签名URL
		backgroundURL, err := utils.GetPresignedImageURL(backgroundFilename)
		if err != nil {
			// 如果背景图不存在，使用空字符串
			backgroundURL = ""
		}

		avatars = append(avatars, AvatarData{
			AvatarName:    avatarFilename,
			AvatarURL:     avatarURL,
			BackgroundURL: backgroundURL,
		})
	}

	// 设置响应数据
	task.Response.Avatars = avatars
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Avatar and background list retrieved successfully"

	return task.Response, nil
}
