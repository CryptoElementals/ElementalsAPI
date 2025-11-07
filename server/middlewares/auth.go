package middlewares

import (
	"encoding/json"
	"net/http"

	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(serverMode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if serverMode == "test" {
			c.Next()
			return
		}

		action := c.GetString("action")
		_params, _ := c.Get("params")
		params, ok := _params.(*map[string]interface{})
		if !ok {
			res := api.MakeErrorResponse(errors.ParamsJudgeError("params assert failed"))
			resJson, _ := json.Marshal(res)
			log.Infof("Send response---> client %s, %s", c.ClientIP(), string(resJson))
			c.Abort()
			c.JSON(http.StatusBadRequest, res)
			return
		}
		authType := api.GetActionAuthType(action)
		requestUUID := (*params)["RequestUUID"].(string)

		switch authType {
		case api.COOKIEAUTH:
			session := sessions.Default(c)
			sessionID := session.ID()
			log.Debugf("Session ID: %s (Client IP: %s)", sessionID, c.ClientIP())
			//从服务器的会话（session）中查找该 Cookie 是否存在。如果 Cookie 不存在或无效，返回错误响应，表示认证失败。
			addr := session.Get(api.SESSION_ADDR_KEY)
			if addr == nil {
				res := api.MakeErrorResponse(errors.LoginCookieInvalid(""))
				res.SetSession(requestUUID)
				res.SetAction(action + "Response")
				resJson, _ := json.Marshal(res)
				log.Infof("%s Send response---> client %s, %s", requestUUID, c.ClientIP(), string(resJson))
				c.Abort()
				c.JSON(http.StatusUnauthorized, res)
				return
			}

			//add addr to params 将会话中的地址信息 addr 存入请求参数中，替代原来请求中的Address（可能是假的）
			//地址不在request里提供，而是根据cookie在redis里查
			(*params)["Address"] = addr.(string)
		}

		// 继续处理请求
		c.Next()
	}
}
