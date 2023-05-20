package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream/hub"
	dauthAuthenticator "github.com/streamingfast/dauth/authenticator"
	dgrpcserver "github.com/streamingfast/dgrpc/server"
	"github.com/streamingfast/dlauncher/launcher"
	"github.com/streamingfast/dmetering"
	"github.com/streamingfast/dmetrics"
	"github.com/streamingfast/dstore"
	firehoseApp "github.com/streamingfast/firehose/app/firehose"
	"github.com/streamingfast/logging"
	substreamsClient "github.com/streamingfast/substreams/client"
	substreamsService "github.com/streamingfast/substreams/service"
	"go.uber.org/zap"
)

var metricset = dmetrics.NewSet()
var headBlockNumMetric = metricset.NewHeadBlockNumber("firehose")
var headTimeDriftmetric = metricset.NewHeadTimeDrift("firehose")

func init() {
	appLogger, _ := logging.PackageLogger("firehose", "github.com/streamingfast/firehose-aptos/firehose")

	launcher.RegisterApp(rootLog, &launcher.AppDef{
		ID:          "firehose",
		Title:       "Block Firehose",
		Description: "Provides on-demand filtered blocks, depends on common-merged-blocks-store-url and common-live-blocks-addr",
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("firehose-grpc-listen-addr", FirehoseGRPCServingAddr, "Address on which the firehose will listen")
			cmd.Flags().Duration("firehose-real-time-tolerance", 1*time.Minute, "firehose will became alive if now - block time is smaller then tolerance")

			cmd.Flags().Bool("substreams-enabled", false, "Whether to enable substreams")
			cmd.Flags().Bool("substreams-tier2", false, "Whether this endpoint is serving tier2 requests (non-public-facing)")
			cmd.Flags().String("substreams-state-store-url", "{data-dir}/localdata", "where substreams state data are stored")
			cmd.Flags().Uint64("substreams-cache-save-interval", uint64(1_000), "Interval in blocks at which to save store snapshots and output caches")
			cmd.Flags().Int("substreams-parallel-subrequest-limit", 4, "number of parallel subrequests substream can make to synchronize its stores")
			cmd.Flags().String("substreams-client-endpoint", "", "firehose endpoint for substreams client. If empty, this endpoint will also serve its own internal tier2 requests")
			cmd.Flags().String("substreams-client-jwt", "", "JWT for substreams client authentication")
			cmd.Flags().Bool("substreams-client-insecure", false, "Substreams client in insecure mode")
			cmd.Flags().Bool("substreams-client-plaintext", true, "Substreams client in plaintext mode")
			cmd.Flags().Int("substreams-sub-request-parallel-jobs", 5, "Substreams subrequest parallel jobs for the scheduler")
			cmd.Flags().Int("substreams-sub-request-block-range-size", 1000, "Substreams subrequest block range size value for the scheduler")
			return nil
		},

		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			sfDataDir := runtime.AbsDataDir

			authenticator, err := dauthAuthenticator.New(viper.GetString("common-auth-plugin"))
			if err != nil {
				return nil, fmt.Errorf("unable to initialize dauth: %w", err)
			}

			metering, err := dmetering.New(viper.GetString("common-metering-plugin"))
			if err != nil {
				return nil, fmt.Errorf("unable to initialize dmetering: %w", err)
			}
			dmetering.SetDefaultMeter(metering)
			firehoseGRPCListenAddr := viper.GetString("firehose-grpc-listen-addr")

			var registerServiceExt firehoseApp.RegisterServiceExtensionFunc
			if viper.GetBool("substreams-enabled") {
				stateStore, err := dstore.NewStore(MustReplaceDataDir(sfDataDir, viper.GetString("substreams-state-store-url")), "", "", true)
				if err != nil {
					return nil, fmt.Errorf("setting up state store for data: %w", err)
				}

				opts := []substreamsService.Option{
					substreamsService.WithCacheSaveInterval(viper.GetUint64("substreams-cache-save-interval")),
				}

				clientEndpoint := viper.GetString("substreams-client-endpoint")

				var runTier1, runTier2 bool
				if viper.GetBool("substreams-tier2") {
					runTier2 = true
				} else {
					runTier1 = true
				}

				if clientEndpoint == "" {
					runTier2 = true // self-contained deployment: run tier2 for our own tier1
					clientEndpoint = firehoseGRPCListenAddr
				}

				clientConfig := substreamsClient.NewSubstreamsClientConfig(
					clientEndpoint,
					os.ExpandEnv(viper.GetString("substreams-client-jwt")),
					viper.GetBool("substreams-client-insecure"),
					viper.GetBool("substreams-client-plaintext"),
				)

				var tier1 *substreamsService.Tier1Service
				var tier2 *substreamsService.Tier2Service

				if runTier1 {
					tier1, err = substreamsService.NewTier1(
						stateStore,
						"aptos.extractor.v1.Block",
						uint64(viper.GetInt("substreams-sub-request-parallel-jobs")),
						uint64(viper.GetInt("substreams-sub-request-block-range-size")),
						clientConfig,
						opts...,
					)
					if err != nil {
						return nil, fmt.Errorf("create substreams service: %w", err)
					}
				}
				if runTier2 {
					tier2 = substreamsService.NewTier2(
						stateStore,
						"aptos.extractor.v1.Block",
						opts...,
					)
				}

				registerServiceExt = func(
					server dgrpcserver.Server,
					mergedBlocksStore dstore.Store,
					forkedBlocksStore dstore.Store, // this can be nil here
					forkableHub *hub.ForkableHub,
					logger *zap.Logger,
				) {
					if tier1 != nil {
						tier1.Register(server, mergedBlocksStore, forkedBlocksStore, forkableHub, logger)
					}
					if tier2 != nil {
						tier2.Register(server, mergedBlocksStore, forkedBlocksStore, forkableHub, logger)
					}
				}
			}

			return firehoseApp.New(appLogger, &firehoseApp.Config{
				OneBlocksStoreURL:       MustReplaceDataDir(sfDataDir, viper.GetString("common-one-block-store-url")),
				MergedBlocksStoreURL:    MustReplaceDataDir(sfDataDir, viper.GetString("common-merged-blocks-store-url")),
				ForkedBlocksStoreURL:    MustReplaceDataDir(sfDataDir, viper.GetString("common-forked-blocks-store-url")),
				BlockStreamAddr:         viper.GetString("common-live-blocks-addr"),
				GRPCListenAddr:          firehoseGRPCListenAddr,
				GRPCShutdownGracePeriod: 1 * time.Second,
			}, &firehoseApp.Modules{
				Authenticator:            authenticator,
				HeadTimeDriftMetric:      headTimeDriftmetric,
				HeadBlockNumberMetric:    headBlockNumMetric,
				RegisterServiceExtension: registerServiceExt,
			}), nil
		},
	})
}
