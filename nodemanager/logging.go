package nodemanager

import (
	"regexp"

	logplugin "github.com/streamingfast/node-manager/log_plugin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// This file configures a logging reader that transforms log lines received from the blockchain process running
// and then logs them inside the Firehose stack logging system.
//
// So our regex look like the one below, extracting the `info` value from a group in the regexp.
var logLineRegex = regexp.MustCompile("^[0123][0-9]{3}-[0-9]{1,2}-[0-9]{1,2}T[0-9]{1,2}:[0-9]{1,2}:[0-9]{1,2}\\.[0-9]+Z\\s*(\\[.*\\])?\\s*(ERROR|WARN|DEBUG|INFO)\\s*(.*)")
var panicLineRegex = regexp.MustCompile("^thread '.*' panicked")

func newToZapLogPlugin(debugFirehoseLogs bool, logger *zap.Logger) *logplugin.ToZapLogPlugin {
	return logplugin.NewToZapLogPlugin(debugFirehoseLogs, logger, logplugin.ToZapLogPluginLogLevel(logLevelReader), logplugin.ToZapLogPluginTransformer(stripPrefix))
}

func logLevelReader(in string) zapcore.Level {
	// If the regex does not match the line, log to `INFO` so at least we see something by default.
	groups := logLineRegex.FindStringSubmatch(in)
	if len(groups) <= 3 {
		if panicLineRegex.MatchString(in) {
			return zap.ErrorLevel
		}

		return zap.InfoLevel
	}

	switch groups[2] {
	case "debug", "DEBUG":
		return zap.DebugLevel
	case "info", "INFO":
		return zap.InfoLevel
	case "warn", "warning", "WARN", "WARNING":
		return zap.WarnLevel
	case "error", "ERROR":
		return zap.ErrorLevel
	default:
		return zap.DebugLevel
	}
}

func stripPrefix(in string) string {
	groups := logLineRegex.FindStringSubmatch(in)
	if len(groups) <= 3 {
		return in
	}

	if groups[1] == "" {
		return groups[3]
	}

	return groups[1] + " " + groups[3]
}
