package util

import (
	"os"

	"github.com/go-logr/logr"
)

type FatalLogr struct {
	logr.Logger
}

func (l *FatalLogr) Fatal(err error, msg string, keysAndValues ...interface{}) {
	l.Error(err, msg, keysAndValues...)
	os.Exit(1)
}
