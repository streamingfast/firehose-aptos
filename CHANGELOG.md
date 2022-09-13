# Change log

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See [MAINTAINERS.md](./MAINTAINERS.md)
for instructions to keep up to date.


## v1.0.0


#### Flags and environment variables

* Renamed `common-blocks-store-url` to `common-merged-blocks-store-url`
* Renamed `common-oneblock-store-url` to `common-one-block-store-url` *now used by firehose and relayer apps*
* Renamed `common-relayer-addr` to `common-live-blocks-addr`
* Renamed the `mindreader` application to `reader`
* Renamed `debug-deep-mind` flags to `debug-firehose-logs`
* Renamed `extractor` to `reader`
* Renamed all the `mindreader-node-*` flags to `reader-node-*`
* Added `common-forked-blocks-store-url` flag *used by merger and firehose*
* Changed `--log-to-file` default from `true` to `false`
* Renamed common `atm` 4 flags to `blocks-cache`:
  `--common-blocks-cache-{enabled|dir|max-recent-entry-bytes|max-entry-by-age-bytes}`





