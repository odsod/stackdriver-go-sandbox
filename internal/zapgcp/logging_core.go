package zapgcp

import (
	"cloud.google.com/go/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

func NewLoggingCore(
	levelEnabler zapcore.LevelEnabler,
	encoder zapcore.Encoder,
	logger *logging.Logger,
	sourceLocator SourceLocator) zapcore.Core {
	return &loggingCore{
		levelEnabler:  levelEnabler,
		encoder:       encoder,
		logger:        logger,
		sourceLocator: sourceLocator,
	}
}

type loggingCore struct {
	levelEnabler  zapcore.LevelEnabler
	encoder       zapcore.Encoder
	logger        *logging.Logger
	sourceLocator SourceLocator
}

func (core *loggingCore) Enabled(level zapcore.Level) bool {
	return core.levelEnabler.Enabled(level)
}

func (core *loggingCore) With(fields []zapcore.Field) zapcore.Core {
	newEncoder := core.encoder.Clone()
	for _, field := range fields {
		field.AddTo(newEncoder)
	}
	return NewLoggingCore(core.levelEnabler, newEncoder, core.logger, core.sourceLocator)
}

func (core *loggingCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if core.Enabled(entry.Level) {
		return checked.AddCore(entry, core)
	}
	return checked
}

func (core *loggingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Generate the message.
	message, err := core.encoder.EncodeEntry(entry, fields)
	if err != nil {
		return errors.Wrap(err, "failed to encode log entry")
	}
	// Determine severity level
	var severity logging.Severity
	switch entry.Level {
	case zapcore.DebugLevel:
		severity = logging.Debug
	case zapcore.InfoLevel:
		severity = logging.Info
	case zapcore.WarnLevel:
		severity = logging.Warning
	case zapcore.ErrorLevel:
		severity = logging.Error
	case zapcore.DPanicLevel:
		severity = logging.Alert
	case zapcore.PanicLevel:
		severity = logging.Alert
	case zapcore.FatalLevel:
		severity = logging.Critical
	default:
		return errors.Errorf("unknown log level: %v", entry.Level)
	}
	// Enqueue the log message for sending
	core.logger.Log(logging.Entry{
		Severity:       severity,
		Payload:        message.String(),
		SourceLocation: core.sourceLocator(&entry.Caller),
	})
	// This always succeeds, but Sync may fail.
	return nil
}

func (core *loggingCore) Sync() error {
	return core.logger.Flush()
}

type StackTracer interface {
	StackTrace() errors.StackTrace
}
