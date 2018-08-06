package zapgcp

import (
	"cloud.google.com/go/errorreporting"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

func NewErrorReportingCore(
	levelEnabler zapcore.LevelEnabler,
	client *errorreporting.Client) zapcore.Core {
	return &errorReportingCore{
		levelEnabler: levelEnabler,
		client:       client,
	}
}

type errorReportingCore struct {
	levelEnabler zapcore.LevelEnabler
	client       *errorreporting.Client
}

func (core *errorReportingCore) Enabled(level zapcore.Level) bool {
	return core.levelEnabler.Enabled(level)
}

func (core *errorReportingCore) With(fields []zapcore.Field) zapcore.Core {
	// Ignore fields
	return NewErrorReportingCore(core.levelEnabler, core.client)
}

func (core *errorReportingCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if core.Enabled(entry.Level) {
		return checked.AddCore(entry, core)
	}
	return checked
}

func (core *errorReportingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	for _, field := range fields {
		if field.Type == zapcore.ErrorType {
			if err, ok := field.Interface.(error); ok {
				core.client.Report(errorreporting.Entry{Error: err})
			}
		}
	}
	return nil
}

func (core *errorReportingCore) Sync() error {
	core.client.Flush()
	return nil
}

type StackTracer interface {
	StackTrace() errors.StackTrace
}
