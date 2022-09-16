## `aptos-node` Firehose Throughput Experiment

In this experiment, we want to answer this question:

- What is the maximum throughput of `aptos-node` when emitting Firehose events?

While we answer this question, we are going to make different variants of `aptos-node` to see what has an impact or not on the maximum throughput in `blocks/s` we can get from it.

### Setup

Configuration of `aptos-node`:
- Commit [48cc64df8a64f2d13012c10d8bd5bf25d94f19dc](https://github.com/aptos-labs/aptos-core/commit/48cc64df8a64f2d13012c10d8bd5bf25d94f19dc)
- Compiled with `RUSTFLAGS="-L /opt/homebrew/opt/postgresql/lib --cfg tokio_unstable" cargo build -p aptos-node --release`
- Pre-populated `devnet` instance RocksDB database up to block ~1,100,000.

Configuration of `firehose-aptos`:
- Commit [f1dbd30d0cb7758ca842c9de0f38edece1634812](https://github.com/streamingfast/firehose-aptos/commit/f1dbd30d0cb7758ca842c9de0f38edece1634812)

Some tests will be performed with `aptos-node` untouched which we call `vanilla` for the rest of the document. Others will make manual tweak in the code, diff for those variants are attached in the section that describes each variant.

For all the tests, we launch `aptos-node` with the following command and with this [config.yaml](#aptos-node-config-file):

```
RUST_LOG=error STARTING_BLOCK=0 aptos-node --config config.yaml
```

Its output is then piped to `/dev/null` or to a reading process in Golang depending on the test case.

In all the tests, we added a slight change to `aptos-node` so that it exits right away when firehose indexer reaches block #75,001, we expect that to have no consequences of any of the results presented here.

### Results

| | `aptos-node` vanilla | [`aptos-node` tweaked to only emit `FIRE BLOCK_END` line but keeps transactions Protobuf encoding without emitting them](#aptos-node-tweaked-to-only-emit-fire-blockend-line-but-keeps-transactions-protobuf-encoding-without-emitting-them)| [`aptos-node` tweaked to only emit `FIRE BLOCK_END` line without Protobuf encoding](#aptos-node-tweaked-to-only-emit-fire-blockend-line-without-protobuf-encoding)|
|-|-|-|-|
|[`stdout` to /dev/null](#stdout-to-devnull)| `963.89 blocks/s` | `994.04 blocks/s` | `995.48 blocks/s` |
|[`stdout` to a Go program reading it and then discard it](#stdout-to-a-go-program-reading-it-and-then-discard-it)| `972.64 blocks/s` | `990.36 blocks/s` | `991.54 blocks/s` |
|[`stdout` to a Go program reading and decoding it through Firehose Console Reader element](#stdout-to-a-go-program-reading-and-decoding-it-through-firehose-console-reader-element)| `965.25 blocks/s` | N/A | N/A |

> In the [`stdout` to a Go program reading and decoding it through Firehose Console Reader element](#stdout-to-a-go-program-reading-and-decoding-it-through-firehose-console-reader-element) case, the two variants on `aptos-node` are not available because Firehose Console Reader expects well formed logs which is not the case when only `FIRE BLOCK_END` is emitted.

#### Graph

![](./graph_aptos_node_throughput.png)

Here the legend and the actual:

- `vanilla read by raw stdin` is `aptos-node` vanilla with [`stdout` to a Go program reading it and then discard it](#stdout-to-a-go-program-reading-it-and-then-discard-it)
- `vanilla read by blocks stdin` is `aptos-node` vanilla with [`stdout` to a Go program reading it and then discard it](#stdout-to-a-go-program-reading-it-and-then-discard-it)
- `limited stdout (with proto encoding) read by raw stdin` is [`aptos-node` tweaked to only emit `FIRE BLOCK_END` line but keeps transactions Protobuf encoding without emitting them](#aptos-node-tweaked-to-only-emit-fire-blockend-line-but-keeps-transactions-protobuf-encoding-without-emitting-them) with [`stdout` to a Go program reading it and then discard it](#stdout-to-a-go-program-reading-it-and-then-discard-it)
- `limited stdout (without proto encoding) read by raw stdin` [`aptos-node` tweaked to only emit `FIRE BLOCK_END` line without Protobuf encoding](#aptos-node-tweaked-to-only-emit-fire-blockend-line-without-protobuf-encoding) is with [`stdout` to a Go program reading it and then discard it](#stdout-to-a-go-program-reading-it-and-then-discard-it)

> This graph does not include the any results for [`stdout` to /dev/null](#stdout-to-devnull) because it was not sampled each second like the others were.

### Analysis

Overall, we can see there is no significative differences between the test. Removing base64, Protobuf and almost all `stdout` writing does not have any big effect of performance throughput.

Introducing a Golang program reading from `stdout` and either processing or discarding the Firehose logs did not change that much neither.

It seems this is the maximum current throughput we could obtain from `aptos-node` right now. Changing the exchange format or the transport (IPC, Named Pipe) would probably bring no significant differences neither because the current bottleneck is not right now in encoding and sending data to `stdout`

### Test details

#### `aptos-node` config file

In this config file:
- `<workdir>/data` contains an `aptos-node` `db` folder that was already synced to block ~1.1M.
- `<workdir>/data/waypoint.txt` is the `devnet` `waypoint.txt` as provided by Aptos
- `<workdir>/data/genesis.blob` is the `devnet` `genesis.blob` as provided by Aptos

```
base:
    # Update this value to the location you want the node to store its database
    data_dir: "<workdir>/data"
    role: "full_node"
    waypoint:
        # Update this value to that which the blockchain publicly provides. Please regard the directions
        # below on how to safely manage your genesis_file_location with respect to the waypoint.
        from_file: "<workdir>/data/waypoint.txt"

execution:
    # Update this to the location to where the genesis.blob is stored, prefer fullpaths
    # Note, this must be paired with a waypoint. If you update your waypoint without a
    # corresponding genesis, the file location should be an empty path.
    genesis_file_location: "<workdir>/data/genesis.blob"

full_node_networks:
    - discovery_method: "onchain"
      # The network must have a listen address to specify protocols. This runs it locally to
      # prevent remote, incoming connections.
      listen_address: "/ip4/127.0.0.1/tcp/6180"
      network_id: "public"
      # Define the upstream peers to connect to
      seeds:
        {}

firehose_stream:
  enabled: true
  starting_version: 0

storage:
  enable_indexer: true

api:
    enabled: true
    address: 127.0.0.1:8080
```


#### `aptos-node` tweaked to only emit `FIRE BLOCK_END` line but keeps transactions Protobuf encoding without emitting them

In this mode, we tweaked `aptos-node` removing emitting of `FIRE BLOCK_START` and `FIRE TRX` lines. Removing `FIRE TRX` removes the `base64` encoder call. However in this test we kept the `Protobuf` encoding and made sure it was not optimized away by the compiler.

In this test, we want to check what impact `stdout` has, for example does it creates excessive blocking so that `aptos-node` is throttled? We also eliminate the `base64` encoding trying to see if both measures increase the throughput.

```
diff --git a/ecosystem/sf-indexer/firehose-stream/src/runtime.rs b/ecosystem/sf-indexer/firehose-stream/src/runtime.rs
index 28fa13d264..e3502b23b3 100644
--- a/ecosystem/sf-indexer/firehose-stream/src/runtime.rs
+++ b/ecosystem/sf-indexer/firehose-stream/src/runtime.rs
@@ -147,7 +147,7 @@ impl FirehoseStreamer {
         // We are validating the block as we convert and print each transactions. The rules are as follows:
         // 1. first (and only first) transaction is a block metadata or genesis 2. versions are monotonically increasing 3. start and end versions match block boundaries
         // Retry if the block is not valid. Panic if there's anything wrong with encoding a transaction.
-        println!("\nFIRE BLOCK_START {}", self.current_block_height);
+        // println!("\nFIRE BLOCK_START {}", self.current_block_height);

         let transactions = match self.context.get_transactions(
             block_start_version,
@@ -262,7 +262,11 @@ impl FirehoseStreamer {
                 transaction
             )
         });
-        println!("\nFIRE TRX {}", base64::encode(buf));
-        metrics::TRANSACTIONS_SENT.inc();
+        // println!("\nFIRE TRX {}", base64::encode(buf));
+
+        // There just to ensure compiler does not optimize away `buf` encoding, has no impact on peformance
+        if buf.len() > 0 {
+            metrics::TRANSACTIONS_SENT.inc();
+        }
     }
 }
```

#### `aptos-node` tweaked to only emit `FIRE BLOCK_END` line without Protobuf encoding

In this mode, we tweaked `aptos-node` removing emitting of `FIRE BLOCK_START` and `FIRE TRX` lines. Removing `FIRE TRX` removes the `base64` encoder call and we also eliminate the Protobuf encoding here.

In this test, we want to check mainly was is the impact of Protobuf encoding, this will be compared more against previous test which kept Protobuf encoding.

```
diff --git a/ecosystem/sf-indexer/firehose-stream/src/runtime.rs b/ecosystem/sf-indexer/firehose-stream/src/runtime.rs
index 28fa13d264..077f116dd5 100644
--- a/ecosystem/sf-indexer/firehose-stream/src/runtime.rs
+++ b/ecosystem/sf-indexer/firehose-stream/src/runtime.rs
@@ -14,8 +14,7 @@ use aptos_types::chain_id::ChainId;
 use aptos_vm::data_cache::StorageAdapterOwned;
 use extractor::Transaction as TransactionPB;
 use futures::channel::mpsc::channel;
-use prost::Message;
-use std::{convert::TryInto, sync::Arc, time::Duration};
+use std::{convert::TryInto, process, sync::Arc, time::Duration};
 use storage_interface::{state_view::DbStateView, DbReader};
 use tokio::{
     runtime::{Builder, Runtime},
@@ -147,7 +146,7 @@ impl FirehoseStreamer {
         // We are validating the block as we convert and print each transactions. The rules are as follows:
         // 1. first (and only first) transaction is a block metadata or genesis 2. versions are monotonically increasing 3. start and end versions match block boundaries
         // Retry if the block is not valid. Panic if there's anything wrong with encoding a transaction.
-        println!("\nFIRE BLOCK_START {}", self.current_block_height);
+        // println!("\nFIRE BLOCK_START {}", self.current_block_height);

         let transactions = match self.context.get_transactions(
             block_start_version,
@@ -242,6 +241,11 @@ impl FirehoseStreamer {
         println!("\nFIRE BLOCK_END {}", self.current_block_height);
         metrics::BLOCKS_SENT.inc();
         self.current_block_height += 1;
+
+        if self.current_block_height == 75001 {
+            process::exit(0);
+        }
+
         result
     }

@@ -254,15 +258,16 @@ impl FirehoseStreamer {
         is_first_txn == is_bm_or_genesis
     }

-    fn print_transaction(&self, transaction: &TransactionPB) {
-        let mut buf = vec![];
-        transaction.encode(&mut buf).unwrap_or_else(|_| {
-            panic!(
-                "Could not convert protobuf transaction to bytes '{:?}'",
-                transaction
-            )
-        });
-        println!("\nFIRE TRX {}", base64::encode(buf));
+    fn print_transaction(&self, _: &TransactionPB) {
+        // let mut buf = vec![];
+        // transaction.encode(&mut buf).unwrap_or_else(|_| {
+        //     panic!(
+        //         "Could not convert protobuf transaction to bytes '{:?}'",
+        //         transaction
+        //     )
+        // });
+        // println!("\nFIRE TRX {}", base64::encode(buf));
+
         metrics::TRANSACTIONS_SENT.inc();
     }
 }
```

#### `stdout` to `/dev/null`

We will try to get a measure here `stdout` is mostly discarded right away the reading part of the data emitted to it has no consequence. It's the last amount of blocking we can get from the reading part.

Actual execution:

```
time RUST_LOG=error STARTING_BLOCK=0 aptos-node --config /Users/maoueh/work/sf/firehose-aptos/devel/devnet/firehose-data/extractor/data/node.yaml > /dev/null
```

And to obtain the average speed we divide 75000 by the clock elapsed time reported, for example if 78s is reported, we do `75000 / 78s` and we obtain `~961 blocks/s`.

#### `stdout` to a Go program reading it and then discard it

This adds some blocking as the Golang process needs to read from supervised process' `stdout` which creates some blocking on the `aptos-node` side which is trying to write to it. This test should showcase the differences of adding a Golang consuming process in the equation.

Code can be seen: https://github.com/streamingfast/firehose-aptos/blob/main/codec/bench/main.go#L125

Actual execution:

```
go build -o /tmp/bench ./codec/bench && time RUST_LOG=error STARTING_BLOCK=0 aptos-node --config /Users/maoueh/work/sf/firehose-aptos/devel/devnet/firehose-data/extractor/data/node.yaml | /tmp/bench "-" rawStdin | tee codec/bench/results_raw_stdin.log
```

The output is in the form:

```
...
#71016 - Blocks 1040 block/s (71017 total), Bytes (Firehose lines) 10464016 bytes/s (786574108 total), Bytes (All lines) 10014020 bytes/s (786628163 total)
#72296 - Blocks 1221 block/s (72297 total), Bytes (Firehose lines) 10415570 bytes/s (797022871 total), Bytes (All lines) 9941770 bytes/s (797076926 total)
#73475 - Blocks 1120 block/s (73476 total), Bytes (Firehose lines) 10722080 bytes/s (807786309 total), Bytes (All lines) 10763438 bytes/s (807840364 total)
#74805 - Blocks 1266 block/s (74806 total), Bytes (Firehose lines) 10301673 bytes/s (818125509 total), Bytes (All lines) 10334918 bytes/s (818179564 total)
#75000 - Blocks 1264 block/s (75001 total), Bytes (Firehose lines) 9680934 bytes/s (819563389 total), Bytes (All lines) 9718042 bytes/s (819617444 total)
Completed
RUST_LOG=error STARTING_BLOCK=0 aptos-node --config   76.92s user 1.71s system 100% cpu 1:18.37 total
```

And we can extract statistics about using StreamingFast's `stats` tool and some bash-fu:

```
cat codec/bench/results_raw_stdin.log | grep "block/s" | cut -d ' ' -f 4 | stats -u " blocks/s"
Count: 78
Range: Min 727.00000 blocks/s - Max 2397.00000 blocks/s
Sum: 74963.00000 blocks/s
Average: 961.06410 blocks/s
Median: 897.00000 blocks/s
Standard Deviation: 229.74228 blocks/s
```

In the report section, we kept only `Average` from this metric. The `time` metrics can be obtained by dividing 75000 by amount of it required to reach it.

#### `stdout` to a Go program reading and decoding it through Firehose Console Reader element

This is a variation of previous test where we now added an actual processing of the read data from `aptos-node`. We pass through the Firehose Console Reader which validates the received input and also decodes it as well as creating a full Firehose `pbaptos.Block` model in Protobuf format that is then serialized and packed in a `*bstream.Block` (contains block's metadata + encoded Protobuf payload in `bytes`).

Code can be seen: https://github.com/streamingfast/firehose-aptos/blob/main/codec/bench/main.go#L43

Actual execution:

```
go build -o /tmp/bench ./codec/bench && time RUST_LOG=error STARTING_BLOCK=0 aptos-node --config /Users/maoueh/work/sf/firehose-aptos/devel/devnet/firehose-data/extractor/data/node.yaml | /tmp/bench "-" blocksStdin | tee codec/bench/results_blocks_stdin.log
```

The output is in the form:

```
...
#71016 - Blocks 1040 block/s (71017 total), Bytes (Firehose lines) 10464016 bytes/s (786574108 total), Bytes (All lines) 10014020 bytes/s (786628163 total)
#72296 - Blocks 1221 block/s (72297 total), Bytes (Firehose lines) 10415570 bytes/s (797022871 total), Bytes (All lines) 9941770 bytes/s (797076926 total)
#73475 - Blocks 1120 block/s (73476 total), Bytes (Firehose lines) 10722080 bytes/s (807786309 total), Bytes (All lines) 10763438 bytes/s (807840364 total)
#74805 - Blocks 1266 block/s (74806 total), Bytes (Firehose lines) 10301673 bytes/s (818125509 total), Bytes (All lines) 10334918 bytes/s (818179564 total)
#75000 - Blocks 1264 block/s (75001 total), Bytes (Firehose lines) 9680934 bytes/s (819563389 total), Bytes (All lines) 9718042 bytes/s (819617444 total)
Completed
RUST_LOG=error STARTING_BLOCK=0 aptos-node --config   76.92s user 1.71s system 100% cpu 1:18.37 total
```

And we can extract statistics about using StreamingFast's `stats` tool and some bash-fu:

```
cat codec/bench/results_raw_stdin.log | grep "block/s" | cut -d ' ' -f 4 | stats -u " blocks/s"
Count: 78
Range: Min 727.00000 blocks/s - Max 2397.00000 blocks/s
Sum: 74963.00000 blocks/s
Average: 961.06410 blocks/s
Median: 897.00000 blocks/s
Standard Deviation: 229.74228 blocks/s
```

In the report section, we kept only `Average` from this metric. The `time` metrics can be obtained by dividing 75000 by amount of it required to reach it.
