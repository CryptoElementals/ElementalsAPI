package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/sse"
	"github.com/gin-gonic/gin"
)

func Handle(c *gin.Context) {
	var err error

	var (
		task api.Task
		res  api.Response
	)

	cookies := c.Request.Cookies()
	for _, cookie := range cookies {
		log.Infof("Cookie: %s = %s\n", cookie.Name, cookie.Value)
	}

	action := c.GetString("action")
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		res := api.MakeErrorResponse(errors.ParamsJudgeError("params assert failed"))
		resJson, _ := json.Marshal(res)
		log.Debugf("Error response params: %s", string(resJson))
		log.Infof("Send response---> client %s, %s", c.ClientIP(), string(resJson))
		c.Abort()
		c.JSON(http.StatusBadRequest, res)
		return
	}

	requestUUID := (*params)["RequestUUID"].(string)

	task, err = api.NewTask(action, params)
	if err != nil {
		res := api.MakeErrorResponse(errors.ParamsJudgeError(err.Error()))
		res.SetSession(requestUUID)
		res.SetAction(action + "Response")
		resJson, _ := json.Marshal(res)
		log.Debugf("Task creation error response: %s", string(resJson))
		log.Infof("Send response---> client %s, %s", c.ClientIP(), string(resJson))
		c.Abort()
		c.JSON(http.StatusBadRequest, res)
		return
	}

	// 检查是否是 SSE 任务（通过 action 名称判断）
	if isSSETask(action) {
		// 切换到 SSE 模式
		handleSSEMode(c, task, action, requestUUID)
		return
	}

	// 普通 HTTP 模式
	res, err = task.Run(c)
	if err == nil {
		resJson, err := json.Marshal(res)
		if err == nil {
			log.Debugf("Success response: %s", string(resJson))
			log.Infof("Send response---> client %s, %s", c.ClientIP(), string(resJson))
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		// 检查是否是自定义错误类型，如果是则直接使用，否则包装成 ActionError
		var errObj errors.Error
		if customErr, ok := err.(errors.Error); ok {
			errObj = customErr
		} else {
			errObj = errors.ActionError(err.Error())
		}
		res := api.MakeErrorResponse(errObj)
		res.SetSession(requestUUID)
		res.SetAction(action + "Response")
		resJson, _ := json.Marshal(res)
		log.Debugf("Task execution error response: %s", string(resJson))
		c.JSON(http.StatusOK, res)
		log.Infof("Send response---> client %s, %s", c.ClientIP(), resJson)
		return
	}
}

// isSSETask 判断是否是 SSE 任务
func isSSETask(action string) bool {
	// 根据 action 名称判断是否是 SSE 任务
	sseActions := []string{"SubscribeGameInfo", "SSEExample"}
	for _, sseAction := range sseActions {
		if action == sseAction {
			return true
		}
	}
	return false
}

// handleSSEMode 处理 SSE 模式
func handleSSEMode(c *gin.Context, task api.Task, action, requestUUID string) {
	// 设置 SSE 必要的头部
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// 获取 writer 和 flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		http.Error(c.Writer, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	log.Infof("SSE connection started - Action: %s, RequestUUID: %s", action, requestUUID)

	// 发送连接建立事件
	startEvent := sse.Event{
		Type: sse.EventTypeNotification,
		Data: map[string]interface{}{
			"Status": "connecting",
		},
		Timestamp:   time.Now(),
		RequestUUID: requestUUID,
	}
	_ = sse.Write(c.Writer, flusher, startEvent)

	// 开始 SSE 流 - 直接使用 Run 方法
	_, err := task.Run(c)
	if err != nil {
		log.Errorf("SSE error for action %s: %v", action, err)
		errorEvent := sse.Event{
			Type: sse.EventTypeError,
			Data: map[string]interface{}{
				"error": err.Error(),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
		_ = sse.Write(c.Writer, flusher, errorEvent)
	}

	log.Infof("SSE connection ended - Action: %s, RequestUUID: %s", action, requestUUID)
}
