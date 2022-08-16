# Firehose on Dummy Blockchain
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/streamingfast/firehose-aptos)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

# Usage

## Setup

1. Build the `aptos-node` binary and bind `aptos-node` to the `PATH`
    - Follow instructions for `devnet`
    - In `aptos-core` repo, run `cd aptos-node` and `cargo install --path .`
1. Run firehose
    ```
    ./devel/localnet/start.sh -c
    ```

    And ensure blocks are flowing:

    ```
    grpcurl -plaintext -import-path ../proto -import-path ./proto -proto sf/aptos/type/v1/type.proto -proto sf/firehose/v2/firehose.proto -d '{"start_block_num": 0}' localhost:18015 sf.firehose.v2.Stream.Blocks
    ```

## Re-generate Protobuf Definitions

1. Ensure that `protoc` is installed:
   ```
   brew install protoc
   ```

1. Ensure that `protoc-gen-go` and `protoc-gen-go-grpc` are installed and at the correct version
    ```
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.25.0
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0
    ```

1. Have a copy of https://github.com/streamingfast/proto cloned sibling of this project:
    ```
    # Assuming you are in `firehose-aptos` root directory directly
    cd ..
    git clone https://github.com/streamingfast/proto.git
    ls
    # Should list both firehose-aptos and proto
    ```

1. Generate go file from modified protobuf

   ```
   ./types/pb/generate.sh
   ```

1. Run tests and fix any problems:

    ```
    ./bin/test.sh
    ```

1. Run `localnet` config to ensure everything is working as expected:

    ```
    ./devel/localnet/start.sh -c
    ```

    And ensure blocks are flowing:

    ```
    grpcurl -plaintext -import-path ../proto -import-path ./proto -proto sf/aptos/type/v1/type.proto -proto sf/firehose/v2/firehose.proto -d '{"start_block_num": 0}' localhost:18015 sf.firehose.v2.Stream.Blocks
    ```

1. (optional) Make .spkg file for substreams
    ```substreams pack substreams/substreams.yaml```
## Release

Use the `./bin/release.sh` Bash script to perform a new release. It will ask you questions
as well as driving all the required commands, performing the necessary operation automatically.
The Bash script runs in dry-mode by default, so you can check first that everything is all right.

Releases are performed using [goreleaser](https://goreleaser.com/).

## Contributing

**Issues and PR in this repo related strictly to the Firehose on Dummy Blockchain.**

Report any protocol-specific issues in their
[respective repositories](https://github.com/streamingfast/streamingfast#protocols)

**Please first refer to the general
[StreamingFast contribution guide](https://github.com/streamingfast/streamingfast/blob/master/CONTRIBUTING.md)**,
if you wish to contribute to this code base.

This codebase uses unit tests extensively, please write and run tests.

## License

[Apache 2.0](LICENSE)
