package errors

import (
	"fmt"
	"strings"
)

type Error struct {
	code    ErrorCode
	message string
	session string
	items   []interface{}
}

func New(code int, message string) error {
	return Error{code: ErrorCode(code), message: message}
}

func (e Error) String() string {
	if len(e.items) == 0 {
		return e.message
	}
	c := strings.Count(e.message, "%")
	if c == 0 {
		return e.message + ", " + fmt.Sprint(e.items...)
	}
	if c < len(e.items) {
		return fmt.Sprintf(e.message, e.items[:c]...) + " " + fmt.Sprint(e.items[c:]...)
	}
	for i := len(e.items); i < c; i++ {
		e.items = append(e.items, "")
	}
	return fmt.Sprintf(e.message, e.items...)
}

func (e Error) Error() string {
	return e.String()
}

func (e Error) Code() ErrorCode {
	return e.code
}

func (e Error) Message() string {
	return e.message
}

const _errResponse = `{"RetCode": %d, "Message": "%s","RequestUUID": "%s"}`

func (e Error) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(_errResponse, e.code, e.String(), e.session)), nil
}
