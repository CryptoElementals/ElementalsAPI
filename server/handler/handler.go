package handler

import (
	"encoding/json"
	"net/http"

	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
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
