package logging

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

type Config struct {
	LogLevel     string
	LogToFile    bool
	LogFilePath  string
	LogToConsole bool
	DevMode      bool
}

// NewLogger Write leveled structured logs to both file destination and console
// In the format: LogTimeStamp Log level Message
func NewLogger(config Config) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %v", err)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var cores []zapcore.Core

	if config.LogToFile {
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
		logFile, err := os.OpenFile(config.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("can't open log file: %v", err)
		}
		writer := zapcore.AddSync(logFile)
		cores = append(cores, zapcore.NewCore(fileEncoder, writer, level))
	}

	if config.LogToConsole {
		var consoleEncoder zapcore.Encoder
		if config.DevMode {
			consoleEncoder = zapcore.NewConsoleEncoder(encoderConfig)
		} else {
			consoleEncoder = zapcore.NewJSONEncoder(encoderConfig)
		}
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
	}

	core := zapcore.NewTee(cores...)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

/*
Example Usage
config := logger.Config{
    LogLevel:     "info",
    LogToFile:    true,
    LogFilePath:  "/var/log/myapp.log",
    LogToConsole: true,
    DevMode:      false,
}

log, err := logger.NewLogger(config)
if err != nil {
    // Handle error
}
defer log.Sync()

// Use the logger
log.Info("Application started")
*/
