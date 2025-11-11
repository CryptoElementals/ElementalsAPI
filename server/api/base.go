package api

import (
	"encoding/json"
	"fmt"

	"github.com/CryptoElementals/common/errors"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
)

const SESSION_USER_KEY = "user"

const (
	LOGIN_TYPE_ADDR  = "addr"
	LOGIN_TYPE_EMAIL = "email"
)

type LoginUser struct {
	Type    string `json:"type"`
	Address string `json:"address,omitempty"`
	Email   string `json:"email,omitempty"`
	Name    string `json:"name,omitempty"`
}

func (u *LoginUser) ToJSON() (string, error) {
	b, err := json.Marshal(u)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func LoginUserFromJSON(s string) (*LoginUser, error) {
	var u LoginUser
	if err := json.Unmarshal([]byte(s), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

type AnyRequest[T any] struct {
	BaseRequest `mapstructure:",squash"`
	Body        T `mapstructure:",squash"`
}

type BaseRequest struct {
	Action      string `mapstructure:"Action"`
	RequestUUID string `mapstructure:"RequestUUID"`
}

func NewBaseRequest(data *map[string]string) (*BaseRequest, error) {
	if len((*data)["RequestUUID"]) == 0 {
		id := uuid.NewString()
		(*data)["RequestUUID"] = id
	}

	var req BaseRequest
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

type BaseResponse struct {
	Action      string `json:"Action,omitempty"`
	RequestUUID string `json:"RequestUUID"`
	RetCode     int    `json:"RetCode"`
	Message     string `json:"Message,omitempty"`
}

type Response interface {
}

func MakeErrorResponse(error errors.Error) *BaseResponse {
	return &BaseResponse{
		RetCode: int(error.Code()),
		Message: string(error.String()),
	}
}

func (br *BaseResponse) SetSession(session string) {
	br.RequestUUID = session
}

func (br *BaseResponse) SetAction(action string) {
	br.Action = action
}

func (br *BaseResponse) SetRetCode(retCode int) {
	br.RetCode = retCode
}

func (br *BaseResponse) SetMessage(message string) {
	br.Message = message
}

func MakeAddrNonceKey(addr string) string {
	return fmt.Sprintf("%s_nonce", addr)
}

// removed legacy addr cookie key helper

func Bool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func String(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func Uint64(ptr *uint64) uint64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func Int64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func Uint32(ptr *uint32) uint32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func Int32(ptr *int32) int32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func Float32(ptr *float32) float32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func Float64(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
