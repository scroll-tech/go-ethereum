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
)

// StateDB represents the StateDB interface
// required to compute the L1 fee
type StateDB interface {
	GetState(common.Address, common.Hash) common.Hash
}

type L1BlocksWorker struct {
	ctx      context.Context
	l1Client sync_service.EthClient
	l1BlocksABI *abi.ABI
}

func NewL1BlocksWorker(ctx context.Context, l1Client sync_service.EthClient) (*L1BlocksWorker, error) {
	l1BlocksAbi, err := abis.L1BlocksMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load L1Blocks ABI, err: %w", err)
	}
	worker := L1BlocksWorker{
		ctx,
		l1Client,
		l1BlocksAbi,
	}
	return &worker, nil
}

func (w *L1BlocksWorker) GetLatestL1BlockNumber(state StateDB) (*big.Int) {
	return state.GetState(rcfg.L1BlocksAddress, rcfg.LatestBlockNumberSlot).Big()
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

func (w *L1BlocksWorker) GenerateL1BlockTx(l1BlockNumber *big.Int) (*types.SystemTx, error) {
	headerRlp, err := w.fetchL1BlockHeaderRlp(l1BlockNumber)
	if err != nil {
		return nil, err
	}
	data, err := w.l1BlocksABI.Pack("setL1BlockHeader", headerRlp)
	if err != nil {
		return nil, fmt.Errorf("failed to pack the calldata for setL1BlockHeader, err: %w", err)
	}

	return &types.SystemTx{
		Sender: rcfg.SystemSenderAddress,
		To:     &rcfg.L1BlocksAddress,
		Data:   data,
	}, nil
}
