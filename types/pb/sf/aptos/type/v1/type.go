package pbaptos

import (
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/streamingfast/bstream"
)

func (b *Transaction) ID() string {
	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, b.Version)

	return hex.EncodeToString(id)
}

func (b *Transaction) Number() uint64 {
	return b.Version
}

func (b *Transaction) PreviousID() string {
	if b.Version <= bstream.GetProtocolFirstStreamableBlock {
		return ""
	}

	previousID := make([]byte, 8)
	binary.BigEndian.PutUint64(previousID, b.Version-1)

	return hex.EncodeToString(previousID)
}

func (b *Transaction) LIBNum() uint64 {
	number := b.Number()
	if number <= bstream.GetProtocolFirstStreamableBlock {
		return number
	}

	// Since there is no forks blocks on Aptos, I'm pretty sure that last irreversible block number
	// is the block's number itself. However I'm not sure overall how the Firehose stack would react
	// to LIBNum == Num so to play safe for now, previous block of current is irreversible.
	return b.Number() - 1
}

func (b *Transaction) Time() time.Time {
	return b.Timestamp.AsTime()
}
