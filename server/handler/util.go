package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func sendError(c *gin.Context, requestId string, err error) {
	c.XML(http.StatusOK, &CommonError{
		RetCode:   getCode(err),
		RequestId: requestId,
		Message:   err.Error(),
	})
}

func sendResp[T any](c *gin.Context, requestId string, body T) {
	c.XML(http.StatusOK, &CommonSuccessResponse[T]{
		RetCode:   0,
		RequestId: requestId,
		any:       body,
	})
}

func getCode(err error) int {
	code := RetCodes[err]
	if code == 0 {
		return http.StatusInternalServerError
	}
	return code
}

func extractRequest[T any](c *gin.Context) (*T, error) {
	req := new(T)
	err := c.ShouldBind(req)
	if err != nil {
		err = MalformedRequestBody
	}
	return req, err
}

type CommonError struct {
	RetCode   int
	RequestId string
	Message   string
}

type CommonSuccessResponse[T any] struct {
	RetCode   int
	RequestId string
	any
}
