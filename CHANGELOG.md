# Change log

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See [MAINTAINERS.md](./MAINTAINERS.md)
for instructions to keep up to date.


## v1.0.0

### BREAKING CHANGES

#### Project rename

* The binary name has changed from `sfnear` to `firenear` (aligned with https://firehose.streamingfast.io/references/naming-conventions)
* The repo name has changed from `sf-near` to `firehose-near`

#### Flags and environment variables

* All config via environment variables that started with `SFNEAR_` now starts with `FIRENEAR_`
* All logs now output on *stderr* instead of *stdout* like previously
* Changed `config-file` default from `./sf.yaml` to `""`, preventing failure without this flag.
* Renamed `common-blocks-store-url` to `common-merged-blocks-store-url`
* Renamed `common-oneblock-store-url` to `common-one-block-store-url` *now used by firehose and relayer apps*
* Renamed `common-blockstream-addr` to `common-live-blocks-addr`
* Renamed the `mindreader` application to `reader`
* Renamed `--dm-enabled` to `firehose-enabled`
* Renamed all the `mindreader-node-*` flags to `reader-node-*`
* Added `common-forked-blocks-store-url` flag *used by merger and firehose*
* Changed `--log-to-file` default from `true` to `false`
* Changed default verbosity level: now all loggers are `INFO` (instead of having most of them to `WARN`). `-v` will now activate all `DEBUG` logs
* Removed `common-block-index-sizes`, `common-index-store-url`
* Added `merger-prune-forked-blocks-after`, `merger-time-between-store-pruning`
* Removed `mindreader-node-start-block-num`, `mindreader-node-wait-upload-complete-on-shutdown`, `mindreader-node-merge-and-store-directly`, `mindreader-node-merge-threshold-block-age`
* Removed `firehose-block-index-sizes`,`firehose-block-index-sizes`, `firehose-irreversible-blocks-index-bundle-sizes`, `firehose-irreversible-blocks-index-url`, `firehose-realtime-tolerance`
* Removed `relayer-buffer-size`, `relayer-merger-addr`, `relayer-min-start-offset`
* Removed `firehose-blocks-store-urls` flag (feature for using multiple stores now deprecated -> causes confusion and issues with block-caching), use `common-blocks-store-url` instead.
* Renamed common `atm` 4 flags to `blocks-cache`:
  `--common-blocks-cache-{enabled|dir|max-recent-entry-bytes|max-entry-by-age-bytes}`

#### Producing merged-blocks in batch

* The `reader` requires Firehose-instrumented Geth binary with instrumentation version *2.x* (tagged `fh2`)
* The `reader` *does NOT merge block files directly anymore*: you need to run it alongside a `merger`:
    * determine a `start` and `stop` block for your reprocessing job, aligned on a 100-blocks boundary right after your Geth data snapshot
    * set `--common-first-streamable-block` to your start-block
    * set `--merger-stop-block` to your stop-block
    * set `--common-one-block-store-url` to a local folder accessible to both `merger` and `mindreader` apps
    * set `--common-merged-blocks-store-url` to the final (ex: remote) folder where you will store your merged-blocks
    * run both apps like this `fireeth start reader,merger --...`
* You can run as many batch jobs like this as you like in parallel to produce the merged-blocks, as long as you have data snapshots for Geth that start at this point





