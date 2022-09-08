package tt

import (
	"github.com/streamingfast/firehose-aptos/types/pb/aptos/extractor/v1"
	pbtimestamp "github.com/streamingfast/firehose-aptos/types/pb/aptos/util/timestamp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Transaction(t *testing.T, version uint64, components ...interface{}) *pbaptos.Transaction {
	trx := &pbaptos.Transaction{
		Version: version,
	}

	for _, component := range components {
		switch v := component.(type) {
		case pbaptos.Transaction_TransactionType:
			trx.Type = v

		case timestamp:
			trx.Timestamp = pbtimestamp.New(time.Time(v))

		default:
			failInvalidComponent(t, "transaction", component)
		}
	}

	return trx
}

var TrxTypeGenesis = pbaptos.Transaction_GENESIS
var TrxTypeUser = pbaptos.Transaction_USER
var TrxTypeBlockMetadata = pbaptos.Transaction_BLOCK_METADATA
var TrxTypeStateCheckpoint = pbaptos.Transaction_STATE_CHECKPOINT

type timestamp time.Time

// Timestamp can be constructed from an RFC 3339 string or from a `time.Time` value directly.
func Timestamp(t testing.T, in interface{}) timestamp {
	switch v := in.(type) {
	case string:
		out, err := time.Parse(time.RFC3339, v)
		require.NoError(t, err, "invalid timestamp")

		return timestamp(out)

	case time.Time:
		return timestamp(v)

	default:
		failInvalidComponent(t, "timestamp", in)
		return timestamp{}
	}
}

type ignoreComponent func(v interface{}) bool

func failInvalidComponent(t testing.T, tag string, component interface{}, options ...interface{}) {
	shouldIgnore := ignoreComponent(func(v interface{}) bool { return false })
	for _, option := range options {
		switch v := option.(type) {
		case ignoreComponent:
			shouldIgnore = v
		}
	}

	if shouldIgnore(component) {
		return
	}

	require.FailNowf(t, "invalid component", "Invalid %s component of type %T", tag, component)
}
