package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"go.uber.org/zap"
)

type bootstrapper struct {
	nodeDataDir               string
	nodeConfigFile            string
	resolvedNodeConfigFile    string
	nodeGenesisFile           string
	nodeWaypointFile          string
	nodeValidatorIdentityFile string
	nodeVFNIdentityFile       string
	logger                    *zap.Logger
}

func (b *bootstrapper) Bootstrap() error {
	b.logger.Info("bootstraping node's configuration")

	if err := makeDirs([]string{b.nodeDataDir}); err != nil {
		return fmt.Errorf(`create "fireaptos" inside node's data dir: %w`, err)
	}

	configContent, err := os.ReadFile(b.nodeConfigFile)
	if err != nil {
		return fmt.Errorf("read config content: %w", err)
	}

	dataDir := tryToMakeAbsolutePath(b.logger, b.nodeDataDir)

	b.logger.Info(`replacing occurrences of "{data-dir}" in config file`, zap.String("config_file", b.nodeConfigFile), zap.String("replacement", dataDir))
	configContent = bytes.ReplaceAll(configContent, []byte("{data-dir}"), []byte(dataDir))

	templatingDirectives := map[string]string{
		"{genesis-file}":            b.nodeGenesisFile,
		"{waypoint-file}":           b.nodeWaypointFile,
		"{validator-identity-file}": b.nodeValidatorIdentityFile,
		"{vfn-identity-file}":       b.nodeVFNIdentityFile,
	}

	for templateDirective, file := range templatingDirectives {
		if configContent, err = b.maybeTemplateFileInConfig(configContent, templateDirective, dataDir, file); err != nil {
			return fmt.Errorf("replacing template directive %q: %w", templateDirective, err)
		}
	}

	if err := os.WriteFile(b.resolvedNodeConfigFile, configContent, os.ModePerm); err != nil {
		return fmt.Errorf("write resolved config file: %w", err)
	}

	return nil
}

func (b *bootstrapper) maybeTemplateFileInConfig(inConfig []byte, templateDirective string, absDataDir string, in string) ([]byte, error) {
	if in != "" {
		resolvedAbsoluteFile, err := b.resolveFileToDataDir(absDataDir, in)
		if err != nil {
			return nil, fmt.Errorf("resolving file: %w", err)
		}

		b.logger.Info(fmt.Sprintf(`replacing occurrences of %s in config file`, templateDirective), zap.String("config_file", b.nodeConfigFile), zap.String("replacement", resolvedAbsoluteFile))
		return bytes.ReplaceAll(inConfig, []byte(templateDirective), []byte(resolvedAbsoluteFile)), nil
	}

	return inConfig, nil
}

var httpSchemePrefixRegex = regexp.MustCompile("^https?://")

func (b *bootstrapper) resolveFileToDataDir(absDataDir string, in string) (absolutePath string, err error) {
	if httpSchemePrefixRegex.MatchString(in) {
		return b.downloadFileToDataDir(absDataDir, in)
	}

	return b.copyFileToDataDir(absDataDir, in)
}

func (b *bootstrapper) copyFileToDataDir(absDataDir string, in string) (absolutePath string, err error) {
	baseName := filepath.Base(in)
	destinationPath := filepath.Join(b.nodeDataDir, baseName)

	if err := copyFile(in, destinationPath); err != nil {
		return "", fmt.Errorf("copy to destination: %w", err)
	}

	return destinationPath, nil
}

func (b *bootstrapper) downloadFileToDataDir(absDataDir string, in string) (absolutePath string, err error) {
	// FIXME: Right now we download it each time we start, should we avoid doing that?

	b.logger.Info("downloading remote file and copying it to node data dir", zap.String("url", in))
	client := http.Client{Timeout: 60 * time.Second}
	response, err := client.Get(in)
	if err != nil {
		return "", fmt.Errorf("fetch file %q: %w", in, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		buf := bytes.NewBuffer(nil)
		if _, err := buf.ReadFrom(response.Body); err != nil {
			buf = bytes.NewBufferString("<Unable to read body>")
		}

		return "", fmt.Errorf("invalid response %q (body %q)", response.Status, buf.String())
	}

	destinationPath := filepath.Join(b.nodeDataDir, path.Base(in))
	b.logger.Debug("copying downloaded file to destination", zap.String("destination", destinationPath))

	outFile, err := os.Create(destinationPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, response.Body)
	if err != nil {
		return "", fmt.Errorf("copy response to %q: %w", destinationPath, err)
	}

	return destinationPath, nil
}

func tryToMakeAbsolutePath(logger *zap.Logger, path string) string {
	out, err := filepath.Abs(path)
	if err == nil {
		return out
	}

	logger.Warn("unable to make path absolute", zap.String("path", path), zap.Error(err))
	return path
}
