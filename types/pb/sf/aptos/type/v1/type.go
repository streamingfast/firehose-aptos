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

func (b *Transaction) Time() time.Time {
	return time.Unix(0, int64(b.Timestamp)).UTC()
}
