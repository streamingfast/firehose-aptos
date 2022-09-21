// Copyright 2021 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/blockstream"
	"github.com/streamingfast/firehose-aptos/codec"
	"github.com/streamingfast/logging"
	nodeManager "github.com/streamingfast/node-manager"
	"github.com/streamingfast/node-manager/mindreader"
	"go.uber.org/zap"
)

func init() {
	registerNode("reader", registerReaderNodeFlags, ReaderNodeManagerAPIAddr)
}

func registerReaderNodeFlags(cmd *cobra.Command) error {
	cmd.Flags().String("reader-node-grpc-listen-addr", ReaderNodeGRPCAddr, "The gRPC listening address to use for serving real-time blocks")
	cmd.Flags().Bool("reader-node-discard-after-stop-num", false, "Ignore remaining blocks being processed after stop num (only useful if we discard the reader data after reprocessing a chunk of blocks)")
	cmd.Flags().String("reader-node-working-dir", "{data-dir}/reader/work", "Path where reader will stores its files")
	cmd.Flags().Uint("reader-node-start-block-num", 0, "Blocks that were produced with smaller block number then the given block num are skipped")
	cmd.Flags().Uint("reader-node-stop-block-num", 0, "Shutdown reader node when we the following 'stop-block-num' has been reached, inclusively.")
	cmd.Flags().Int("reader-node-blocks-chan-capacity", 100, "Capacity of the channel holding blocks read by the reader. Process will shutdown superviser/geth if the channel gets over 90% of that capacity to prevent horrible consequences. Raise this number when processing tiny blocks very quickly")
	cmd.Flags().String("reader-node-one-block-suffix", "default", FlagDescription(`
		Unique identifier for reader node, so that it can produce 'oneblock files' in the same store as another instance without competing
		for writes. You should set this flag if you have multiple reader nodes running, each one should get a unique identifier, the
		hostname value is a good value to use.
	`))

	return nil
}

func getReaderLogPlugin(
	blockStreamServer *blockstream.Server,
	oneBlocksStoreURL string,
	workingDir string,
	batchStartBlockNum uint64,
	batchStopBlockNum uint64,
	blocksChanCapacity int,
	oneBlockFileSuffix string,
	operatorShutdownFunc func(error),
	onLastBlockSeen func(uint64),
	metricsAndReadinessManager *nodeManager.MetricsAndReadinessManager,
	appLogger *zap.Logger,
	appTracer logging.Tracer,
) (*mindreader.MindReaderPlugin, error) {
	if err := makeDirs([]string{workingDir}); err != nil {
		return nil, fmt.Errorf("creating working directory: %w", err)
	}

	consoleReaderFactory := func(lines chan string) (mindreader.ConsolerReader, error) {
		return codec.NewConsoleReader(appLogger, lines)
	}

	plugin, err := mindreader.NewMindReaderPlugin(
		oneBlocksStoreURL,
		workingDir,
		consoleReaderFactory,
		batchStartBlockNum,
		batchStopBlockNum,
		blocksChanCapacity,
		metricsAndReadinessManager.UpdateHeadBlock,
		func(error) {
			operatorShutdownFunc(nil)
		},
		oneBlockFileSuffix,
		blockStreamServer,
		appLogger,
		appTracer,
	)
	if err != nil {
		return nil, fmt.Errorf("new reader plugin: %w", err)
	}

	syncStateFile := filepath.Join(workingDir, "sync_state.json")

	plugin.OnBlockWritten(func(block *bstream.Block) error {
		// It's much faster to serialized to memory than write to file than trying to be clever and keep the file
		// open and seek to top of it which was used before. There is a 100x gain and writing the file in one swift.
		if err := writeNodeSyncState(appLogger, &readerNodeSyncState{BlockNum: block.Num()}, syncStateFile); err != nil {
			return fmt.Errorf("write node sync state: %w", err)
		}

		onLastBlockSeen(block.Num())

		return nil
	})

	return plugin, nil
}

type readerNodeSyncState struct {
	BlockNum uint64 `json:"last_seen_block_num"`

	// Deprecated: There for backward compatibility reading
	Version uint64 `json:"last_seen_version,omitempty"`
}

func readNodeSyncState(logger *zap.Logger, path string) (state *readerNodeSyncState, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	if len(content) == 0 {
		logger.Warn("reader node sync state file content is empty, this is unexpected, resetting sync state to block num 0", zap.String("path", path))
		return &readerNodeSyncState{BlockNum: 0}, nil
	}

	if err := json.Unmarshal(content, &state); err != nil {
		return nil, fmt.Errorf("unmarshal file %q: %w", path, err)
	}

	if state.Version != 0 && state.BlockNum == 0 {
		// Once we remove the deprecated `Version` struct and it's removed, we should remove that (and the `state.Version = 0` below)
		logger.Info("converting wrong 'last_seen_version' field in sync state file to 'last_seen_block_num' (which is accurately represents what was actually stored in 'last_seen_version')")
		state.BlockNum = state.Version
	}

	state.Version = 0

	return state, nil
}

func writeNodeSyncState(logger *zap.Logger, state *readerNodeSyncState, path string) (err error) {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	return os.WriteFile(path, data, os.ModePerm)
}
