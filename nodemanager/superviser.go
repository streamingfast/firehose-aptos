package nodemanager

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ShinyTrinkets/overseer"
	nodeManager "github.com/streamingfast/node-manager"
	logplugin "github.com/streamingfast/node-manager/log_plugin"
	"github.com/streamingfast/node-manager/metrics"
	"github.com/streamingfast/node-manager/superviser"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Superviser struct {
	*superviser.Superviser

	infoMutex     sync.Mutex
	binary        string
	arguments     []string
	dataDir       string
	lastBlockSeen uint64
	serverId      string
}

func (s *Superviser) GetName() string {
	return "aptos-node"
}

func NewSuperviser(
	binary string,
	arguments []string,
	dataDir string,
	debugDeepMind bool,
	logToZap bool,
	lastSeenBlockNum uint64,
	appLogger *zap.Logger,
	nodelogger *zap.Logger,
) *Superviser {
	// Ensure process manager line buffer is large enough (50 MiB) for our Deep Mind instrumentation outputting lot's of text.
	overseer.DEFAULT_LINE_BUFFER_SIZE = 50 * 1024 * 1024

	supervisor := &Superviser{
		Superviser:    superviser.New(appLogger, binary, arguments),
		binary:        binary,
		arguments:     arguments,
		dataDir:       dataDir,
		lastBlockSeen: lastSeenBlockNum,
	}

	if logToZap {
		supervisor.RegisterLogPlugin(newToZapLogPlugin(debugDeepMind, nodelogger))
	} else {
		supervisor.RegisterLogPlugin(logplugin.NewToConsoleLogPlugin(debugDeepMind))
	}

	appLogger.Info("created aptos superviser", zap.Object("superviser", supervisor))
	return supervisor
}

func (s *Superviser) GetCommand() string {
	return s.binary + " " + strings.Join(s.arguments, " ")
}

func (s *Superviser) Start(options ...nodeManager.StartOption) error {
	s.Logger.Info("re-configuring environment variable to start syncing at correct location", zap.Uint64("starting_version", s.lastBlockSeen))
	// We inherit from parent process env (via `os.Environ()`) and add
	// STARTING_VERSION which will be picked by `apots-node` to determine
	// at which "version" to start.
	s.Env = append(os.Environ(), fmt.Sprintf("STARTING_VERSION=%d", s.lastBlockSeen))

	return s.Superviser.Start(options...)
}

func (s *Superviser) IsRunning() bool {
	isRunning := s.Superviser.IsRunning()
	isRunningMetricsValue := float64(0)
	if isRunning {
		isRunningMetricsValue = float64(1)
	}

	metrics.NodeosCurrentStatus.SetFloat64(isRunningMetricsValue)

	return isRunning
}

func (s *Superviser) LastSeenBlockNum() uint64 {
	return s.lastBlockSeen
}

func (s *Superviser) ServerID() (string, error) {
	return s.serverId, nil
}

func (s *Superviser) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("binary", s.binary)
	enc.AddArray("arguments", stringArray(s.arguments))
	enc.AddString("data_dir", s.dataDir)
	enc.AddUint64("last_block_seen", s.lastBlockSeen)
	enc.AddString("server_id", s.serverId)

	return nil
}

// FIXME: Right now we have the real version because our "Block" are actual Aptos transaction
// but if we change so that a `Block` becomes a set of transactions, then we need to change
// here.
func (s *Superviser) SetLastBlockSeen(blockNum uint64) {
	s.lastBlockSeen = blockNum
}
