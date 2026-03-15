package observability

import "go.uber.org/zap"

func MustLogger(service string) *zap.Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	return logger.With(zap.String("service", service))
}

