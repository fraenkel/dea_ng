package utils

import (
	steno "github.com/cloudfoundry/gosteno"
)

func Logger(loggerName string) *steno.Logger {
	return steno.NewLogger(loggerName)
}
