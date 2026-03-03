package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	LogLevel        string
	LogToFile       bool
	LogFilePath     string
	LogToConsole    bool
	ConsoleLogLevel string
	DevMode         bool

	// StacktraceLevel controls which log level triggers a stack trace.
	// Default: "dpanic". Valid: "debug","info","warn","error","dpanic","panic","fatal".
	StacktraceLevel string

	// SamplingInitial is the number of identical messages per second to log
	// before sampling kicks in. 0 = sampling disabled.
	SamplingInitial int
	// SamplingThereafter is the 1-in-N rate after the initial burst.
	// Both SamplingInitial and SamplingThereafter must be > 0 to enable sampling.
	SamplingThereafter int

	// File rotation (only used when LogToFile=true).
	MaxFileSizeMB   int  // Max megabytes before rotation. Default: 100.
	MaxBackups      int  // Max rotated files to keep. Default: 3.
	MaxFileAgeDays  int  // Max days to retain rotated files. Default: 28.
	CompressRotated bool // Compress rotated files. Default: true.
}

func NewLogger(config Config) (*zap.Logger, func(), error) {
	level, err := zapcore.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid log level: %v", err)
	}

	consoleLevel, err := zapcore.ParseLevel(config.ConsoleLogLevel)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid console log level: %v", err)
	}

	stackLevel := zapcore.DPanicLevel
	if config.StacktraceLevel != "" {
		parsed, err := zapcore.ParseLevel(config.StacktraceLevel)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid stacktrace level: %v", err)
		}
		stackLevel = parsed
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
	var closeFns []func()

	if config.LogToFile {
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

		maxSize := config.MaxFileSizeMB
		if maxSize == 0 {
			maxSize = 100
		}
		maxBackups := config.MaxBackups
		if maxBackups == 0 {
			maxBackups = 3
		}
		maxAge := config.MaxFileAgeDays
		if maxAge == 0 {
			maxAge = 28
		}
		compress := config.CompressRotated
		if config.MaxFileSizeMB == 0 && config.MaxBackups == 0 && config.MaxFileAgeDays == 0 {
			compress = true
		}

		lj := &lumberjack.Logger{
			Filename:   config.LogFilePath,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		}
		closeFns = append(closeFns, func() { lj.Close() })
		writer := zapcore.AddSync(lj)
		cores = append(cores, zapcore.NewCore(fileEncoder, writer, level))
	}

	if config.LogToConsole {
		var consoleEncoder zapcore.Encoder
		if config.DevMode {
			consoleEncoder = zapcore.NewConsoleEncoder(encoderConfig)
		} else {
			consoleEncoder = zapcore.NewJSONEncoder(encoderConfig)
		}
		consoleLevelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= consoleLevel
		})
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), consoleLevelEnabler))
	}

	// If no logging output is configured, log to stderr as a fallback
	if len(cores) == 0 {
		consoleEncoder := zapcore.NewJSONEncoder(encoderConfig)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stderr), level))
	}

	core := zapcore.NewTee(cores...)

	// Apply log sampling if configured
	if config.SamplingInitial > 0 && config.SamplingThereafter > 0 {
		core = zapcore.NewSamplerWithOptions(core, time.Second, config.SamplingInitial, config.SamplingThereafter)
	}

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(stackLevel))

	cleanup := func() {
		if err := logger.Sync(); err != nil && config.LogToFile {
			fmt.Fprintf(os.Stderr, "zap logger sync error: %v\n", err)
		}
		for _, fn := range closeFns {
			fn()
		}
	}

	return logger, cleanup, nil
}

// --- Context-aware logger helpers ---

type ctxKey struct{}

// ContextWithLogger stores a *zap.Logger in the given context.
func ContextWithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// LoggerFromContext retrieves the *zap.Logger from ctx.
// Returns a no-op logger if none is set.
func LoggerFromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return zap.NewNop()
}

// --- Service identity helper ---

// WithServiceIdentity returns a child logger with service, version, and
// environment fields pre-attached.
func WithServiceIdentity(logger *zap.Logger, service, version, environment string) *zap.Logger {
	return logger.With(
		zap.String("service", service),
		zap.String("version", version),
		zap.String("environment", environment),
	)
}

// --- slog adapter for Echo v5 ---

// NewSlogHandler returns an slog.Handler that routes all log output through
// the given Zap logger. Use this to unify Echo's internal slog-based logging
// with the application's Zap pipeline.
func NewSlogHandler(logger *zap.Logger) slog.Handler {
	return &zapSlogHandler{logger: logger}
}

type zapSlogHandler struct {
	logger *zap.Logger
	attrs  []zap.Field
}

func (h *zapSlogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *zapSlogHandler) Handle(_ context.Context, record slog.Record) error {
	fields := make([]zap.Field, 0, len(h.attrs)+record.NumAttrs())
	fields = append(fields, h.attrs...)
	record.Attrs(func(a slog.Attr) bool {
		fields = append(fields, slogAttrToZapField(a))
		return true
	})

	switch {
	case record.Level >= slog.LevelError:
		h.logger.Error(record.Message, fields...)
	case record.Level >= slog.LevelWarn:
		h.logger.Warn(record.Message, fields...)
	case record.Level >= slog.LevelInfo:
		h.logger.Info(record.Message, fields...)
	default:
		h.logger.Debug(record.Message, fields...)
	}
	return nil
}

func (h *zapSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	fields := make([]zap.Field, 0, len(h.attrs)+len(attrs))
	fields = append(fields, h.attrs...)
	for _, a := range attrs {
		fields = append(fields, slogAttrToZapField(a))
	}
	return &zapSlogHandler{logger: h.logger, attrs: fields}
}

func (h *zapSlogHandler) WithGroup(name string) slog.Handler {
	return &zapSlogHandler{logger: h.logger.Named(name), attrs: h.attrs}
}

func slogAttrToZapField(a slog.Attr) zap.Field {
	switch a.Value.Kind() {
	case slog.KindString:
		return zap.String(a.Key, a.Value.String())
	case slog.KindInt64:
		return zap.Int64(a.Key, a.Value.Int64())
	case slog.KindFloat64:
		return zap.Float64(a.Key, a.Value.Float64())
	case slog.KindBool:
		return zap.Bool(a.Key, a.Value.Bool())
	case slog.KindDuration:
		return zap.Duration(a.Key, a.Value.Duration())
	case slog.KindTime:
		return zap.Time(a.Key, a.Value.Time())
	default:
		return zap.Any(a.Key, a.Value.Any())
	}
}
