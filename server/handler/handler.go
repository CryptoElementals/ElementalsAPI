package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		res := api.MakeErrorResponse(errors.ActionError(err.Error()))
		res.SetSession(requestUUID)
		res.SetAction(action + "Response")
		resJson, _ := json.Marshal(res)
		log.Debugf("Task execution error response: %s", string(resJson))
		c.JSON(http.StatusOK, res)
		log.Infof("Send response---> client %s, %s", c.ClientIP(), resJson)
		return
	}
}

// HandleSSE 处理 Server-Sent Events 请求
func HandleSSE(c *gin.Context) {
	// 设置 SSE 必要的头部
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// 获取 writer 和 flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		http.Error(c.Writer, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// 获取 action 和参数（通过中间件解析）
	action := c.GetString("action")
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		// 发送错误事件
		errorEvent := events.Event{
			Type: events.EventTypeError,
			Data: map[string]interface{}{
				"error": "params assert failed",
			},
			RequestUUID: uuid.NewString(),
		}
		sendSSEEvent(c.Writer, flusher, errorEvent)
		return
	}

	requestUUID := (*params)["RequestUUID"].(string)
	log.Infof("SSE connection started - Action: %s, RequestUUID: %s", action, requestUUID)

	// 创建任务
	task, err := api.NewTask(action, params)
	if err != nil {
		errorEvent := events.Event{
			Type: events.EventTypeError,
			Data: map[string]interface{}{
				"error": err.Error(),
			},
			RequestUUID: requestUUID,
		}
		sendSSEEvent(c.Writer, flusher, errorEvent)
		return
	}

	// 检查任务是否支持 SSE
	sseTask, ok := task.(api.SSETask)
	if !ok {
		errorEvent := events.Event{
			Type: events.EventTypeError,
			Data: map[string]interface{}{
				"error": fmt.Sprintf("action %s does not support SSE", action),
			},
			RequestUUID: requestUUID,
		}
		sendSSEEvent(c.Writer, flusher, errorEvent)
		return
	}

	// 创建 SSE 连接
	connID := fmt.Sprintf("%s_%s", action, requestUUID)
	conn := events.NewSSEConnection(connID, c.Writer, flusher, requestUUID)

	// 注册连接到事件管理器
	eventManager := events.GetEventManager()
	eventManager.AddConnection(conn)

	// 发送连接建立事件
	startEvent := events.Event{
		Type: events.EventTypeStatusUpdate,
		Data: map[string]interface{}{
			"status": "connected",
			"action": action,
		},
		RequestUUID: requestUUID,
	}
	sendSSEEvent(c.Writer, flusher, startEvent)

	// 开始 SSE 流
	ctx := c.Request.Context()
	err = sseTask.RunSSE(ctx, c.Writer, flusher, requestUUID)
	if err != nil {
		log.Errorf("SSE error for action %s: %v", action, err)
		errorEvent := events.Event{
			Type: events.EventTypeError,
			Data: map[string]interface{}{
				"error": err.Error(),
			},
			RequestUUID: requestUUID,
		}
		sendSSEEvent(c.Writer, flusher, errorEvent)
	}

	// 清理连接
	eventManager.RemoveConnection(connID)
}

// sendSSEEvent 发送 SSE 事件
func sendSSEEvent(writer http.ResponseWriter, flusher http.Flusher, event events.Event) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Failed to marshal SSE event: %v", err)
		return
	}

	// SSE 格式：data: {json}\n\n
	eventStr := fmt.Sprintf("data: %s\n\n", string(jsonData))
	_, err = writer.Write([]byte(eventStr))
	if err != nil {
		log.Errorf("Failed to write SSE event: %v", err)
		return
	}

	flusher.Flush()
}
