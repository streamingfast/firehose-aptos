module github.com/streamingfast/firehose-aptos

go 1.16

require (
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/abourget/llerrgroup v0.2.0 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/spf13/cobra v1.4.0
	github.com/spf13/viper v1.8.1
	github.com/streamingfast/bstream v0.0.2-0.20220728190336-c25798bc7ae3
	github.com/streamingfast/cli v0.0.4-0.20220113202443-f7bcefa38f7e
	github.com/streamingfast/dauth v0.0.0-20220307162109-cca1810ae757
	github.com/streamingfast/dbin v0.0.0-20210809205249-73d5eca35dc5
	github.com/streamingfast/derr v0.0.0-20220307162255-f277e08753fa
	github.com/streamingfast/dgrpc v0.0.0-20220307180102-b2d417ac8da7
	github.com/streamingfast/dlauncher v0.0.0-20220307153121-5674e1b64d40
	github.com/streamingfast/dmetering v0.0.0-20220307162406-37261b4b3de9
	github.com/streamingfast/dmetrics v0.0.0-20220307162521-2389094ab4a1
	github.com/streamingfast/dstore v0.1.1-0.20220607202639-35118aeaf648
	github.com/streamingfast/dtracing v0.0.0-20220305214756-b5c0e8699839 // indirect
	github.com/streamingfast/firehose v0.1.1-0.20220727195340-e08fa990c0bc
	github.com/streamingfast/logging v0.0.0-20220304214715-bc750a74b424
	github.com/streamingfast/merger v0.0.3-0.20220728182037-7b5841d3c98f
	github.com/streamingfast/node-manager v0.0.2-0.20220728184759-72d699d3450f
	github.com/streamingfast/pbgo v0.0.6-0.20220630154121-2e8bba36234e
	github.com/streamingfast/relayer v0.0.2-0.20220726150209-4341fcef75ec
	github.com/streamingfast/sf-tools v0.0.0-20220727183125-3348eaca0e25
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.21.0
	google.golang.org/grpc v1.44.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/ShinyTrinkets/overseer => github.com/streamingfast/overseer v0.2.1-0.20210326144022-ee491780e3ef
