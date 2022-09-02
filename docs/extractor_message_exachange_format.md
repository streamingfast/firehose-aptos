## reader Message Exchange Format

This documents outlines what is the actual format for messages exchanges between the instrumented process and the Firehose stack that is consuming those message to feed them in the block flow.

### Overview

Here a quick example of the output format, you will find below extra details

```
FIRE INIT aptos-node 0.1.0 aptos 0 0
FIRE BLOCK_START <height>
FIRE TRX <sf.aptos.types.v1.TransactionTrace>
FIRE TRX <sf.aptos.types.v1.TransactionTrace>
FIRE TRX <sf.aptos.types.v1.TransactionTrace>
...
FIRE BLOCK_END <height>
```

The `<sf.aptos.types.v1.TransactionTrace>` and any other Protobuf structure are encoded with base64 standard with padding.

### Standard

Each message are form in a single line of UTF-8 text characters, each line must be terminated by a single line ending terminal symbol specific to the platform we currently runs on, e.g. `\n` on Unix and OS X, `\r\n` on Windows.

Each line must be prefixed by `FIRE ` to add some validation to all messages received.

Each message **not** prefixed by `FIRE ` received must be ignored by the reader.

Each "parameter" element in the line must separated by a space character.

Each "parameter" element in the line must written in string form **without** double-quotes, so a pure string like `bob` is written the same way as the number `10` respectively `FIRE TRX bob` and `FIRE TRX 10`.

#### `INIT`

```
FIRE INIT <client_name> <client_version> <fork_name> <firehose_major> <firehose_minor>
```

The first message that should be sent is the `FIRE INIT` message that is used to ensure the reader is able to handle the format the instrumented process is about to send to use. The `INIT` messages is also used to exchange some basic information like the client's name, client's version, fork name and Firehose major and minor version.

And here the description of each parameters:

|Name|Description|
|-|-|
|`<client_name>`|Should be the name of the client that is instrumented, in our case a single client exist so it should be `aptos-node`|
|`<client_version>`|Should be the triplet version of the compiled client|
|`<fork_name>`|Should be the name of the actual fork of the client if relevant. For example, on Ethereum world, Polygon is a fork of Geth, in this case `<client_name>` would be `geth` while `<fork_name>` would be `polygon`. Right now, `aptos` has no known fork so the `aptos` string should be hard-coded here|
|`<firehose_major>`|The major version of the specification currently implemented by the instrumented process, right now we will use `0` until further re-consideration. This can be used by consumer to determine if they are capable of handling the stream that is going to be produced by the instrumented binary|
|`<firehose_minor>`|The minor version of this specification, right now we will use `0` until further re-consideration. Right now, we should start with `0` and continue forward each time some buf fixes are made in the instrumentation without changing the format|

##### Considerations

Every messages that fits the `FIRE` format received before the `FIRE INIT` message must be ignored.



#### `TRX`

```
FIRE TRX <sf.aptos.types.v1.Transaction>
```

Issue after the intial `INIT` message for each and single transaction of any types happening on the network. The single and only parameter should be the fully constructed Protobuf object of type [sf.aptos.types.v1.Transaction](../proto/sf/aptos/type/v1/type.proto#L11), bytes encoded and serialized to base64 standard with padding.

Final line would look like:

```
FIRE TRX 1QChXYTVkjypOUcHZjvk_wbLo6Rx8hs-cdsOeRfSep5QEDL53miXiA4qVeIdneG69gpmVyO5B6pn5O4VRuMQljPw
```

Where doing a base64 decoding of `1QChXYTVkjypOUcHZjvk_wbLo6Rx8hs-cdsOeRfSep5QEDL53miXiA4qVeIdneG69gpmVyO5B6pn5O4VRuMQljPw` would yield a series of bytes that could be decoded to `sf.aptos.types.v1.Transaction` Protobuf structure.
