package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/server/api"

	"github.com/CryptoElementals/common/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func PreJobMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		requestDump, err := httputil.DumpRequest(c.Request, false) //只打印请求头，不打印请求体中的二进制数据
		if err != nil {
			handleAbort(c, errors.ParamsJudgeError("dump request failed"))
			return
		}
		log.Infof("--->Receive request, client %s, %s\n", c.ClientIP(), string(requestDump))

		contentType := c.ContentType()
		params := make(map[string]interface{})

		switch contentType {
		case "application/json":
			if err := c.BindJSON(&params); err != nil {
				handleAbort(c, errors.ParamsJudgeError(err.Error()))
				return
			}

		case "multipart/form-data":
			// log.Infof("processing multipart/form-data")
			if err := c.Request.ParseMultipartForm(5 << 20); err != nil { // 5 MB max memory
				handleAbort(c, errors.ParamsJudgeError("parse multipart form failed"))
				return
			}
			// 处理字段并打印日志
			for key, values := range c.Request.MultipartForm.Value {
				if len(values) > 0 {
					params[key] = values[0]
					// log.Infof("Form Field: %s, Value: %s", key, values[0])
				}
			}

			_, fileHeader, err := c.Request.FormFile("File")
			if err == nil {
				params["FileHeader"] = fileHeader
				// log.Infof("File Field: FileHeader, Filename: %s, Size: %d, Header: %v",
				// 	fileHeader.Filename, fileHeader.Size, fileHeader.Header)
			} else {
				log.Infof("No file uploaded or error: %v", err)
			}

		default:
			handleAbort(c, errors.ParamsJudgeError("Unsupported Content-Type, must be application/json or multipart/form-data"))
			return
		}

		// 格式化参数（补 RequestUUID）
		formatJsonParams(&params)

		// log.Infof("request data: %v", params)

		// 校验 action 字段
		action, myErr := actionCheck(&params)
		if myErr.Code() != 0 {
			handleAbort(c, myErr)
			return
		}

		c.Set("action", action)
		c.Set("params", &params)

		// 请求继续
		c.Next()
	}
}

func actionCheck(params *map[string]interface{}) (string, errors.Error) {
	action := (*params)["Action"]
	if action == nil {
		return "", errors.MissAction("")
	}
	actionStr := fmt.Sprintf("%v", action)
	if !api.Exist(actionStr) {
		return "", errors.ParamsError("action")
	}
	return actionStr, errors.New(0, "").(errors.Error)
}

func formatJsonParams(params *map[string]interface{}) error {
	if (*params)["RequestUUID"] == nil {
		(*params)["RequestUUID"] = uuid.NewString()
	}
	return nil
}

func handleAbort(c *gin.Context, errObj errors.Error) {
	res := api.MakeErrorResponse(errObj)
	resJson, _ := json.Marshal(res)
	log.Infof("Send response---> client %s, %s", c.ClientIP(), string(resJson))

	c.Abort()
	c.JSON(http.StatusBadRequest, res)
}
