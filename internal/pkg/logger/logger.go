package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log глобальный логгер
	Log *zap.Logger
)

// Init инициализирует глобальный логгер
func Init(level string) error {
	// Настройка уровня логирования
	var zapLevel zapcore.Level
	err := zapLevel.UnmarshalText([]byte(level))
	if err != nil {
		zapLevel = zapcore.InfoLevel
	}

	// Настройка вывода
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Создаем конфигурацию
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Создаем логгер
	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	Log = logger
	return nil
}

// WithContext добавляет контекстные поля к логгеру
func WithContext(fields ...zapcore.Field) *zap.Logger {
	return Log.With(fields...)
}

// Debug логирует сообщение с уровнем Debug
func Debug(msg string, fields ...zapcore.Field) {
	Log.Debug(msg, fields...)
}

// Info логирует сообщение с уровнем Info
func Info(msg string, fields ...zapcore.Field) {
	Log.Info(msg, fields...)
}

// Warn логирует сообщение с уровнем Warn
func Warn(msg string, fields ...zapcore.Field) {
	Log.Warn(msg, fields...)
}

// Error логирует сообщение с уровнем Error
func Error(msg string, fields ...zapcore.Field) {
	Log.Error(msg, fields...)
}

// Fatal логирует сообщение с уровнем Fatal и завершает программу
func Fatal(msg string, fields ...zapcore.Field) {
	Log.Fatal(msg, fields...)
	os.Exit(1)
}

// Field создает поле для логирования
func Field(key string, value interface{}) zapcore.Field {
	return zap.Any(key, value)
}
