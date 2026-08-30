package response

import (
	"backend/pkg/errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"code": errors.CodeOK,
		"data": data,
		"msg":  "success",
	})
}

func OkWithMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code": errors.CodeOK,
		"msg":  msg,
	})
}

func AcceptedWithMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusAccepted, gin.H{
		"code": errors.CodeOK,
		"msg":  msg,
	})
}

func Error(c *gin.Context, err *errors.AppError) {
	c.JSON(err.StatusCode, gin.H{
		"code": err.Code,
		"msg":  err.Message,
	})
}

func Pagination(c *gin.Context, data any, total int64) {
	c.JSON(http.StatusOK, gin.H{
		"code": errors.CodeOK,
		"data": gin.H{
			"items": data,
			"total": total,
		},
		"message": "success",
	})
}

// MessageResponse is the OpenAPI schema for data-less success responses.
type MessageResponse struct {
	Code int    `json:"code" example:"0"`
	Msg  string `json:"msg" example:"success"`
}

// ErrorResponse is the OpenAPI schema for error responses.
type ErrorResponse struct {
	Code int    `json:"code" example:"233"`
	Msg  string `json:"msg" example:"业务错误"`
}
