package errors

import (
	"errors"
	"fmt"
)

type CodeError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *CodeError) Error() string {
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

func NewCodeError(code int, message string) *CodeError {
	return &CodeError{Code: code, Message: message}
}

func IsCodeError(err error) bool {
	_, ok := err.(*CodeError)
	return ok
}

func FromError(err error) *CodeError {
	if err == nil {
		return nil
	}
	if codeErr, ok := err.(*CodeError); ok {
		return codeErr
	}
	return NewCodeError(500, err.Error())
}

var (
	ErrNotFound         = NewCodeError(404, "资源不存在")
	ErrUnauthorized     = NewCodeError(401, "未授权")
	ErrForbidden         = NewCodeError(403, "禁止访问")
	ErrValidationFailed = NewCodeError(400, "参数验证失败")
	ErrInternalServer    = NewCodeError(500, "内部服务器错误")
	ErrDatabase         = NewCodeError(500, "数据库错误")
	ErrRecordNotFound   = errors.New("record not found")
)
