package middleware

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func SentryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		hub := sentry.CurrentHub()
		if hub == nil {
			c.Next()
			return
		}

		transactionName := fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		transaction := sentry.StartTransaction(
			c.Request.Context(),
			transactionName,
			sentry.ContinueFromRequest(c.Request),
		)
		defer func() {
			if c.Writer != nil {
				transaction.Status = sentry.HTTPtoSpanStatus(c.Writer.Status())
			}
			transaction.Finish()
		}()

		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetContext("Request", map[string]interface{}{
				"Method":  c.Request.Method,
				"URL":     c.Request.URL.String(),
				"Headers": getSafeHeaders(c.Request.Header),
			})
			scope.SetTag("http.method", c.Request.Method)
			scope.SetTag("http.route", c.Request.URL.Path)
		})

		c.Request = c.Request.WithContext(transaction.Context())
		c.Next()
	}
}

func getSafeHeaders(h http.Header) map[string]interface{} {
	safe := make(map[string]interface{})
	for k, v := range h {
		if strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "Cookie") {
			safe[k] = "[FILTERED]"
		} else {
			safe[k] = v
		}
	}
	return safe
}
