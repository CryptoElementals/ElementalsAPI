package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/CryptoElementals/common/db"
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

			// 从 session 中获取 player_id，然后查询用户档案，注入 Address/Email
			playerID := userStr.(string)
			profile, err := db.GetUserProfileByPlayerID(playerID)
			if err != nil || profile == nil {
				res := api.MakeErrorResponse(errors.LoginCookieInvalid("invalid user session"))
				res.SetSession(requestUUID)
				res.SetAction(action + "Response")
				resJson, _ := json.Marshal(res)
				log.Infof("%s Send response---> client %s, %s", requestUUID, c.ClientIP(), string(resJson))
				c.Abort()
				c.JSON(http.StatusUnauthorized, res)
				return
			}

			// 防御性校验：如果请求体中已经带了 PlayerID，则要求与会话中的一致
			if rawReqPlayerID, exists := (*params)["PlayerID"]; exists && rawReqPlayerID != nil {
				reqPlayerID := fmt.Sprintf("%v", rawReqPlayerID)
				if reqPlayerID != playerID {
					res := api.MakeErrorResponse(errors.LoginCookieInvalid("unexpected player id"))
					res.SetSession(requestUUID)
					res.SetAction(action + "Response")
					resJson, _ := json.Marshal(res)
					log.Infof("%s Send response---> client %s, %s", requestUUID, c.ClientIP(), string(resJson))
					c.Abort()
					c.JSON(http.StatusUnauthorized, res)
					return
				}
			}

			// 注入身份字段以兼容现有 API（覆盖请求体中的 PlayerID）
			(*params)["Address"] = profile.Address
			(*params)["Email"] = profile.Email
			(*params)["PlayerID"] = strconv.FormatInt(profile.PlayerID, 10)

			log.Infof("params: %+v", *params)
		}

		// 继续处理请求
		c.Next()
	}
}
