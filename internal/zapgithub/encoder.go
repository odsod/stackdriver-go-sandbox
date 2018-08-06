package zapgithub

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const gitHubDotComPathSegment = "github.com/"

func GitHubCallerEncoder(commitHash string) zapcore.CallerEncoder {
	return func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		gitHubURL, err := ParseGitHubURL(caller.File, caller.Line, commitHash)
		if err != nil {
			// Fall back to short caller encoder
			zapcore.ShortCallerEncoder(caller, enc)
		}
		enc.AppendString(gitHubURL)
	}
}

func ParseGitHubURL(filePath string, line int, commitHash string) (string, error) {
	gitHubDotComIndex := strings.Index(filePath, gitHubDotComPathSegment)
	if gitHubDotComIndex == -1 {
		return "", errors.Errorf(
			"path does not contain %v folder: %v",
			gitHubDotComPathSegment, filePath)
	}
	gitHubDotComPath := filePath[gitHubDotComIndex:]
	var numSlashes int
	var repoRootIndex = -1
	for i, r := range gitHubDotComPath {
		if r == '/' {
			numSlashes += 1
		}
		if numSlashes == 3 {
			// github.com/<org>/<repo>/
			//           1     2      3
			//                        i
			repoRootIndex = i
			break
		}
	}
	if repoRootIndex == -1 {
		return "", errors.Errorf("invalid GitHub path: %v", filePath)
	}
	repoRoot := gitHubDotComPath[:repoRootIndex] // without trailing slash
	repoPath := gitHubDotComPath[repoRootIndex:] // with leading slash
	gitHubURL := "https://" + repoRoot +
		"/blob/" + commitHash + repoPath +
		"#L" + strconv.FormatInt(int64(line), 10)
	return gitHubURL, nil
}
