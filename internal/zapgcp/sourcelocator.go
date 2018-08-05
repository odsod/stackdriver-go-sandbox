package zapgcp

import (
	"runtime"

	"go.uber.org/zap/zapcore"
	logpb "google.golang.org/genproto/googleapis/logging/v2"
)

// SourceLocator converts a zap EntryCaller to a LogEntrySourceLocation.
type SourceLocator func(*zapcore.EntryCaller) *logpb.LogEntrySourceLocation

// NewGitHubSourceLocator returns a locator that resolves an EntryCaller to a GitHub URL.
func NewGitHubSourceLocator(commitHash string) (SourceLocator, error) {
	return nil, nil
}

// FileAndFunctionSourceLocator attempts to resolve the function name of the EntryCaller PC.
func FileAndFunctionSourceLocator(entryCaller *zapcore.EntryCaller) *logpb.LogEntrySourceLocation {
	if !entryCaller.Defined {
		return nil
	}
	var functionName string
	if f := runtime.FuncForPC(entryCaller.PC); f != nil {
		functionName = f.Name()
	}
	return &logpb.LogEntrySourceLocation{
		File:     entryCaller.File,
		Line:     int64(entryCaller.Line),
		Function: functionName,
	}
}

// NoSourceLocator always returns nil
func NoSourceLocator(*zapcore.EntryCaller) *logpb.LogEntrySourceLocation {
	return nil
}
