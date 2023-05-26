package sync_service

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// BridgeClient is a wrapper around EthClient that adds
// methods for conveniently collecting L1 messages.
type BridgeClient struct {
	client                EthClient
	confirmations         rpc.BlockNumber
	l1MessageQueueAddress common.Address
}

func newBridgeClient(ctx context.Context, l1Client EthClient, l1ChainId uint64, confirmations rpc.BlockNumber, l1MessageQueueAddress *common.Address) (*BridgeClient, error) {
	if l1MessageQueueAddress == nil {
		return nil, errors.New("must pass l1MessageQueueAddress to BridgeClient")
	}

	// sanity check: compare chain IDs
	got, err := l1Client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query L1 chain ID, err = %w", err)
	}
	if got.Cmp(big.NewInt(0).SetUint64(l1ChainId)) != 0 {
		return nil, fmt.Errorf("unexpected chain ID, expected = %v, got = %v", l1ChainId, got)
	}

	client := BridgeClient{
		client:                l1Client,
		confirmations:         confirmations,
		l1MessageQueueAddress: *l1MessageQueueAddress,
	}

	return &client, nil
}

// fetchMessagesInRange retrieves and parses all L1 messages between the
// provided from and to L1 block numbers (inclusive).
func (c *BridgeClient) fetchMessagesInRange(ctx context.Context, from, to uint64) ([]types.L1MessageTx, error) {
	log.Trace("Sync service fetchMessagesInRange", "fromBlock", from, "toBlock", to)

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(0).SetUint64(from),
		ToBlock:   big.NewInt(0).SetUint64(to),
		Addresses: []common.Address{
			c.l1MessageQueueAddress,
		},
		Topics: [][]common.Hash{
			{L1QueueTransactionEventSignature},
		},
	}

	logs, err := c.client.FilterLogs(ctx, query)
	if err != nil {
		log.Trace("eth_getLogs failed", "query", query, "err", err)
		return nil, fmt.Errorf("eth_getLogs failed: %w", err)
	}
	if len(logs) == 0 {
		return nil, nil
	}

	msgs, err := c.parseLogs(logs)
	if err != nil {
		log.Trace("failed to parse emitted event logs", "logs", logs, "err", err)
		return nil, fmt.Errorf("failed to parse emitted event logs: %w", err)
	}
	log.Trace("Received new L1 events", "fromBlock", from, "toBlock", to, "msgs", msgs)
	return msgs, nil
}

func (c *BridgeClient) parseLogs(logs []types.Log) ([]types.L1MessageTx, error) {
	var msgs []types.L1MessageTx

	for _, vLog := range logs {
		event := L1QueueTransactionEvent{}
		err := unpackLog(L1MessageQueueABI, &event, "QueueTransaction", vLog)
		if err != nil {
			return msgs, fmt.Errorf("failed to unpack L1 QueueTransaction event: %w", err)
		}

		if !event.QueueIndex.IsUint64() || !event.GasLimit.IsUint64() {
			return nil, fmt.Errorf("invalid QueueTransaction event: QueueIndex = %v, GasLimit = %v", event.QueueIndex, event.GasLimit)
		}

		msgs = append(msgs, types.L1MessageTx{
			QueueIndex: event.QueueIndex.Uint64(),
			Gas:        event.GasLimit.Uint64(),
			To:         &event.Target,
			Value:      event.Value,
			Data:       event.Data,
			Sender:     event.Sender,
		})
	}

	return msgs, nil
}

func unpackLog(c *abi.ABI, out interface{}, event string, log types.Log) error {
	if event, ok := c.Events[event]; !ok || log.Topics[0] != event.ID {
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

func (c *BridgeClient) getLatestConfirmedBlockNumber(ctx context.Context) (uint64, error) {
	// confirmation based on "safe" or "finalized" block tag
	if c.confirmations == rpc.SafeBlockNumber || c.confirmations == rpc.FinalizedBlockNumber {
		tag := big.NewInt(int64(c.confirmations))
		header, err := c.client.HeaderByNumber(ctx, tag)
		if err != nil {
			return 0, err
		}
		if !header.Number.IsInt64() {
			return 0, fmt.Errorf("received invalid block confirm: %v", header.Number)
		}
		return header.Number.Uint64(), nil
	}

	// confirmation based on latest block number
	if c.confirmations == rpc.LatestBlockNumber {
		number, err := c.client.BlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		return number, nil
	}

	// confirmation based on a certain number of blocks
	if c.confirmations.Int64() >= 0 {
		number, err := c.client.BlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		confirmations := uint64(c.confirmations.Int64())
		if number >= confirmations {
			return number - confirmations, nil
		}
		return 0, nil
	}

	return 0, fmt.Errorf("unknown confirmation type: %v", c.confirmations)
}
