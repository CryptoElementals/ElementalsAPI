package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthType uint8

const (
	_          AuthType = iota
	NOAUTH              // 无需认证
	VERIFYAUTH          // 鉴权但不使用cookie
	COOKIEAUTH          // 使用cookie认证
)

type Task interface {
	Run(c *gin.Context) (Response, error)
}

// SSETask interface used for Server-Sent Events
type SSETask interface {
	Task
	RunSSE(ctx context.Context, c *gin.Context, writer http.ResponseWriter, flusher http.Flusher, requestUUID string) error
}

type creator func(data *map[string]interface{}) (Task, error)

type component struct {
	creator  creator
	authType AuthType
}

var _factory = make(map[string]component)

func Register(action string, createHandler creator, authType AuthType) {
	_factory[action] = component{
		creator:  createHandler,
		authType: authType,
	}
}

func NewTask(action string, data *map[string]interface{}) (Task, error) {
	return _factory[action].creator(data)
}

func GetAllAction() []string {
	l := make([]string, 0)
	for a := range _factory {
		l = append(l, a)
	}
	return l
}

func Exist(action string) bool {
	_, ok := _factory[action]
	return ok
}

func GetActionAuthType(action string) AuthType {
	return _factory[action].authType
}
