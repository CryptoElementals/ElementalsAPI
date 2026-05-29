package api

import (
	"github.com/CryptoElementals/common/config"
	"github.com/gin-gonic/gin"
)

const (
	SESSION_SERVER_TYPE_KEY = "server_type"
	ParamsServerTypeKey     = "ServerType"
)

// ApplyServerTypeToParams injects server type into request params.
func ApplyServerTypeToParams(params *map[string]interface{}, serverType string) {
	(*params)[ParamsServerTypeKey] = config.NormalizeServerType(serverType)
}

// ServerTypeFromGin reads server type from middleware-injected params.
func ServerTypeFromGin(c *gin.Context) string {
	_params, ok := c.Get("params")
	if !ok {
		return config.ServerTypeTrial
	}
	params, ok := _params.(*map[string]interface{})
	if !ok {
		return config.ServerTypeTrial
	}
	if v, ok := (*params)[ParamsServerTypeKey].(string); ok && v != "" {
		return config.NormalizeServerType(v)
	}
	return config.ServerTypeTrial
}
