package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

// NewLogger Write leveled structured logs to both file destination and console
// In the format: LogTimeStamp Log level Message
func NewLogger() *zap.Logger {
	_, encodeToConsole := getEncoder() // log to file
	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewTee(
		zapcore.NewCore(encodeToConsole, zapcore.AddSync(os.Stdout), defaultLogLevel),
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger
}

// serialise the log to JSON format
func getEncoder() (zapcore.Encoder, zapcore.Encoder) {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	return zapcore.NewJSONEncoder(config), zapcore.NewConsoleEncoder(config)
}
