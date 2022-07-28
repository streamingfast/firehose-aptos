package tools

import (
	"encoding/base64"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	pbaptos "github.com/streamingfast/firehose-aptos/types/pb/sf/aptos/type/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

var decodeCmd = &cobra.Command{Use: "decode", Short: "Various utilities around decoding like decoding output types"}

var decodeTrxCmd = &cobra.Command{
	Use:   "trx <input> [<input>...]",
	Short: "Receives a base64 standard with padding string expecting to contain an sf.aptos.type.v1.Transaction object and decodes it",
	Args:  cobra.MinimumNArgs(1),
	RunE:  decodeTrxE,
	Example: ExamplePrefixed("sfeth tools decode trx", `
		"CgoIqe2LlwYQj8cjEBMaoQEKILhgUcdnChYZzDe+VbmXy5kAJoiN7SEZrKDvIZs6OK1HEiA2G1IsMLUxQZBTzPRwScN5LxpQuOP7atP4VcK5KJkSwBogQUNDVU1VTEFUT1JfUExBQ0VIT0xERVJfSEFTSAAAAAAoATIVRXhlY3V0ZWQgc3VjY2Vzc2Z1bGx5OiBp+0BzUC0AF95n4YzdKFku3axn0P2COQP3UR1dTzJeeyACKAowAkoA"
	`),
}

func init() {
	Cmd.AddCommand(decodeCmd)
	decodeCmd.AddCommand(decodeTrxCmd)
}

func decodeTrxE(cmd *cobra.Command, args []string) error {
	for _, input := range args {
		data, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return fmt.Errorf("invalid base64 standard with padding transaction's input: %w", err)
		}

		transaction := &pbaptos.Transaction{}
		if err := proto.Unmarshal(data, transaction); err != nil {
			return fmt.Errorf("invalid transaction's bytes: %w", err)
		}

		fmt.Println(protojson.Format(transaction))
	}

	return nil
}
