# Firehose on Dummy Blockchain
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/streamingfast/firehose-aptos)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

# Usage

## Setup
1. Build the aptos node and bind aptos-node to the path
    a. Follow instructions for devnet
    b. In aptos-core repo, run `cd aptos-node` and `cargo install --path .`
2. Update devel/devnet/config/full_node.yaml with correct configs (back in firehose-aptos repo)
    a. Update data_dir, from_file, and genesis_file_location with correct paths. 
    b. Make sure that api address is correct
3. Build spkg file
    a. install substreams
    b. `substreams pack substreams/substreams.yaml`
4. Run firehose
    a. `./devel/localnet/start.sh -c`

## How to modify protobuf
1. Ensure that protoc and protoc-gen-go are installed
    a. `brew install protoc && brew install protoc-gen-go && brew install protoc-gen-go-grpc`
2. Generate go file from modified protobuf
    a. `types/pb/generate.sh`
3. Run firehose
    a.`./devel/localnet/start.sh -c`

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
