package da_syncer

import (
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rollup/types/encoding/codecv0"
	"github.com/scroll-tech/go-ethereum/rollup/types/encoding/codecv1"
)

type DAType int

const (
	// CommitBatchV0 contains data of event of CommitBatchV0
	CommitBatchV0 DAType = iota
	// CommitBatchV1 contains data of event of CommitBatchV1
	CommitBatchV1
	// RevertBatch contains data of event of RevertBatch
	RevertBatch
	// FinalizeBatch contains data of event of FinalizeBatch
	FinalizeBatch
)

type DAEntry interface {
	DAType() DAType
	GetL1BlockNumber() uint64
}

type DA []DAEntry

type CommitBatchDaV0 struct {
	DaType                 DAType
	Version                uint8
	BatchIndex             uint64
	ParentBatchHeader      *codecv0.DABatch
	SkippedL1MessageBitmap []byte
	Chunks                 []*codecv0.DAChunkRawTx
	L1Txs                  []*types.L1MessageTx

	L1BlockNumber uint64
}

func NewCommitBatchDaV0(version uint8, batchIndex uint64, parentBatchHeader *codecv0.DABatch, skippedL1MessageBitmap []byte, chunks []*codecv0.DAChunkRawTx, l1Txs []*types.L1MessageTx, l1BlockNumber uint64) DAEntry {
	return &CommitBatchDaV0{
		DaType:                 CommitBatchV0,
		Version:                version,
		BatchIndex:             batchIndex,
		ParentBatchHeader:      parentBatchHeader,
		SkippedL1MessageBitmap: skippedL1MessageBitmap,
		Chunks:                 chunks,
		L1Txs:                  l1Txs,
		L1BlockNumber:          l1BlockNumber,
	}
}

func (f *CommitBatchDaV0) DAType() DAType {
	return f.DaType
}

func (f *CommitBatchDaV0) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}

type CommitBatchDaV1 struct {
	DaType                 DAType
	Version                uint8
	BatchIndex             uint64
	ParentBatchHeader      *codecv1.DABatch
	SkippedL1MessageBitmap []byte
	Chunks                 []*codecv1.DAChunkRawTx
	L1Txs                  []*types.L1MessageTx

	L1BlockNumber uint64
}

func NewCommitBatchDaV1(version uint8, batchIndex uint64, parentBatchHeader *codecv1.DABatch, skippedL1MessageBitmap []byte, chunks []*codecv1.DAChunkRawTx, l1Txs []*types.L1MessageTx, l1BlockNumber uint64) DAEntry {
	return &CommitBatchDaV1{
		DaType:                 CommitBatchV1,
		Version:                version,
		BatchIndex:             batchIndex,
		ParentBatchHeader:      parentBatchHeader,
		SkippedL1MessageBitmap: skippedL1MessageBitmap,
		Chunks:                 chunks,
		L1Txs:                  l1Txs,
		L1BlockNumber:          l1BlockNumber,
	}
}

func (f *CommitBatchDaV1) DAType() DAType {
	return f.DaType
}

func (f *CommitBatchDaV1) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}

type RevertBatchDA struct {
	DaType     DAType
	BatchIndex uint64

	L1BlockNumber uint64
}

func NewRevertBatchDA(batchIndex uint64) DAEntry {
	return &RevertBatchDA{
		DaType:     RevertBatch,
		BatchIndex: batchIndex,
	}
}

func (f *RevertBatchDA) DAType() DAType {
	return f.DaType
}

func (f *RevertBatchDA) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}

type FinalizeBatchDA struct {
	DaType     DAType
	BatchIndex uint64

	L1BlockNumber uint64
}

func NewFinalizeBatchDA(batchIndex uint64) DAEntry {
	return &FinalizeBatchDA{
		DaType:     FinalizeBatch,
		BatchIndex: batchIndex,
	}
}

func (f *FinalizeBatchDA) DAType() DAType {
	return f.DaType
}

func (f *FinalizeBatchDA) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}
