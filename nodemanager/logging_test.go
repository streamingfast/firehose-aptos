package nodemanager

import (
	"testing"

	"github.com/streamingfast/logging"
	"github.com/stretchr/testify/require"
)

var zlog, _ = logging.PackageLogger("test", "github.com/streamingfast/firehose-aptos/nodemanager/tests")

func init() {
	logging.InstantiateLoggers()
}

func Test_newToZapLogPlugin(t *testing.T) {
	type args struct {
		line string
	}
	tests := []struct {
		name   string
		line   string
		output string
	}{
		{
			"debug level",
			`2022-08-13T17:33:13.498748Z [api] DEBUG  message`,
			`{"level":"debug","msg":"[api] message"}`,
		},
		{
			"info level",
			`2022-08-13T17:33:13.498748Z [api] INFO  message`,
			`{"level":"info","msg":"[api] message"}`,
		},
		{
			"warning level",
			`2022-08-13T17:33:13.498748Z [api] WARN  message`,
			`{"level":"warn","msg":"[api] message"}`,
		},
		{
			"error level",
			`2022-08-13T17:33:13.498748Z [api] ERROR  message`,
			`{"level":"error","msg":"[api] message"}`,
		},
		{
			"multi [component] line",
			`2022-08-13T17:33:13.498748Z [api] INFO file.rs:1 [api] message`,
			`{"level":"info","msg":"[api] file.rs:1 [api] message"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewTestLogger(t)

			logPlugin := newToZapLogPlugin(false, logger.Instance())
			logPlugin.LogLine(tt.line)

			writtenLines := logger.RecordedLines(t)
			require.True(t, len(writtenLines) <= 1)

			actual := ""
			if len(writtenLines) > 0 {
				actual = writtenLines[0]
			}

			require.Equal(t, tt.output, actual)
		})
	}
}
