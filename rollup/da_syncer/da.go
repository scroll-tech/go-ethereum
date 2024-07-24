package da_syncer

import (
	"github.com/scroll-tech/da-codec/encoding/codecv0"
	"github.com/scroll-tech/da-codec/encoding/codecv1"
	"github.com/scroll-tech/da-codec/encoding/codecv2"

	"github.com/scroll-tech/go-ethereum/core/types"
)

type DAType int

const (
	// CommitBatchV0 contains data of event of CommitBatchV0
	CommitBatchV0 DAType = iota
	// CommitBatchV1 contains data of event of CommitBatchV1
	CommitBatchV1
	// CommitBatchV2 contains data of event of CommitBatchV2
	CommitBatchV2
	// RevertBatch contains data of event of RevertBatch
	RevertBatch
	// FinalizeBatch contains data of event of FinalizeBatch
	FinalizeBatch
	// FinalizeBatchV3 contains data of event of FinalizeBatch v3
	FinalizeBatchV3
)

type DAEntry interface {
	DAType() DAType
	GetL1BlockNumber() uint64
}

type DA []DAEntry

type CommitBatchDAV0 struct {
	Version                    uint8
	BatchIndex                 uint64
	ParentTotalL1MessagePopped uint64
	SkippedL1MessageBitmap     []byte
	Chunks                     []*codecv0.DAChunkRawTx
	L1Txs                      []*types.L1MessageTx

	L1BlockNumber uint64
}

func NewCommitBatchDAV0(version uint8, batchIndex uint64, parentTotalL1MessagePopped uint64, skippedL1MessageBitmap []byte, chunks []*codecv0.DAChunkRawTx, l1Txs []*types.L1MessageTx, l1BlockNumber uint64) DAEntry {
	return &CommitBatchDAV0{
		Version:                    version,
		BatchIndex:                 batchIndex,
		ParentTotalL1MessagePopped: parentTotalL1MessagePopped,
		SkippedL1MessageBitmap:     skippedL1MessageBitmap,
		Chunks:                     chunks,
		L1Txs:                      l1Txs,
		L1BlockNumber:              l1BlockNumber,
	}
}

func (f *CommitBatchDAV0) DAType() DAType {
	return CommitBatchV0
}

func (f *CommitBatchDAV0) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}

type CommitBatchDAV1 struct {
	Version                    uint8
	BatchIndex                 uint64
	ParentTotalL1MessagePopped uint64
	SkippedL1MessageBitmap     []byte
	Chunks                     []*codecv1.DAChunkRawTx
	L1Txs                      []*types.L1MessageTx

	L1BlockNumber uint64
}

func NewCommitBatchDAV1(version uint8, batchIndex uint64, parentTotalL1MessagePopped uint64, skippedL1MessageBitmap []byte, chunks []*codecv1.DAChunkRawTx, l1Txs []*types.L1MessageTx, l1BlockNumber uint64) DAEntry {
	return &CommitBatchDAV1{
		Version:                    version,
		BatchIndex:                 batchIndex,
		ParentTotalL1MessagePopped: parentTotalL1MessagePopped,
		SkippedL1MessageBitmap:     skippedL1MessageBitmap,
		Chunks:                     chunks,
		L1Txs:                      l1Txs,
		L1BlockNumber:              l1BlockNumber,
	}
}

func (f *CommitBatchDAV1) DAType() DAType {
	return CommitBatchV1
}

func (f *CommitBatchDAV1) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}

type CommitBatchDAV2 struct {
	Version                    uint8
	BatchIndex                 uint64
	ParentTotalL1MessagePopped uint64
	SkippedL1MessageBitmap     []byte
	Chunks                     []*codecv2.DAChunkRawTx
	L1Txs                      []*types.L1MessageTx

	L1BlockNumber uint64
}

func NewCommitBatchDAV2(version uint8, batchIndex uint64, parentTotalL1MessagePopped uint64, skippedL1MessageBitmap []byte, chunks []*codecv2.DAChunkRawTx, l1Txs []*types.L1MessageTx, l1BlockNumber uint64) DAEntry {
	return &CommitBatchDAV2{
		Version:                    version,
		BatchIndex:                 batchIndex,
		ParentTotalL1MessagePopped: parentTotalL1MessagePopped,
		SkippedL1MessageBitmap:     skippedL1MessageBitmap,
		Chunks:                     chunks,
		L1Txs:                      l1Txs,
		L1BlockNumber:              l1BlockNumber,
	}
}

func (f *CommitBatchDAV2) DAType() DAType {
	return CommitBatchV2
}

func (f *CommitBatchDAV2) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}

type RevertBatchDA struct {
	BatchIndex uint64

	L1BlockNumber uint64
}

func NewRevertBatchDA(batchIndex uint64) DAEntry {
	return &RevertBatchDA{
		BatchIndex: batchIndex,
	}
}

func (f *RevertBatchDA) DAType() DAType {
	return RevertBatch
}

func (f *RevertBatchDA) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}

type FinalizeBatchDA struct {
	BatchIndex uint64

	L1BlockNumber uint64
}

func NewFinalizeBatchDA(batchIndex uint64) DAEntry {
	return &FinalizeBatchDA{
		BatchIndex: batchIndex,
	}
}

func (f *FinalizeBatchDA) DAType() DAType {
	return FinalizeBatch
}

func (f *FinalizeBatchDA) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}

type FinalizeBatchDAV3 struct {
	BatchIndex uint64

	L1BlockNumber uint64
}

func NewFinalizeBatchDAV3(batchIndex uint64) DAEntry {
	return &FinalizeBatchDAV3{
		BatchIndex: batchIndex,
	}
}

func (f *FinalizeBatchDAV3) DAType() DAType {
	return FinalizeBatchV3
}

func (f *FinalizeBatchDAV3) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}
