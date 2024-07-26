package da

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/core/types"
)

type Type int

const (
	// CommitBatchV0Type contains data of event of CommitBatchV0Type
	CommitBatchV0Type Type = iota
	// CommitBatchV1Type contains data of event of CommitBatchV1Type
	CommitBatchV1Type
	// CommitBatchV2Type contains data of event of CommitBatchV2Type
	CommitBatchV2Type
	// RevertBatchType contains data of event of RevertBatchType
	RevertBatchType
	// FinalizeBatchType contains data of event of FinalizeBatchType
	FinalizeBatchType
	// FinalizeBatchV3Type contains data of event of FinalizeBatchType v3
	FinalizeBatchV3Type
)

type Entry interface {
	Type() Type
	BatchIndex() uint64
	L1BlockNumber() uint64
}

type EntryWithBlocks interface {
	Entry
	Blocks() ([]*PartialBlock, error)
}

type Entries []Entry

type PartialHeader struct {
	Number     uint64
	Time       uint64
	BaseFee    *big.Int
	GasLimit   uint64
	Difficulty *big.Int
	ExtraData  []byte
}

func (h *PartialHeader) ToHeader() *types.Header {
	return &types.Header{
		Number:     big.NewInt(0).SetUint64(h.Number),
		Time:       h.Time,
		BaseFee:    h.BaseFee,
		GasLimit:   h.GasLimit,
		Difficulty: h.Difficulty,
		Extra:      h.ExtraData,
	}
}

type PartialBlock struct {
	PartialHeader *PartialHeader
	Transactions  types.Transactions
}

func NewPartialBlock(partialHeader *PartialHeader, txs types.Transactions) *PartialBlock {
	return &PartialBlock{
		PartialHeader: partialHeader,
		Transactions:  txs,
	}
}
