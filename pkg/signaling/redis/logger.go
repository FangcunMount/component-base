package redis

// ErrorLogger 适配具备 Error 方法的日志实现。
type ErrorLogger interface {
	Error(args ...any)
}

func toErrorHandler(logger ErrorLogger) func(error) {
	if logger == nil {
		return nil
	}
	return func(err error) {
		if err != nil {
			logger.Error("signaling redis watch error:", err)
		}
	}
}
