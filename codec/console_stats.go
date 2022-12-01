package codec

import (
	"context"
	"time"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dmetrics"
	"go.uber.org/zap"
)

type consoleReaderStats struct {
	lastBlock             bstream.BlockRef
	blockRate             *dmetrics.RateCounter
	blockAverageParseTime *dmetrics.AvgDurationCounter
	transactionRate       *dmetrics.RateCounter

	cancelPeriodicLogger context.CancelFunc
}

func newConsoleReaderStats() *consoleReaderStats {
	return &consoleReaderStats{
		lastBlock:             bstream.BlockRefEmpty,
		blockRate:             dmetrics.NewPerSecondLocalRateCounter("blocks"),
		blockAverageParseTime: dmetrics.NewAvgDurationCounter(5*time.Second, time.Millisecond, "ms/block"),
		transactionRate:       dmetrics.NewPerSecondLocalRateCounter("trxs"),
	}
}

func (s *consoleReaderStats) StartPeriodicLogToZap(ctx context.Context, logger *zap.Logger, logEach time.Duration) {
	ctx, s.cancelPeriodicLogger = context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(logEach)
		for {
			select {
			case <-ticker.C:
				logger.Info("reader node statistics", s.ZapFields()...)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *consoleReaderStats) StopPeriodicLogToZap() {
	if s.cancelPeriodicLogger != nil {
		s.cancelPeriodicLogger()
	}
}

func (s *consoleReaderStats) ZapFields() []zap.Field {
	return []zap.Field{
		zap.Stringer("block_rate", s.blockRate),
		zap.Stringer("trx_rate", s.transactionRate),
		zap.Stringer("last_block", s.lastBlock),
		zap.Stringer("block_average_parse_time", s.blockAverageParseTime),
	}
}
