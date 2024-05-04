package system_contracts

import (
	"context"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rlp"
	"github.com/scroll-tech/go-ethereum/rollup/abis"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// StateDB represents the StateDB interface
type StateDB interface {
	GetState(common.Address, common.Hash) common.Hash
	GetNonce(common.Address) uint64
}

type L1BlocksWorker struct {
	ctx           context.Context
	l1Client      sync_service.EthClient
	l1BlocksABI   *abi.ABI
	confirmations rpc.BlockNumber
}

func NewL1BlocksWorker(ctx context.Context, l1Client sync_service.EthClient, l1ChainId uint64, confirmations rpc.BlockNumber) (*L1BlocksWorker, error) {
	// sanity check: compare chain IDs
	got, err := l1Client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query L1 chain ID, err = %w", err)
	}
	if got.Cmp(big.NewInt(0).SetUint64(l1ChainId)) != 0 {
		return nil, fmt.Errorf("unexpected chain ID, expected = %v, got = %v", l1ChainId, got)
	}

	// get the L1Blocks ABI
	l1BlocksAbi, err := abis.L1BlocksMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load L1Blocks ABI, err: %w", err)
	}

	worker := L1BlocksWorker{
		ctx,
		l1Client,
		l1BlocksAbi,
		confirmations,
	}
	return &worker, nil
}

func (w *L1BlocksWorker) GetLatestL1BlockNumberOnL2(state StateDB) (*big.Int) {
	return state.GetState(rcfg.L1BlocksAddress, rcfg.LatestBlockNumberSlot).Big()
}

func (w *L1BlocksWorker) GetLatestConfirmedL1BlockNumber() (uint64, error) {
	// confirmation based on "safe" or "finalized" block tag
	if w.confirmations == rpc.SafeBlockNumber || w.confirmations == rpc.FinalizedBlockNumber {
		tag := big.NewInt(int64(w.confirmations))
		header, err := w.l1Client.HeaderByNumber(w.ctx, tag)
		if err != nil {
			return 0, err
		}
		if !header.Number.IsInt64() {
			return 0, fmt.Errorf("received unexpected block number in BridgeClient: %v", header.Number)
		}
		return header.Number.Uint64(), nil
	}

	// confirmation based on latest block number
	if w.confirmations == rpc.LatestBlockNumber {
		number, err := w.l1Client.BlockNumber(w.ctx)
		if err != nil {
			return 0, err
		}
		return number, nil
	}

	// confirmation based on a certain number of blocks
	if w.confirmations.Int64() >= 0 {
		number, err := w.l1Client.BlockNumber(w.ctx)
		if err != nil {
			return 0, err
		}
		confirmations := uint64(w.confirmations.Int64())
		if number >= confirmations {
			return number - confirmations, nil
		}
		return 0, nil
	}

	return 0, fmt.Errorf("unknown confirmation type: %v", w.confirmations)
}

func (w *L1BlocksWorker) fetchL1BlockHeaderRlp(l1BlockNumber *big.Int) ([]byte, error) {
	header, err := w.l1Client.HeaderByNumber(w.ctx, l1BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get L1 block header, err: %w", err)
	}
	headerRlp, err := rlp.EncodeToBytes(header)
	if err != nil {
		return nil, fmt.Errorf("failed in RLP encoding of L1 block header, err: %w", err)
	}
	return headerRlp, nil
}

func (w *L1BlocksWorker) generateL1BlockMsg(l1BlockNumber uint64, nonce uint64) (*types.SystemTx, error) {
	headerRlp, err := w.fetchL1BlockHeaderRlp(big.NewInt(int64(l1BlockNumber)))
	if err != nil {
		return nil, err
	}
	data, err := w.l1BlocksABI.Pack("setL1BlockHeader", headerRlp)
	if err != nil {
		return nil, fmt.Errorf("failed to pack the calldata for setL1BlockHeader, err: %w", err)
	}

	return &types.SystemTx{
		Sender: rcfg.SystemSenderAddress,
		To:     rcfg.L1BlocksAddress,
		Nonce:  nonce,
		Data:   data,
	}, nil
}

func (w *L1BlocksWorker) GenerateL1BlockMsgs(from, to uint64, state StateDB) ([]types.SystemTx, error) {
	if to < from {
		return nil, fmt.Errorf("invalid block range", "from", from, "to", to)
	}
	msgs := make([]types.SystemTx, (to - from + 1))
	nonce := state.GetNonce(rcfg.SystemSenderAddress)
	var i uint64
	for i = 0; i < to - from + 1; i++ {
		msg, err := w.generateL1BlockMsg(from + i, nonce + i)
		if err != nil {
			return nil, err
		}
		msgs[i] = *msg
	}
	return msgs, nil
}
