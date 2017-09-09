package leader

import "log"

var (
	logger *log.Logger
)

// Logger is an interface that can be implemented to provide custom log output.
type Logger interface {
	Printf(string, ...interface{})
}

var DefaultLogger Logger = defaultLogger{}

type defaultLogger struct{}

func (defaultLogger) Printf(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf(format, args...)
		return
	}
	log.Printf(format, args...)
}
