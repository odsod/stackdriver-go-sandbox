package zapgcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestNewGitHubSourceLocator(t *testing.T) {
	gitHubSourceLocator := NewGitHubSourceLocator("b70b22b80c6cd75bbd5dda7b728c4bfc5ffa9647")
	expected := "https://github.com/odsod/stackdriver-go-sandbox/blob/b70b22b80c6cd75bbd5dda7b728c4bfc5ffa9647/cmd/client/main.go#L42"
	actual := gitHubSourceLocator(&zapcore.EntryCaller{
		Defined: true,
		File:    "/home/odsod/go/src/github.com/odsod/stackdriver-go-sandbox/cmd/client/main.go",
		Line:    42,
	}).File
	assert.Equal(t, expected, actual)
}
