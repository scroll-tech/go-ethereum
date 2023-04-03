package sync_service

import (
	"context"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// UnpackLog unpacks a retrieved log into the provided output structure.
// @todo: add unit test.
func UnpackLog(c *abi.ABI, out interface{}, event string, log types.Log) error {
	if log.Topics[0] != c.Events[event].ID {
		return fmt.Errorf("event signature mismatch")
	}
	if len(log.Data) > 0 {
		if err := c.UnpackIntoInterface(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range c.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopics(out, indexed, log.Topics[1:])
}

// GetLatestConfirmedBlockNumber get confirmed block number by rpc.BlockNumber type.
func GetLatestConfirmedBlockNumber(ctx context.Context, client *ethclient.Client, confirm rpc.BlockNumber) (uint64, error) {
	if confirm == rpc.SafeBlockNumber || confirm == rpc.FinalizedBlockNumber {
		var tag *big.Int
		if confirm == rpc.FinalizedBlockNumber {
			tag = big.NewInt(int64(rpc.FinalizedBlockNumber))
		} else {
			tag = big.NewInt(int64(rpc.SafeBlockNumber))
		}

		header, err := client.HeaderByNumber(ctx, tag)
		if err != nil {
			return 0, err
		}
		if !header.Number.IsInt64() {
			return 0, fmt.Errorf("received invalid block confirm: %v", header.Number)
		}
		return header.Number.Uint64(), nil
	} else if confirm == rpc.LatestBlockNumber {
		number, err := client.BlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		return number, nil
	} else if confirm.Int64() >= 0 { // If it's positive integer, consider it as a certain confirm value.
		number, err := client.BlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		cfmNum := uint64(confirm.Int64())

		if number >= cfmNum {
			return number - cfmNum, nil
		}
		return 0, nil
	} else {
		return 0, fmt.Errorf("unknown confirmation type: %v", confirm)
	}
}
