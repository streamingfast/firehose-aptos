package nodemanager

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"github.com/ShinyTrinkets/overseer"
	pbaptos "github.com/streamingfast/firehose-aptos/types/pb/sf/aptos/type/v1"
	logplugin "github.com/streamingfast/node-manager/log_plugin"
	"github.com/streamingfast/node-manager/metrics"
	"github.com/streamingfast/node-manager/superviser"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"
)

type Superviser struct {
	*superviser.Superviser

	infoMutex     sync.Mutex
	binary        string
	arguments     []string
	dataDir       string
	lastBlockSeen uint64
	serverId      string
	Logger        *zap.Logger
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
	appLogger *zap.Logger,
	nodelogger *zap.Logger,
) *Superviser {
	// Ensure process manager line buffer is large enough (50 MiB) for our Deep Mind instrumentation outputting lot's of text.
	overseer.DEFAULT_LINE_BUFFER_SIZE = 50 * 1024 * 1024

	supervisor := &Superviser{
		Superviser: superviser.New(appLogger, binary, arguments),
		Logger:     appLogger,
		binary:     binary,
		arguments:  arguments,
		dataDir:    dataDir,
	}

	supervisor.RegisterLogPlugin(logplugin.LogPluginFunc(supervisor.lastBlockSeenLogPlugin))

	if logToZap {
		supervisor.RegisterLogPlugin(newToZapLogPlugin(debugDeepMind, nodelogger))
	} else {
		supervisor.RegisterLogPlugin(logplugin.NewToConsoleLogPlugin(debugDeepMind))
	}

	appLogger.Info("created aptos superviser", zap.Object("superviser", supervisor))
	return supervisor
}

func (s *Superviser) GetCommand() string {
	adjustedArguments := append(s.arguments, "")

	return s.binary + " " + strings.Join(s.arguments, " ")
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

func (s *Superviser) lastBlockSeenLogPlugin(line string) {
	if !strings.HasPrefix(line, "DMLOG TRX") {
		fmt.Println("Received line missed the line!!!!", line)
		return
	}

	// FIXME: That's is really inefficient, we should ask Aptos team to change the format of the
	// message to right away include the version in the line so we don't need to fully decode the
	// content here.
	dataBase64 := line[9:]
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		s.Logger.Warn("unable to decode DMLOG TRX content", zap.Error(err))
		return
	}

	transactionTrimmed := &pbaptos.TransactionTrimmed{}
	if err := proto.Unmarshal(data, transactionTrimmed); err != nil {
		s.Logger.Warn("unable to unmarshal DMLOG TRX content", zap.Error(err))
		return
	}

	// FIXME: Right now we have the real version because our "Block" are actual Aptos transaction
	// but if we change so that a `Block` becomes a set of transactions, then we need to change
	// here.
	s.lastBlockSeen = transactionTrimmed.Version
}
