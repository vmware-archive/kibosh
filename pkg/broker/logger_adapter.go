package broker

import (
	"code.cloudfoundry.org/lager"
	"github.com/Sirupsen/logrus"
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
		l.logger.Debug(payload.Message, payload.Error, payload.Data)
		break
	case lager.INFO:
		l.logger.Info(payload.Message, payload.Error, payload.Data)
		break
	case lager.ERROR:
		l.logger.Error(payload.Message, payload.Error, payload.Data)
		break
	case lager.FATAL:
		l.logger.Fatal(payload.Message, payload.Error, payload.Data)
		break
	}
}
