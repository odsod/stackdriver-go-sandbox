package zapgcp

import "go.uber.org/zap/zapcore"

func NewEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		NameKey:        "logger",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}
