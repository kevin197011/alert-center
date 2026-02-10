package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func Fail(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    -1,
		Message: message,
		Data:    nil,
	})
}

func FailWithCode(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

func InvalidParams(c *gin.Context, message string) {
	FailWithCode(c, 400, message)
}

func Unauthorized(c *gin.Context, message string) {
	FailWithCode(c, 401, message)
}

func Forbidden(c *gin.Context, message string) {
	FailWithCode(c, 403, message)
}

func NotFound(c *gin.Context, message string) {
	FailWithCode(c, 404, message)
}

func ServerError(c *gin.Context, message string) {
	FailWithCode(c, 500, message)
}

// Error sends JSON with the given HTTP status code and message.
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}
