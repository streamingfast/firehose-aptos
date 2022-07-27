package types

import (
	"fmt"

	"github.com/streamingfast/bstream"
	pbaptos "github.com/streamingfast/firehose-aptos/types/pb/sf/aptos/type/v1"
	pbbstream "github.com/streamingfast/pbgo/sf/bstream/v1"
	"google.golang.org/protobuf/proto"
)

func BlockFromProto(b *pbaptos.Transaction) (*bstream.Block, error) {
	content, err := proto.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal to binary form: %s", err)
	}

	block := &bstream.Block{
		Id:         b.ID(),
		Number:     b.Number(),
		PreviousId: b.PreviousID(),
		Timestamp:  b.Time(),
		// Since there is no forks blocks, I'm pretty sure LIB num is itself, however
		// I'm not sure overall how the Firehose stack would react to LIBNum == Num so
		// to play safe for now, previous block is irreversible.
		LibNum:         b.Number() - 1,
		PayloadKind:    pbbstream.Protocol_UNKNOWN,
		PayloadVersion: 1,
	}

	return bstream.GetBlockPayloadSetter(block, content)
}
