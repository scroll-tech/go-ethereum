package sync_service

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// BridgeClient is a wrapper around EthClient that adds
// methods for conveniently collecting L1 messages and encryption keys.
type BridgeClient struct {
	client        EthClient
	confirmations rpc.BlockNumber

	l1MessageQueueV1Address common.Address
	filtererV1              *L1MessageQueueFilterer

	enableMessageQueueV2    bool
	l1MessageQueueV2Address common.Address
	filtererV2              *L1MessageQueueFilterer

	scrollChainAddress common.Address
}

func newBridgeClient(ctx context.Context, l1Client EthClient, l1ChainId uint64, confirmations rpc.BlockNumber, l1MessageQueueV1Address common.Address, enableMessageQueueV2 bool, l1MessageQueueV2Address common.Address, scrollChainAddress common.Address) (*BridgeClient, error) {
	if l1MessageQueueV1Address == (common.Address{}) {
		return nil, errors.New("must pass non-zero l1MessageQueueV1Address to BridgeClient")
	}
	if enableMessageQueueV2 && l1MessageQueueV2Address == (common.Address{}) {
		return nil, errors.New("must pass non-zero l1MessageQueueV2Address to BridgeClient")
	}

	// sanity check: compare chain IDs
	got, err := l1Client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query L1 chain ID, err = %w", err)
	}
	if got.Cmp(big.NewInt(0).SetUint64(l1ChainId)) != 0 {
		return nil, fmt.Errorf("unexpected chain ID, expected = %v, got = %v", l1ChainId, got)
	}

	filtererV1, err := NewL1MessageQueueFilterer(l1MessageQueueV1Address, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize L1MessageQueueV1Filterer, err = %w", err)
	}

	var filtererV2 *L1MessageQueueFilterer
	if enableMessageQueueV2 {
		filtererV2, err = NewL1MessageQueueFilterer(l1MessageQueueV2Address, l1Client)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize L1MessageQueueV2Filterer, err = %w", err)
		}
	}

	client := BridgeClient{
		client:        l1Client,
		confirmations: confirmations,

		l1MessageQueueV1Address: l1MessageQueueV1Address,
		filtererV1:              filtererV1,

		enableMessageQueueV2:    enableMessageQueueV2,
		l1MessageQueueV2Address: l1MessageQueueV2Address,
		filtererV2:              filtererV2,

		scrollChainAddress: scrollChainAddress,
	}

	return &client, nil
}

