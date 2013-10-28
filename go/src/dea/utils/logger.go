package utils

import (
	steno "github.com/cloudfoundry/gosteno"
)

func Logger(loggerName string, data map[string]interface{}) *steno.Logger {
	logger := steno.NewLogger(loggerName)
	if data != nil {
		for k, v := range data {
			logger.Set(k, v)
		}
	}
	return logger
}
