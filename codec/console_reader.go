package codec

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/firehose-aptos/types"
	pbaptos "github.com/streamingfast/firehose-aptos/types/pb/aptos/extractor/v1"
	"go.uber.org/zap"
)

// ConsoleReader is what reads the `geth` output directly. It builds
// up some LogEntry objects. See `LogReader to read those entries .
type ConsoleReader struct {
	lines  chan string
	close  func()
	done   chan interface{}
	logger *zap.Logger

	activeBlockStartTime time.Time
	activeBlock          *pbaptos.Block
	initRead             bool
	stats                *consoleReaderStats
}

func NewConsoleReader(logger *zap.Logger, lines chan string) (*ConsoleReader, error) {
	l := &ConsoleReader{
		lines:  lines,
		close:  func() {},
		done:   make(chan interface{}),
		logger: logger,

		stats: newConsoleReaderStats(),
	}

	l.stats.StartPeriodicLogToZap(context.Background(), logger, 30*time.Second)

	return l, nil
}

func (r *ConsoleReader) Done() <-chan interface{} {
	return r.done
}

func (r *ConsoleReader) Close() {
	r.stats.StopPeriodicLogToZap()

	r.close()
}

func (r *ConsoleReader) ReadBlock() (out *bstream.Block, err error) {
	block, err := r.next()
	if err != nil {
		return nil, err
	}

	return types.BlockFromProto(block)
}

const (
	LogPrefix     = "FIRE"
	LogInit       = "INIT"
	LogBlockStart = "BLOCK_START"
	LogTrx        = "TRX"
	LogBlockEnd   = "BLOCK_END"
)

func (r *ConsoleReader) next() (out *pbaptos.Block, err error) {
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

		if !r.initRead {
			if tokens[0] == LogInit {
				if r.readInit(tokens[1:]); err != nil {
					return nil, lineError(line, err)
				}
			} else {
				r.logger.Debug("received line with prefix %q but we did not see INIT yet, skipping line", zap.String("prefix", tokens[0]))
			}

			continue
		}

		// Order the case from most occurring line prefix to least occurring
		switch tokens[0] {
		case LogTrx:
			err = r.readTransaction(tokens[1:])

		case LogBlockStart:
			err = r.readBlockStart(tokens[1:])

		case LogBlockEnd:
			// This end the execution of the reading loop as we have a full block here
			block, err := r.readBlockEnd(tokens[1:])
			if err != nil {
				return nil, lineError(line, err)
			}

			return block, nil

		case LogInit:
			err = fmt.Errorf("received INIT line while one has already been read")

		default:
			if r.logger.Core().Enabled(zap.DebugLevel) {
				r.logger.Debug("skipping unknown deep mind log line", zap.String("line", line))
			}

			continue
		}

		if err != nil {
			return nil, lineError(line, err)
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
// FIRE INIT <client_name> <client_version> <fork> <firehose_major> <firehose_minor>
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

	r.logger.Info("initialized console reader correclty",
		zap.String("client_name", clientName),
		zap.String("client_version", clientVersion),
		zap.String("fork", fork),
		zap.Uint64("firehose_major", firehoseMajor),
		zap.Uint64("firehose_minor", firehoseMinor),
	)

	r.initRead = true

	return nil
}

// Format:
// FIRE BLOCK_START <height>
func (r *ConsoleReader) readBlockStart(params []string) error {
	if err := validateChunk(params, 1); err != nil {
		return fmt.Errorf("invalid BLOCK_START line: %w", err)
	}

	height, err := strconv.ParseUint(params[0], 10, 64)
	if err != nil {
		return fmt.Errorf(`invalid BLOCK_START "height" param: %w`, err)
	}

	if r.activeBlock != nil {
		r.logger.Info("received BLOCK_START while one is already active, resetting active block and starting over",
			zap.Uint64("previous_active_block_height", r.activeBlock.Height),
			zap.Uint64("new_active_block_height", height),
		)
	}

	r.activeBlockStartTime = time.Now()
	r.activeBlock = &pbaptos.Block{Height: height}

	return nil
}

// Format:
// FIRE TRX <sf.aptos.type.v1.Transaction>
func (r *ConsoleReader) readTransaction(params []string) error {
	if err := validateChunk(params, 1); err != nil {
		return fmt.Errorf("invalid log line length: %w", err)
	}

	out, err := base64.StdEncoding.DecodeString(params[0])
	if err != nil {
		return fmt.Errorf("read trx: invalid base64 value: %w", err)
	}

	transaction := &pbaptos.Transaction{}
	if err := proto.Unmarshal(out, transaction); err != nil {
		return fmt.Errorf("read trx: invalid proto: %w", err)
	}

	if r.activeBlock == nil {
		return fmt.Errorf("no active block in progress when reading TRX of type %q", transaction.Type)
	}

	if len(r.activeBlock.Transactions) == 0 {
		r.logger.Debug("received first transaction of block, ensuring its a valid first transaction", zap.Uint64("active_block_height", r.activeBlock.Height))

		if !transaction.IsBlockStartBoundaryType() {
			return fmt.Errorf("received first TRX of type %q that is not a valid block start boundary transaction (only Block Metadata and Genesis transaction are)", transaction.Type)
		}

		// Block timestamp is the timestamp of the first transaction (all of the transactions in a block actually share the same timestamp)
		r.activeBlock.Timestamp = transaction.Timestamp
	} else {
		// We already saw the first transaction, ensure we are not seeing again a block start boundary transaction
		if transaction.IsBlockStartBoundaryType() {
			return fmt.Errorf("received non-first block start boundary TRX of type %q, expecting to only ever receive a single block satrt boundary transaction within an active block", transaction.Type)
		}
	}

	r.activeBlock.Transactions = append(r.activeBlock.Transactions, transaction)

	return nil
}

// Format:
// FIRE BLOCK_END <height>
func (r *ConsoleReader) readBlockEnd(params []string) (*pbaptos.Block, error) {
	if err := validateChunk(params, 1); err != nil {
		return nil, fmt.Errorf("invalid BLOCK_END line: %w", err)
	}

	height, err := strconv.ParseUint(params[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf(`invalid BLOCK_END "height" param: %w`, err)
	}

	if r.activeBlock == nil {
		return nil, fmt.Errorf("no active block in progress when reading BLOCK_END")
	}

	if r.activeBlock.Height != height {
		return nil, fmt.Errorf("active block's height %d does not match BLOCK_END received height %d", r.activeBlock.Height, height)
	}

	if len(r.activeBlock.Transactions) == 0 {
		return nil, fmt.Errorf("active block height %d does not contain any transaction", r.activeBlock.Height)
	}

	r.stats.blockRate.Inc()
	r.stats.transactionRate.IncBy(int64(len(r.activeBlock.Transactions)))
	r.stats.blockAverageParseTime.IncByElapsedTime(r.activeBlockStartTime, time.Millisecond)

	r.logger.Debug("console reader read block",
		zap.String("id", r.activeBlock.ID()),
		zap.Uint64("height", r.activeBlock.Height),
		zap.Time("timestamp", r.activeBlock.Timestamp.AsTime()),
	)

	block := r.activeBlock
	r.resetActiveBlock()

	return block, nil
}

func (r *ConsoleReader) resetActiveBlock() {
	r.activeBlock = nil
	r.activeBlockStartTime = time.Time{}
}

func validateChunk(params []string, count int) error {
	if len(params) != count {
		return fmt.Errorf("%d fields required but found %d", count, len(params))
	}
	return nil
}

func lineError(line string, source error) error {
	return fmt.Errorf("%w (on line %q)", source, line)
}
