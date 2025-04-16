package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

func InitSentry(dsn string) error {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      os.Getenv("APP_ENV"),
		Release:          "wellness-center@" + os.Getenv("APP_VERSION"),
		TracesSampleRate: 0.2,
	})

	if err != nil {
		return fmt.Errorf("sentry initialization failed: %w", err)
	}

	// Flush buffered events before the program terminates
	defer sentry.Flush(2 * time.Second)

	return nil
}

func CaptureError(err error, context map[string]interface{}) {
	if hub := sentry.CurrentHub(); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			for k, v := range context {
				scope.SetExtra(k, v)
			}
			hub.CaptureException(err)
		})
	}
}
