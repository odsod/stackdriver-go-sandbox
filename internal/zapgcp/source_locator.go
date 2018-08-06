package zapgcp

import (
	"runtime"

	"github.com/odsod/stackdriver-go-sandbox/internal/zapgithub"
	"go.uber.org/zap/zapcore"
	logpb "google.golang.org/genproto/googleapis/logging/v2"
)

// SourceLocator converts a zap EntryCaller to a LogEntrySourceLocation.
type SourceLocator func(*zapcore.EntryCaller) *logpb.LogEntrySourceLocation

// NewGitHubSourceLocator returns a locator that resolves an EntryCaller to a GitHub URL.
func NewGitHubSourceLocator(commitHash string) SourceLocator {
	// File: "/home/oscar/go/src/github.com/odsod/stackdriver-go-sandbox/cmd/client/main.go"
	return func(caller *zapcore.EntryCaller) *logpb.LogEntrySourceLocation {
		if !caller.Defined {
			return nil
		}
		gitHubURL, err := zapgithub.ParseGitHubURL(caller.File, caller.Line, commitHash)
		if err != nil {
			// Fall back to file and function source locator
			return FileAndFunctionSourceLocator(caller)
		}
		return &logpb.LogEntrySourceLocation{File: gitHubURL}
	}
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
