package sync_service

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

type BridgeClient struct {
	client                *ethclient.Client
	confirmations         rpc.BlockNumber
	l1MessageQueueAddress common.Address
}

func newBridgeClient(ctx context.Context, l1Endpoint string, l1ChainId uint64, confirmations rpc.BlockNumber, l1MessageQueueAddress *common.Address) (*BridgeClient, error) {
	if l1MessageQueueAddress == nil {
		return nil, errors.New("must pass l1MessageQueueAddress to BridgeClient")
	}

	ethClient, err := ethclient.Dial(l1Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to dial L1 endpoint: %w", err)
	}

	// sanity check: compare chain IDs
	got, err := ethClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	if got.Uint64() != l1ChainId { // TODO
		return nil, fmt.Errorf("unexpected chain ID, expected = %v, got = %v", l1ChainId, got)
	}

	client := BridgeClient{
		client:                ethClient,
		confirmations:         confirmations,
		l1MessageQueueAddress: *l1MessageQueueAddress,
	}

	return &client, nil
}

func (c *BridgeClient) fetchMessagesInRange(ctx context.Context, from, to uint64) ([]types.L1MessageTx, error) {
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
		log.Warn("eth_getLogs failed", "err", err)
		return nil, err
	}

	if len(logs) == 0 {
		return nil, nil
	}

	log.Info("Received new L1 events", "fromBlock", from, "toBlock", to, "count", len(logs))

	msgs, err := c.parseLogs(logs)
	if err != nil {
		log.Error("Failed to parse emitted events logs", "err", err)
		return nil, err
	}

	return msgs, nil
}

func (c *BridgeClient) parseLogs(logs []types.Log) ([]types.L1MessageTx, error) {
	var msgs []types.L1MessageTx

	for _, vLog := range logs {
		event := L1QueueTransactionEvent{}
		err := UnpackLog(L1MessageQueueABI, &event, "QueueTransaction", vLog)
		if err != nil {
			log.Warn("Failed to unpack L1 QueueTransaction event", "err", err)
			return msgs, err
		}

		msgs = append(msgs, types.L1MessageTx{
			Nonce:  event.QueueIndex.Uint64(),
			Gas:    event.GasLimit.Uint64(),
			To:     &event.Target,
			Value:  event.Value,
			Data:   event.Data,
			Sender: &event.Sender,
		})
	}

	return msgs, nil
}

// getLatestConfirmedBlockNumber get confirmed block number by rpc.BlockNumber type.
func (c *BridgeClient) getLatestConfirmedBlockNumber(ctx context.Context) (uint64, error) {
	if c.confirmations == rpc.SafeBlockNumber || c.confirmations == rpc.FinalizedBlockNumber {
		var tag *big.Int
		if c.confirmations == rpc.FinalizedBlockNumber {
			tag = big.NewInt(int64(rpc.FinalizedBlockNumber))
		} else {
			tag = big.NewInt(int64(rpc.SafeBlockNumber))
		}

		header, err := c.client.HeaderByNumber(ctx, tag)
		if err != nil {
			return 0, err
		}
		if !header.Number.IsInt64() {
			return 0, fmt.Errorf("received invalid block confirm: %v", header.Number)
		}
		return header.Number.Uint64(), nil
	} else if c.confirmations == rpc.LatestBlockNumber {
		number, err := c.client.BlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		return number, nil
	} else if c.confirmations.Int64() >= 0 { // If it's positive integer, consider it as a certain confirm value.
		number, err := c.client.BlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		cfmNum := uint64(c.confirmations.Int64())

		if number >= cfmNum {
			return number - cfmNum, nil
		}
		return 0, nil
	} else {
		return 0, fmt.Errorf("unknown confirmation type: %v", c.confirmations)
	}
}
