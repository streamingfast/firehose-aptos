package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/streamingfast/cli"
	"github.com/streamingfast/derr"
	"github.com/streamingfast/dmetrics"
	"github.com/streamingfast/firehose-aptos/codec"
	"github.com/streamingfast/logging"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

var decimalRegex = regexp.MustCompile("[0-9]+")

var zlog, _ = logging.ApplicationLogger("bench", "github.com/streamingfast/firehose-aptos/codec/bench")

func main() {
	validExperiments := map[string]func(string){
		"rawStdin":    rawStdin,
		"blocksStdin": blocksStdin,
	}

	// Not used for now
	feeder := os.Args[1]
	// cli.Ensure(cli.FileExists(feeder), "Feeder binary %q does not exist", feeder)

	experiment, found := validExperiments[os.Args[2]]
	cli.Ensure(found, "No experiments named %q found, valid are %q", os.Args[2], strings.Join(maps.Keys(validExperiments), ", "))

	experiment(feeder)
}

func blocksStdin(_ string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 100*1024*1024), 100*1024*1024)

	stdOutBytesCounter := dmetrics.NewLocalRateCounter(time.Second, "bytes")
	firelogBytesCounter := dmetrics.NewLocalRateCounter(time.Second, "bytes")
	blockRateCounter := dmetrics.NewLocalRateCounter(time.Second, "block")

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

	stdOutBytesCounter := dmetrics.NewLocalRateCounter(time.Second, "bytes")
	firelogBytesCounter := dmetrics.NewLocalRateCounter(time.Second, "bytes")
	blockRateCounter := dmetrics.NewLocalRateCounter(time.Second, "block")

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
