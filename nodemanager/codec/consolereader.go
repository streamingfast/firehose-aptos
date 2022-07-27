package codec

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/firehose-aptos/types"
	pbaptos "github.com/streamingfast/firehose-aptos/types/pb/sf/aptos/type/v1"
	"go.uber.org/zap"
)

// ConsoleReader is what reads the `geth` output directly. It builds
// up some LogEntry objects. See `LogReader to read those entries .
type ConsoleReader struct {
	lines chan string
	close func()
	done  chan interface{}

	logger *zap.Logger
}

func NewConsoleReader(logger *zap.Logger, lines chan string) (*ConsoleReader, error) {
	l := &ConsoleReader{
		lines:  lines,
		close:  func() {},
		done:   make(chan interface{}),
		logger: logger,
	}
	return l, nil
}

func (r *ConsoleReader) Done() <-chan interface{} {
	return r.done
}

func (r *ConsoleReader) Close() {
	r.close()
}

type parsingStats struct {
	startAt  time.Time
	blockNum uint64
	data     map[string]int
	logger   *zap.Logger
}

func newParsingStats(logger *zap.Logger, block uint64) *parsingStats {
	return &parsingStats{
		startAt:  time.Now(),
		blockNum: block,
		data:     map[string]int{},
		logger:   logger,
	}
}

func (s *parsingStats) log() {
	s.logger.Info("extractor block stats",
		zap.Uint64("block_num", s.blockNum),
		zap.Int64("duration", int64(time.Since(s.startAt))),
		zap.Reflect("stats", s.data),
	)
}

func (s *parsingStats) inc(key string) {
	if s == nil {
		return
	}
	k := strings.ToLower(key)
	value := s.data[k]
	value++
	s.data[k] = value
}

func (r *ConsoleReader) ReadBlock() (out *bstream.Block, err error) {
	block, err := r.next()
	if err != nil {
		return nil, err
	}

	return types.BlockFromProto(block)
}

const (
	LogPrefix = "DMLOG"
	LogInit   = "INIT"
	LogTrx    = "TRX"
)

func (r *ConsoleReader) next() (out *pbaptos.Transaction, err error) {
	for line := range r.lines {
		if !strings.HasPrefix(line, LogPrefix) {
			continue
		}

		// This code assumes that distinct element do not contains space. This can happen
		// for example when exchanging JSON object (although we strongly discourage usage of
		// JSON, use serialized Protobuf object). If you happen to have spaces in the last element,
		// refactor the code here to avoid the split and perform the split in the line handler directly
		// instead.
		tokens := strings.Split(line[len(LogPrefix)+1:], " ")
		if len(tokens) < 2 {
			return nil, fmt.Errorf("invalid log line %q, expecting at least two tokens", line)
		}

		// Order the case from most occurring line prefix to least occurring
		switch tokens[0] {
		case LogInit:
			err = r.readInit(tokens[1:])
		case LogTrx:
			// This end the execution of the reading loop as we have a full block here
			return r.readTransaction(tokens[1:])
		default:
			if r.logger.Core().Enabled(zap.DebugLevel) {
				r.logger.Debug("skipping unknown deep mind log line", zap.String("line", line))
			}
			continue
		}

		if err != nil {
			chunks := strings.SplitN(line, " ", 2)
			return nil, fmt.Errorf("%s: %w (line %q)", chunks[0], err, line)
		}
	}

	r.logger.Info("lines channel has been closed")
	return nil, io.EOF
}

func (r *ConsoleReader) ProcessData(reader io.Reader) error {
	scanner := r.buildScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		r.lines <- line
	}

	if scanner.Err() == nil {
		close(r.lines)
		return io.EOF
	}

	return scanner.Err()
}

func (r *ConsoleReader) buildScanner(reader io.Reader) *bufio.Scanner {
	buf := make([]byte, 50*1024*1024)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(buf, 50*1024*1024)

	return scanner
}

// Format:
// DMLOG INIT <client_name> <client_version> <fork> <firehose_major> <firehose_minor>
func (r *ConsoleReader) readInit(params []string) error {
	if err := validateChunk(params, 5); err != nil {
		return fmt.Errorf("invalid log line length: %w", err)
	}

	clientName := params[0]
	clientVersion := params[1]
	fork := params[2]

	firehoseMajor, err := strconv.ParseUint(params[3], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid firehose major version %q: %w", params[3], err)
	}

	firehoseMinor, err := strconv.ParseUint(params[3], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid firehose minor version %q: %w", params[4], err)
	}

	if firehoseMajor != 0 {
		return fmt.Errorf("only able to consume firehose format with major version 0, got %d", firehoseMajor)
	}

	r.logger.Info("initialized firehose extractor consumer correclty",
		zap.String("client_name", clientName),
		zap.String("client_version", clientVersion),
		zap.String("fork", fork),
		zap.Uint64("firehose_major", firehoseMajor),
		zap.Uint64("firehose_minor", firehoseMinor),
	)

	return nil
}

// Format:
// DMLOG TRX <sf.aptos.type.v1.Transaction>
func (r *ConsoleReader) readTransaction(params []string) (*pbaptos.Transaction, error) {
	if err := validateChunk(params, 1); err != nil {
		return nil, fmt.Errorf("invalid log line length: %w", err)
	}

	out, err := base64.StdEncoding.DecodeString(params[0])
	if err != nil {
		return nil, fmt.Errorf("read trx: invalid base64 value: %w", err)
	}

	transaction := &pbaptos.Transaction{}
	if err := proto.Unmarshal(out, transaction); err != nil {
		return nil, fmt.Errorf("read trx: invalid proto: %w", err)
	}

	r.logger.Debug("console reader read transaction",
		zap.Uint64("version", transaction.Version),
		zap.String("hash", transaction.ID()),
		zap.Uint64("block_height", transaction.BlockHeight),
	)

	return transaction, nil
}

func validateChunk(params []string, count int) error {
	if len(params) != count {
		return fmt.Errorf("%d fields required but found %d", count, len(params))
	}
	return nil
}
