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
			userStr := session.Get(api.SESSION_USER_KEY)
			if userStr == nil {
				res := api.MakeErrorResponse(errors.LoginCookieInvalid(""))
				res.SetSession(requestUUID)
				res.SetAction(action + "Response")
				resJson, _ := json.Marshal(res)
				log.Infof("%s Send response---> client %s, %s", requestUUID, c.ClientIP(), string(resJson))
				c.Abort()
				c.JSON(http.StatusUnauthorized, res)
				return
			}

			// decode user json
			log.Infof("userStr: %s", userStr.(string))
			user, err := api.LoginUserFromJSON(userStr.(string))
			if err != nil || user == nil {
				res := api.MakeErrorResponse(errors.LoginCookieInvalid("invalid user session"))
				res.SetSession(requestUUID)
				res.SetAction(action + "Response")
				resJson, _ := json.Marshal(res)
				log.Infof("%s Send response---> client %s, %s", requestUUID, c.ClientIP(), string(resJson))
				c.Abort()
				c.JSON(http.StatusUnauthorized, res)
				return
			}
			log.Infof("user: %+v", user)
			// 注入身份字段，兼容钱包与邮箱
			(*params)["Address"] = user.Address
			(*params)["Email"] = user.Email

			log.Infof("params: %+v", *params)
		}

		// 继续处理请求
		c.Next()
	}
}
