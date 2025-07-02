package main

import (
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	api.Register(TestAction, newTestFooTask, api.COOKIEAUTH)
}

var TestAction = "TEST"

type TestFooRequest struct {
	api.BaseRequest
	Foo string
}
type TestbarResponse struct {
	api.BaseResponse
	Bar string
}

type testFooTask struct {
	Request  *TestFooRequest
	Response *TestbarResponse
}

func (t *testFooTask) Run(c *gin.Context) (api.Response, error) {
	if t.Request.Foo != "FOO" {
		return nil, errors.ParamsError("request message is not FOO")
	}
	t.Response.Bar = "BAR"
	return t.Response, nil
}

// 将 map 类型的数据解码为 LoginDillRequest 结构体，并提取 RequestUUID
func newTestFooRequest(data *map[string]interface{}) (*TestFooRequest, error) {
	req := &TestFooRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)

	return req, nil
}

func newTestFooResponse(sessionId string) *TestbarResponse {
	return &TestbarResponse{
		BaseResponse: api.BaseResponse{
			Action:      TestAction + "Response",
			RequestUUID: sessionId,
		},
	}
}

func newTestFooTask(data *map[string]interface{}) (api.Task, error) {
	req, err := newTestFooRequest(data)
	if err != nil {
		return nil, err
	}
	task := &testFooTask{
		Request:  req,
		Response: newTestFooResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}
