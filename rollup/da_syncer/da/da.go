package da

import (
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
	Blocks() ([]*types.Block, error)
}

type Entries []Entry
