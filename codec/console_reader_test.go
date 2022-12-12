// Copyright 2021 dfuse Platform Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codec

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/firehose-aptos/types"
	pbaptos "github.com/streamingfast/firehose-aptos/types/pb/aptos/extractor/v1"
	tt "github.com/streamingfast/firehose-aptos/types/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFromFile(t *testing.T) {
	tests := []struct {
		name        string
		lines       []string
		assertError require.ErrorAssertionFunc
	}{
		{
			"malformed init",
			[]string{
				fireInitCustom("wrong"),
			},
			EqualErrorAssertion(`invalid log line length: 6 or 7 fields required but found 1 (on line "FIRE INIT wrong")`),
		},

		{
			"genesis",
			[]string{
				fireInit(),

				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockEnd(1),
			},
			require.NoError,
		},

		{
			"genesis with chain_id field name in init",
			[]string{
				fireInitCustom("aptos-node 0.0.0 aptos 0 0 chain_id 4"),

				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockEnd(1),
			},
			require.NoError,
		},

		{
			"pre-init ignored",
			[]string{
				fireBlockStart(2),
				fireTrx(tt.Transaction(t, 2, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockEnd(2),

				fireInit(),

				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockEnd(1),
			},
			require.NoError,
		},

		{
			"new block start resets active one",
			[]string{
				fireInit(),

				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockEnd(1),
			},
			require.NoError,
		},

		{
			"multiple transaction in one block",
			[]string{
				fireInit(),

				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeUser, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeStateCheckpoint, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockEnd(1),
			},
			require.NoError,
		},

		{
			"multiple transaction in multiple block",
			[]string{
				fireInit(),

				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireTrx(tt.Transaction(t, 2, tt.TrxTypeUser, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireTrx(tt.Transaction(t, 3, tt.TrxTypeStateCheckpoint, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockEnd(1),

				fireBlockStart(2),
				fireTrx(tt.Transaction(t, 4, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireTrx(tt.Transaction(t, 5, tt.TrxTypeUser, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireTrx(tt.Transaction(t, 6, tt.TrxTypeStateCheckpoint, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockEnd(2),
			},
			require.NoError,
		},

		{
			"init received multiple time",
			[]string{
				fireInit(),

				fireInit(),
			},
			EqualErrorAssertion(`received INIT line while one has already been read (on line "FIRE INIT aptos-node 0.0.0 aptos 0 0 4")`),
		},

		{
			"block has no transasction",
			[]string{
				fireInit(),

				fireBlockStart(1),
				fireBlockEnd(1),
			},
			EqualErrorAssertion(`active block height 1 does not contain any transaction (on line "FIRE BLOCK_END 1")`),
		},

		{
			"no active block on block end",
			[]string{
				fireInit(),

				fireBlockEnd(1),
			},
			EqualErrorAssertion(`no active block in progress when reading BLOCK_END (on line "FIRE BLOCK_END 1")`),
		},

		{
			"first trx is not block start boundary",
			[]string{
				fireInit(),

				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeUser)),
				fireBlockEnd(1),
			},
			EqualErrorAssertion(`received first TRX of type "USER" that is not a valid block start boundary transaction (only Block Metadata and Genesis transaction are) (on line "FIRE TRX EAEwAw==")`),
		},

		{
			"first trx is not block start boundary after reset",
			[]string{
				fireInit(),

				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 1, tt.TrxTypeUser)),
				fireBlockEnd(1),
			},
			EqualErrorAssertion(`received first TRX of type "USER" that is not a valid block start boundary transaction (only Block Metadata and Genesis transaction are) (on line "FIRE TRX EAEwAw==")`),
		},

		{
			"multi trx is block start boundary",
			[]string{
				fireInit(),

				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 2, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireTrx(tt.Transaction(t, 3, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockEnd(1),
			},
			EqualErrorAssertion(`received non-first block start boundary TRX of type "GENESIS", expecting to only ever receive a single block satrt boundary transaction within an active block (on line "FIRE TRX CgYI5Yy48AUQAw==")`),
		},

		{
			"block end height does not match block start height",
			[]string{
				fireInit(),

				fireBlockStart(1),
				fireTrx(tt.Transaction(t, 2, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
				fireBlockEnd(2),
			},
			EqualErrorAssertion(`active block's height 1 does not match BLOCK_END received height 2 (on line "FIRE BLOCK_END 2")`),
		},

		{
			"received trx while no block is active",
			[]string{
				fireInit(),

				fireTrx(tt.Transaction(t, 2, tt.TrxTypeGenesis, tt.Timestamp(t, "2020-01-02T15:04:05Z"))),
			},
			EqualErrorAssertion(`no active block in progress when reading TRX (on line "FIRE TRX CgYI5Yy48AUQAg==")`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr := testStringConsoleReader(t, strings.Join(test.lines, "\n"))
			buf := &bytes.Buffer{}
			buf.Write([]byte("["))

			var readBlockErr error

			for first := true; true; first = false {
				out, err := cr.ReadBlock()
				if err != nil {
					if err == io.EOF {
						break
					}

					readBlockErr = err
					break
				}

				block, err := types.BlockDecoder(out)
				require.NoError(t, err)

				if !first {
					buf.Write([]byte(","))
				}

				// FIXMME: jsonpb needs to be updated to latest version of used gRPC
				//         elements. We are disaligned and using that breaks now.
				//         Needs to check what is the latest way to properly serialize
				//         Proto generated struct to JSON.
				// value, err := jsonpb.MarshalIndentToString(v, "  ")
				// require.NoError(t, err)

				value, err := json.MarshalIndent(block, "", "  ")
				require.NoError(t, err)

				buf.Write(value)
			}

			test.assertError(t, readBlockErr)
			if readBlockErr != nil {
				// We do not write the golden file if there was an error producing blocks
				return
			}

			if len(buf.Bytes()) != 0 {
				buf.Write([]byte("\n"))
			}

			buf.Write([]byte("]"))

			goldenFile := fmt.Sprintf("testdata/%s.golden.json", strings.ReplaceAll(test.name, " ", "_"))
			if os.Getenv("GOLDEN_UPDATE") == "true" {
				ioutil.WriteFile(goldenFile, buf.Bytes(), os.ModePerm)
			}

			cnt, err := ioutil.ReadFile(goldenFile)
			require.NoError(t, err)

			if !assert.Equal(t, string(cnt), buf.String()) {
				t.Error("previous diff:\n" + unifiedDiff(t, cnt, buf.Bytes()))
			}
		})
	}
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}

func testFileConsoleReader(t *testing.T, filename string) *ConsoleReader {
	t.Helper()

	fl, err := os.Open(filename)
	require.NoError(t, err)

	return testReaderConsoleReader(t, fl)
}

func testStringConsoleReader(t *testing.T, content string) *ConsoleReader {
	t.Helper()

	return testReaderConsoleReader(t, (*bufferCloser)(bytes.NewBufferString(content)))
}

func testReaderConsoleReader(t *testing.T, reader io.ReadCloser) *ConsoleReader {
	t.Helper()

	cr, err := NewConsoleReader(zlog, make(chan string, 10000))
	require.NoError(t, err)

	cr.close = func() { reader.Close() }
	cr.stats.StopPeriodicLogToZap()

	go cr.ProcessData(reader)

	return cr
}

func unifiedDiff(t *testing.T, cnt1, cnt2 []byte) string {
	file1 := "/tmp/gotests-linediff-1"
	file2 := "/tmp/gotests-linediff-2"
	err := ioutil.WriteFile(file1, cnt1, 0600)
	require.NoError(t, err)

	err = ioutil.WriteFile(file2, cnt2, 0600)
	require.NoError(t, err)

	cmd := exec.Command("diff", "-u", file1, file2)
	out, _ := cmd.Output()

	return string(out)
}

func generateSyntheticFirelogFile(filename string, lines ...string) error {
	content := strings.Join(lines, "\n")

	return os.WriteFile(filename, []byte(content), os.ModePerm)
}

func fireInit() string {
	return fireInitCustom("aptos-node 0.0.0 aptos 0 0 4")
}

func fireInitCustom(data string) string {
	return fmt.Sprintf("FIRE INIT %s", data)
}

func fireBlockStart(height uint64) string {
	return fmt.Sprintf("FIRE BLOCK_START %d", height)
}

func fireBlockEnd(height uint64) string {
	return fmt.Sprintf("FIRE BLOCK_END %d", height)
}

func fireTrx(trx *pbaptos.Transaction) string {
	encoded, err := proto.Marshal(trx)
	if err != nil {
		panic(fmt.Errorf("encode trx proto: %w", err))
	}

	return fmt.Sprintf("FIRE TRX %s", base64.StdEncoding.EncodeToString(encoded))
}

type bufferCloser bytes.Buffer

func (c *bufferCloser) Read(p []byte) (n int, err error) {
	return (*bytes.Buffer)(c).Read(p)
}

func (*bufferCloser) Close() error {
	return nil
}

func EqualErrorAssertion(errString string) require.ErrorAssertionFunc {
	return func(tt require.TestingT, err error, i ...interface{}) {
		require.EqualError(tt, err, errString, i...)
	}
}
