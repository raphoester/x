package local_broker

import (
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xtime"
)

var defaultBroker *Broker

func init() {
	loggingConfig := xlog.SLoggerConfig{}
	loggingConfig.ResetToDefault()
	timeProvider := xtime.RealProvider{}
	defaultBroker = New(xlog.NewSLogger(loggingConfig, timeProvider))
}

func GetDefaultBroker() *Broker {
	return defaultBroker
}

func SetDefaultBroker(b *Broker) {
	defaultBroker = b
}
