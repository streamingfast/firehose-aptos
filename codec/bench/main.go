package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ShinyTrinkets/overseer"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/cli"
	"github.com/streamingfast/derr"
	"github.com/streamingfast/dmetrics"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/firehose-aptos/codec"
	"github.com/streamingfast/logging"
	"github.com/streamingfast/node-manager/mindreader"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

var decimalRegex = regexp.MustCompile("[0-9]+")

var zlog, tracer = logging.ApplicationLogger("bench", "github.com/streamingfast/firehose-aptos/codec/bench")

func main() {
	validExperiments := map[string]func(string){
		"rawStdin":          rawStdin,
		"blocksStdin":       blocksStdin,
		"systemTrimmedDown": systemTrimmedDown,
	}

	// Not used for now
	feeder := os.Args[1]
	if feeder != "" && feeder != "-" {
		cli.Ensure(cli.FileExists(feeder), "Feeder binary %q does not exist", feeder)
	}

	experiment, found := validExperiments[os.Args[2]]
	cli.Ensure(found, "No experiments named %q found, valid are %q", os.Args[2], strings.Join(maps.Keys(validExperiments), ", "))

	experiment(feeder)
}

func blocksStdin(_ string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 100*1024*1024), 100*1024*1024)

	stdOutBytesCounter := dmetrics.NewPerSecondLocalRateCounter("bytes")
	firelogBytesCounter := dmetrics.NewPerSecondLocalRateCounter("bytes")
	blockRateCounter := dmetrics.NewPerSecondLocalRateCounter("block")

	currentHeadBlock := ""

	lifecycle := shutter.New()

	lines := make(chan string, 10)
	reader, err := codec.NewConsoleReader(zap.NewNop(), lines)
	cli.NoError(err, "Unable to create console reader")

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "FIRE ") {
				lines <- line
				firelogBytesCounter.IncBy(int64(len(line)))
			}

			stdOutBytesCounter.IncBy(int64(len(line)))
		}

		cli.NoError(scanner.Err(), "Scanner terminated unexpectedly")

		close(lines)
		lifecycle.Shutdown(nil)
	}()

	go func() {
		readerLifecycle := shutter.New()
		lifecycle.OnTerminating(func(_ error) {
			// Ensure we wait until reader has finished its job
			<-readerLifecycle.Terminated()
		})

		defer func() {
			// Notify we our done
			readerLifecycle.Shutdown(nil)
		}()

		for {
			block, err := reader.ReadBlock()
			if err != nil {
				if err == io.EOF {
					return
				}

				cli.NoError(err, "Console reader terminated unexpectedly")
			}

			currentHeadBlock = strconv.FormatUint(block.Number, 10)
			blockRateCounter.Inc()
		}
	}()

	printStats := func() {
		fmt.Printf("#%s - Blocks %s, Bytes (Firehose lines) %s, Bytes (All lines) %s\n", currentHeadBlock, blockRateCounter, firelogBytesCounter, stdOutBytesCounter)
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			<-ticker.C
			printStats()
		}
	}()

	select {
	case <-derr.SetupSignalHandler(0):
	case <-lifecycle.Terminated():
	}

	printStats()
	fmt.Println("Completed")
}

func rawStdin(_ string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 100*1024*1024), 100*1024*1024)

	stdOutBytesCounter := dmetrics.NewPerSecondLocalRateCounter("bytes")
	firelogBytesCounter := dmetrics.NewPerSecondLocalRateCounter("bytes")
	blockRateCounter := dmetrics.NewPerSecondLocalRateCounter("block")

	currentHeadBlock := ""

	lifecycle := shutter.New()

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "FIRE ") {
				if strings.HasPrefix(line, "FIRE BLOCK_END") {
					blockRateCounter.Inc()
					currentHeadBlock = decimalRegex.FindString(line)
				}

				firelogBytesCounter.IncBy(int64(len(line)))
			}

			stdOutBytesCounter.IncBy(int64(len(line)))
		}

		if scanner.Err() != nil {
			panic(fmt.Errorf("scanner terminated: %w", scanner.Err()))
		}

		lifecycle.Shutdown(nil)
	}()

	printStats := func() {
		fmt.Printf("#%s - Blocks %s, Bytes (Firehose lines) %s, Bytes (All lines) %s\n", currentHeadBlock, blockRateCounter, firelogBytesCounter, stdOutBytesCounter)
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			<-ticker.C
			printStats()
		}
	}()

	select {
	case <-derr.SetupSignalHandler(0):
	case <-lifecycle.Terminated():
	}

	printStats()
	fmt.Println("Completed")
}

func systemTrimmedDown(feeder string) {
	overseer.DEFAULT_LINE_BUFFER_SIZE = 100 * 1024 * 1024

	cmd := overseer.NewCmd(feeder, overseer.Options{
		Streaming: true,
	})

	lifecycle := shutter.New()

	lines := make(chan string, 10000)
	blocks := make(chan *bstream.Block, 100)

	reader, err := codec.NewConsoleReader(zlog, lines)
	cli.NoError(err, "Unable to create console reader")

	cmd.Stdout = lines

	go func() {
		localOnBlocksStoreURL := path.Join("/tmp/data", "uploadable-oneblock")
		localOneBlocksStore, err := dstore.NewStore(localOnBlocksStoreURL, "dbin", "", false)
		cli.NoError(err, "new local one block store: %w")

		remoteBlocksStoreURL := path.Join("/tmp/data", "final-oneblock")
		remoteBlocksStore, err := dstore.NewStore(remoteBlocksStoreURL, "dbin.zst", "zstd", false)
		cli.NoError(err, "new local one block store: %w")

		archiver := mindreader.NewArchiver(0, "test", localOneBlocksStore, remoteBlocksStore, bstream.GetBlockWriterFactory, zlog, tracer)

		go archiver.Start(context.Background())

		zlog.Info("starting consume flow")

		for {
			block, ok := <-blocks
			cli.Ensure(ok, "Blocks channel ")

			err = archiver.StoreBlock(context.Background(), block)
			cli.NoError(err, "Unable to store block")
		}
	}()

	go func() {
		readerLifecycle := shutter.New()
		lifecycle.OnTerminating(func(_ error) {
			// Ensure we wait until reader has finished its job
			<-readerLifecycle.Terminated()
		})

		defer func() {
			// Notify we our done
			zlog.Info("console reader is done")
			readerLifecycle.Shutdown(nil)
		}()

		for {
			block, err := reader.ReadBlock()
			if err != nil {
				if err == io.EOF {
					return
				}

				cli.NoError(err, "Console reader terminated unexpectedly")
			}

			blocks <- block
		}
	}()

consumeStderr:
	for {
		select {
		case <-cmd.Start():
			status := cmd.Status()
			zlog.Error("Command terminated unxpectecdly", zap.Error(status.Error), zap.Int("exit_code", status.Exit))
		case line := <-cmd.Stderr:
			// Received line
			_ = line

		case <-time.After(60 * time.Second):
			cmd.Stop()
			break consumeStderr
		}
	}

	<-cmd.Done()

	fmt.Println("Completed")
}
