# Change log

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See [MAINTAINERS.md](./MAINTAINERS.md)
for instructions to keep up to date.

## Unreleased

### Changed

* Updated `--substreams-output-cache-save-interval` default value to 1000.

    > **Warning** The `--substreams-output-cache-save-interval` and `--substreams-stores-save-interval` flags must be the exact same value to avoid bogus behavior with Substreams engine. In a future update, we are going to merge those two flags together in a single flag to avoid any mistake. For now, ensure the two value are identical.

* Updated to Substreams `v0.1.0`, please refer to [release page](https://github.com/streamingfast/substreams/releases/tag/v0.1.0) for further info about Substreams changes.

    > **Warning** The state output format for `map` and `store` modules has changed internally to be more compact in Protobuf format. When deploying this new version and using Substreams feature, previous existing state files should be deleted or deployment updated to point to a new store location. The state output store is defined by the flag `--substreams-state-store-url` flag.

### Fixed

* Fixed `localnet` config for latest `aptos-node`.

## v0.2.0

#### Substreams rust bindings

* Bumped to latest .proto definitions (September 2022)

#### Flags and environment variables

* Renamed `common-one-blocks-store-url` to `common-one-block-store-url`
* Renamed `common-relayer-addr` to `common-live-blocks-addr`
* Renamed the `extractor-node` application to `reader-node`
* Renamed tools command `debug-deep-mind` flags to `debug-firehose-logs`
* Renamed all the `extractor-node-*` flags to `reader-node-*`
* Changed `--log-to-file` default from `true` to `false`
