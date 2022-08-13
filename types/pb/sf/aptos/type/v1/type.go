package pbaptos

import (
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/streamingfast/bstream"
)

func (b *Block) ID() string {
	return uint64ToID(b.Height)
}

func (b *Block) Number() uint64 {
	return b.Height
}

func (b *Block) PreviousID() string {
	return uint64ToID(b.PreviousNum())
}

func (b *Block) PreviousNum() uint64 {
	if b.Height <= bstream.GetProtocolFirstStreamableBlock {
		return bstream.GetProtocolFirstStreamableBlock
	}

	return b.Height - 1
}

func (b *Block) LIBNum() uint64 {
	number := b.Number()
	if number <= bstream.GetProtocolFirstStreamableBlock {
		return number
	}

	// Since there is no forks blocks on Aptos, I'm pretty sure that last irreversible block number
	// is the block's number itself. However I'm not sure overall how the Firehose stack would react
	// to LIBNum == Num so to play safe for now, previous block of current is irreversible.
	return b.Number() - 1
}

func (b *Block) Time() time.Time {
	return b.Timestamp.AsTime()
}

func (t *Transaction) ID() string {
	return uint64ToID(t.Version)
}

func (t *Transaction) Time() time.Time {
	return t.Timestamp.AsTime()
}

func (t *Transaction) IsBlockStartBoundaryType() bool {
	return t.Type == Transaction_BLOCK_METADATA || t.Type == Transaction_GENESIS
}

func uint64ToHash(height uint64) []byte {
	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, height)

	return id
}

func uint64ToID(height uint64) string {
	return hex.EncodeToString(uint64ToHash(height))
}
