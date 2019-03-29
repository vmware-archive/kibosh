package logger

import (
	"code.cloudfoundry.org/lager"
	"github.com/sirupsen/logrus"
)

func NewLogrusSink(logger *logrus.Logger) *LogrusSink {
	return &LogrusSink{
		logger: logger,
	}
}

type LogrusSink struct {
	logger *logrus.Logger
}

func (l *LogrusSink) Log(payload lager.LogFormat) {
	switch payload.LogLevel {
	case lager.DEBUG:
		l.logger.Debug(payload.Message, payload.Data)
		break
	case lager.INFO:
		l.logger.Info(payload.Message, payload.Data)
		break
	case lager.ERROR:
		l.logger.WithError(payload.Error).Error(payload.Message, payload.Data)
		break
	case lager.FATAL:
		l.logger.WithError(payload.Error).Fatal(payload.Message, payload.Data)
		break
	}
}
