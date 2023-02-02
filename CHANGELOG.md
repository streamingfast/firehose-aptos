# Change log

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See [MAINTAINERS.md](./MAINTAINERS.md)
for instructions to keep up to date.

## Unreleased

### Added

* Added support for "requester pays" buckets on Google Storage in url, ex: `gs://my-bucket/path?project=my-project-id`

### Changed

* **Breaking** Config value `substreams-stores-save-interval` and `substreams-output-cache-save-interval` have been merged together as a single value to avoid potential bugs that would arise when the value is different for those two. The new configuration value is called `substreams-cache-save-interval`.

    *  To migrate, remove usage of `substreams-stores-save-interval: <number>` and `substreams-output-cache-save-interval: <number>` if defined in your config file and replace with `substreams-cache-save-
interval: <number>`, if you had two different value before, pick the biggest of the two as the new value to put. We are currently setting to `1000` for Ethereum Mainnet.

* Updated to Substreams `v0.2.0`, please refer to [release page](https://github.com/streamingfast/substreams/releases/tag/v0.2.0) for further info about Substreams changes.

* Updated `--substreams-output-cache-save-interval` default value to 1000.

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
