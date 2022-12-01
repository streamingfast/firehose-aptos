# Change log

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See [MAINTAINERS.md](./MAINTAINERS.md)
for instructions to keep up to date.

## Unreleased

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
