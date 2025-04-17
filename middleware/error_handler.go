package middleware

import (
	"github.com/gin-gonic/gin"
	"wellness-step-by-step/step-08/utils"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Сначала выполняем все обработчики

		// Проверяем, есть ли ошибки
		if len(c.Errors) > 0 {
			for _, ginErr := range c.Errors {
				utils.CaptureError(ginErr.Err, map[string]interface{}{
					"endpoint": c.Request.URL.Path,
					"method":   c.Request.Method,
					"status":   c.Writer.Status(),
				})
			}
		}
	}
}
