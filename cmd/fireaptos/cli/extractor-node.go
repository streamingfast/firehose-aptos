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
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/blockstream"
	"github.com/streamingfast/firehose-aptos/nodemanager/codec"
	"github.com/streamingfast/logging"
	nodeManager "github.com/streamingfast/node-manager"
	"github.com/streamingfast/node-manager/mindreader"
	"go.uber.org/zap"
)

func init() {
	registerNode("extractor", registerExtractorNodeFlags, ExtractorNodeManagerAPIAddr)
}

func registerExtractorNodeFlags(cmd *cobra.Command) error {
	cmd.Flags().String("extractor-node-grpc-listen-addr", ExtractorNodeGRPCAddr, "The gRPC listening address to use for serving real-time blocks")
	cmd.Flags().Bool("extractor-node-discard-after-stop-num", false, "Ignore remaining blocks being processed after stop num (only useful if we discard the extractor data after reprocessing a chunk of blocks)")
	cmd.Flags().String("extractor-node-working-dir", "{data-dir}/extractor/work", "Path where extractor will stores its files")
	cmd.Flags().Uint("extractor-node-start-block-num", 0, "Blocks that were produced with smaller block number then the given block num are skipped")
	cmd.Flags().Uint("extractor-node-stop-block-num", 0, "Shutdown extractor when we the following 'stop-block-num' has been reached, inclusively.")
	cmd.Flags().Int("extractor-node-blocks-chan-capacity", 100, "Capacity of the channel holding blocks read by the extractor. Process will shutdown superviser/geth if the channel gets over 90% of that capacity to prevent horrible consequences. Raise this number when processing tiny blocks very quickly")
	cmd.Flags().String("extractor-node-one-block-suffix", "default", FlagDescription(`
		Unique identifier for extractor, so that it can produce 'oneblock files' in the same store as another instance without competing
		for writes. You should set this flag if you have multiple extractor running, each one should get a unique identifier, the
		hostname value is a good value to use.
	`))

	return nil
}

func getMindreaderLogPlugin(
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

	syncStateFile := filepath.Join(workingDir, "sync_state.json")

	// We never close the sync state file but I don't think it's a problem. It's meant
	// to be ever open so that's not a problem. There is still risk of an abrupt process
	// closing that would close the file descriptor in the middle of a write. But even there,
	// we write so less data in bytes that I'm pretty sure the write is going to be done in atomic
	// fashion in one swift so we are confident that doing how we do it is safe here.
	syncStateFileWriter, err := os.Create(syncStateFile)
	if err != nil {
		return nil, fmt.Errorf("open sync state file for writing: %w", err)
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
		return nil, fmt.Errorf("new mindreader plugin: %w", err)
	}

	plugin.OnBlockWritten(func(block *bstream.Block) error {
		if _, err := syncStateFileWriter.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("reset sync state file to offset 0: %w", err)
		}

		err := json.NewEncoder(syncStateFileWriter).Encode(extractorNodeSyncState{
			// FIXME: Right now we have the real version because our "Block" are actual Aptos transaction
			// but if we change so that a `Block` becomes a set of transactions, then we need to change
			// here.
			Version: block.Num(),
		})
		if err != nil {
			return fmt.Errorf("encode sync state to file: %w", err)
		}

		onLastBlockSeen(block.Num())

		return nil
	})

	return plugin, nil
}

type extractorNodeSyncState struct {
	Version uint64 `json:"last_seen_version"`
}

func readNodeSyncState(path string) (state *extractorNodeSyncState, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	if err := json.Unmarshal(content, &state); err != nil {
		return nil, fmt.Errorf("unmarshal file: %w", err)
	}

	return state, nil
}
