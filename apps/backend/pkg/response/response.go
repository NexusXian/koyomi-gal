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

func Error(c *gin.Context, err *errors.AppError) {
	c.JSON(http.StatusInternalServerError, gin.H{
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