// fetchMessagesInRange retrieves and parses all L1 messages between the
// provided from and to L1 block numbers (inclusive).
func (c *BridgeClient) fetchMessagesInRange(ctx context.Context, from, to uint64, queryL1MessagesV1 bool) ([]types.L1MessageTx, []types.L1MessageTx, error) {
	log.Trace("BridgeClient fetchMessagesInRange", "fromBlock", from, "toBlock", to)

	var msgsV1, msgsV2 []types.L1MessageTx

	opts := bind.FilterOpts{
		Start:   from,
		End:     &to,
		Context: ctx,
	}

	// Query L1MessageQueueV1 if enabled. We disable querying of L1MessageQueueV1 once L1MessageQueueV2 is enabled,
	// and we have received the first QueueTransaction event from L1MessageQueueV2.
	if queryL1MessagesV1 {
		it, err := c.filtererV1.FilterQueueTransaction(&opts, nil, nil)
		if err != nil {
			return nil, nil, err
		}

		for it.Next() {
			event := it.Event
			log.Trace("Received new L1 QueueTransaction event from L1MessageQueueV1", "event", event)

			if !event.GasLimit.IsUint64() {
				return nil, nil, fmt.Errorf("invalid QueueTransaction event: QueueIndex = %v, GasLimit = %v", event.QueueIndex, event.GasLimit)
			}

			msgsV1 = append(msgsV1, types.L1MessageTx{
				QueueIndex: event.QueueIndex,
				Gas:        event.GasLimit.Uint64(),
				To:         &event.Target,
				Value:      event.Value,
				Data:       event.Data,
				Sender:     event.Sender,
			})
		}

		if err = it.Error(); err != nil {
			return nil, nil, err
		}
	}

	// We allow to explicitly enable/disable querying of L1MessageQueueV2. This is useful for running the node without
	// MessageQueueV2 available on L1 or for testing purposes.
	if !c.enableMessageQueueV2 {
		return msgsV1, nil, nil
	}

	// We always query L1MessageQueueV2. Before EuclidV2 L1 upgrade tx this will return an empty list as we don't use
	// L1MessageQueueV2 to enqueue L1 messages before EuclidV2.
	it, err := c.filtererV2.FilterQueueTransaction(&opts, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	for it.Next() {
		event := it.Event
		log.Trace("Received new L1 QueueTransaction event from L1MessageQueueV2", "event", event)

		if !event.GasLimit.IsUint64() {
			return nil, nil, fmt.Errorf("invalid QueueTransaction event: QueueIndex = %v, GasLimit = %v", event.QueueIndex, event.GasLimit)
		}

		msgsV2 = append(msgsV2, types.L1MessageTx{
			QueueIndex: event.QueueIndex,
			Gas:        event.GasLimit.Uint64(),
			To:         &event.Target,
			Value:      event.Value,
			Data:       event.Data,
			Sender:     event.Sender,
		})
	}

	if err = it.Error(); err != nil {
		return nil, nil, err
	}

	return msgsV1, msgsV2, nil
}

// fetchEncryptionKeysInRange retrieves and parses all NewEncryptionKey events between the
// provided from and to L1 block numbers (inclusive).
func (c *BridgeClient) fetchEncryptionKeysInRange(ctx context.Context, from, to uint64) ([]types.EncryptionKey, error) {
	log.Trace("BridgeClient fetchEncryptionKeysInRange", "fromBlock", from, "toBlock", to)

	var keys []types.EncryptionKey

	opts := bind.FilterOpts{
		Start:   from,
		End:     &to,
		Context: ctx,
	}

	// Get the NewEncryptionKey event ID from the ABI
	eventID := l1.ScrollChainValidiumABI.Events["NewEncryptionKey"].ID

	// Query NewEncryptionKey events from ScrollChainValidium
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(from)),
		ToBlock:   big.NewInt(int64(to)),
		Addresses: []common.Address{c.scrollChainAddress},
		Topics:    [][]common.Hash{{eventID}},
	}

	logs, err := c.client.FilterLogs(opts.Context, query)
	if err != nil {
		return nil, err
	}

	for _, vLog := range logs {
		// Parse event
		var event struct {
			KeyId    *big.Int
			MsgIndex *big.Int
			Key      []byte
		}

		err := l1.ScrollChainValidiumABI.UnpackIntoInterface(&event, "NewEncryptionKey", vLog.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack NewEncryptionKey event: %w", err)
		}

		// Extract keyId from indexed parameter (first topic after event signature)
		if len(vLog.Topics) < 2 {
			return nil, fmt.Errorf("NewEncryptionKey event missing keyId topic")
		}
		event.KeyId = new(big.Int).SetBytes(vLog.Topics[1].Bytes())

		if !event.KeyId.IsUint64() || !event.MsgIndex.IsUint64() {
			return nil, fmt.Errorf("invalid NewEncryptionKey event: KeyId = %v, MsgIndex = %v", event.KeyId, event.MsgIndex)
		}

		// Validate key length: ECIES public keys are 33 bytes (compressed) or 65 bytes (uncompressed)
		if len(event.Key) != 33 && len(event.Key) != 65 {
			return nil, fmt.Errorf("invalid encryption key length %d for keyId %d (expected 33 or 65 bytes)", len(event.Key), event.KeyId.Uint64())
		}

		encKey := types.EncryptionKey{
			KeyId:    event.KeyId.Uint64(),
			MsgIndex: event.MsgIndex.Uint64(),
			Key:      event.Key,
		}

		log.Info("Fetched new encryption key from ScrollChainValidium",
			"keyId", encKey.KeyId,
			"msgIndex", encKey.MsgIndex,
			"keyLength", len(encKey.Key),
			"blockNumber", vLog.BlockNumber,
			"txHash", vLog.TxHash.Hex())

		keys = append(keys, encKey)
	}

	return keys, nil
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
			return 0, fmt.Errorf("received unexpected block number in BridgeClient: %v", header.Number)
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
