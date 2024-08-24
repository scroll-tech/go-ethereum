package rawdb

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
)

func TestWriteRollupEventSyncedL1BlockNumber(t *testing.T) {
	blockNumbers := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()

	// read non-existing value
	if got := ReadRollupEventSyncedL1BlockNumber(db); got != nil {
		t.Fatal("Expected 0 for non-existing value", "got", *got)
	}

	for _, num := range blockNumbers {
		WriteRollupEventSyncedL1BlockNumber(db, num)
		got := ReadRollupEventSyncedL1BlockNumber(db)

		if *got != num {
			t.Fatal("Block number mismatch", "expected", num, "got", got)
		}
	}
}

func TestFinalizedL2BlockNumber(t *testing.T) {
	blockNumbers := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()

	// read non-existing value
	if got := ReadFinalizedL2BlockNumber(db); got != nil {
		t.Fatal("Expected nil for non-existing value", "got", *got)
	}

	for _, num := range blockNumbers {
		WriteFinalizedL2BlockNumber(db, num)
		got := ReadFinalizedL2BlockNumber(db)

		if *got != num {
			t.Fatal("Block number mismatch", "expected", num, "got", got)
		}
	}
}

func TestLastFinalizedBatchIndex(t *testing.T) {
	batchIndxes := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()

	// read non-existing value
	if got := ReadLastFinalizedBatchIndex(db); got != nil {
		t.Fatal("Expected nil for non-existing value", "got", *got)
	}

	for _, num := range batchIndxes {
		WriteLastFinalizedBatchIndex(db, num)
		got := ReadLastFinalizedBatchIndex(db)

		if *got != num {
			t.Fatal("Batch index mismatch", "expected", num, "got", got)
		}
	}
}

func TestFinalizedBatchMeta(t *testing.T) {
	batches := []*FinalizedBatchMeta{
		{
			BatchHash:            common.BytesToHash([]byte("batch1")),
			TotalL1MessagePopped: 123,
			StateRoot:            common.BytesToHash([]byte("stateRoot1")),
			WithdrawRoot:         common.BytesToHash([]byte("withdrawRoot1")),
		},
		{
			BatchHash:            common.BytesToHash([]byte("batch2")),
			TotalL1MessagePopped: 456,
			StateRoot:            common.BytesToHash([]byte("stateRoot2")),
			WithdrawRoot:         common.BytesToHash([]byte("withdrawRoot2")),
		},
		{
			BatchHash:            common.BytesToHash([]byte("batch3")),
			TotalL1MessagePopped: 789,
			StateRoot:            common.BytesToHash([]byte("stateRoot3")),
			WithdrawRoot:         common.BytesToHash([]byte("withdrawRoot3")),
		},
	}

	db := NewMemoryDatabase()

	for i, batch := range batches {
		batchIndex := uint64(i)
		WriteFinalizedBatchMeta(db, batchIndex, batch)
	}

	for i, batch := range batches {
		batchIndex := uint64(i)
		readBatch := ReadFinalizedBatchMeta(db, batchIndex)
		if readBatch == nil {
			t.Fatal("Failed to read batch from database")
		}
		if readBatch.BatchHash != batch.BatchHash || readBatch.TotalL1MessagePopped != batch.TotalL1MessagePopped ||
			readBatch.StateRoot != batch.StateRoot || readBatch.WithdrawRoot != batch.WithdrawRoot {
			t.Fatal("Mismatch in read batch", "expected", batch, "got", readBatch)
		}
	}

	// over-write
	newBatch := &FinalizedBatchMeta{
		BatchHash:            common.BytesToHash([]byte("newBatch")),
		TotalL1MessagePopped: 999,
		StateRoot:            common.BytesToHash([]byte("newStateRoot")),
		WithdrawRoot:         common.BytesToHash([]byte("newWithdrawRoot")),
	}
	WriteFinalizedBatchMeta(db, 0, newBatch) // over-writing the batch with index 0
	readBatch := ReadFinalizedBatchMeta(db, 0)
	if readBatch.BatchHash != newBatch.BatchHash || readBatch.TotalL1MessagePopped != newBatch.TotalL1MessagePopped ||
		readBatch.StateRoot != newBatch.StateRoot || readBatch.WithdrawRoot != newBatch.WithdrawRoot {
		t.Fatal("Mismatch after over-writing batch", "expected", newBatch, "got", readBatch)
	}

	// read non-existing value
	nonExistingIndex := uint64(len(batches) + 1)
	readBatch = ReadFinalizedBatchMeta(db, nonExistingIndex)
	if readBatch != nil {
		t.Fatal("Expected nil for non-existing value", "got", readBatch)
	}
}

func TestWriteReadCommittedBatchMeta(t *testing.T) {
	db := NewMemoryDatabase()

	testCases := []struct {
		batchIndex uint64
		meta       *CommittedBatchMeta
	}{
		{
			batchIndex: 0,
			meta: &CommittedBatchMeta{
				Version:             0,
				BlobVersionedHashes: []common.Hash{},
				ChunkBlockRanges:    []*ChunkBlockRange{},
			},
		},
		{
			batchIndex: 1,
			meta: &CommittedBatchMeta{
				Version:             1,
				BlobVersionedHashes: []common.Hash{common.HexToHash("0x1234")},
				ChunkBlockRanges:    []*ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 10}},
			},
		},
		{
			batchIndex: 255,
			meta: &CommittedBatchMeta{
				Version:             255,
				BlobVersionedHashes: []common.Hash{common.HexToHash("0xabcd"), common.HexToHash("0xef01")},
				ChunkBlockRanges:    []*ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 10}, {StartBlockNumber: 11, EndBlockNumber: 20}},
			},
		},
	}

	for _, tc := range testCases {
		WriteCommittedBatchMeta(db, tc.batchIndex, tc.meta)
		got := ReadCommittedBatchMeta(db, tc.batchIndex)

		if got == nil {
			t.Fatalf("Expected non-nil value for batch index %d", tc.batchIndex)
		}

		if !compareCommittedBatchMeta(tc.meta, got) {
			t.Fatalf("CommittedBatchMeta mismatch for batch index %d, expected %+v, got %+v", tc.batchIndex, tc.meta, got)
		}
	}

	// reading a non-existing value
	if got := ReadCommittedBatchMeta(db, 256); got != nil {
		t.Fatalf("Expected nil for non-existing value, got %+v", got)
	}
}

func TestOverwriteCommittedBatchMeta(t *testing.T) {
	db := NewMemoryDatabase()

	batchIndex := uint64(42)
	initialMeta := &CommittedBatchMeta{
		Version:             1,
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x1234")},
		ChunkBlockRanges:    []*ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 10}},
	}
	newMeta := &CommittedBatchMeta{
		Version:             2,
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x5678"), common.HexToHash("0x9abc")},
		ChunkBlockRanges:    []*ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 20}, {StartBlockNumber: 21, EndBlockNumber: 30}},
	}

	// write initial meta
	WriteCommittedBatchMeta(db, batchIndex, initialMeta)
	got := ReadCommittedBatchMeta(db, batchIndex)

	if !compareCommittedBatchMeta(initialMeta, got) {
		t.Fatalf("Initial write failed, expected %+v, got %+v", initialMeta, got)
	}

	// overwrite with new meta
	WriteCommittedBatchMeta(db, batchIndex, newMeta)
	got = ReadCommittedBatchMeta(db, batchIndex)

	if !compareCommittedBatchMeta(newMeta, got) {
		t.Fatalf("Overwrite failed, expected %+v, got %+v", newMeta, got)
	}

	// read non-existing batch index
	nonExistingIndex := uint64(999)
	got = ReadCommittedBatchMeta(db, nonExistingIndex)

	if got != nil {
		t.Fatalf("Expected nil for non-existing batch index, got %+v", got)
	}
}

func compareCommittedBatchMeta(a, b *CommittedBatchMeta) bool {
	if a.Version != b.Version {
		return false
	}
	if len(a.BlobVersionedHashes) != len(b.BlobVersionedHashes) {
		return false
	}
	for i := range a.BlobVersionedHashes {
		if a.BlobVersionedHashes[i] != b.BlobVersionedHashes[i] {
			return false
		}
	}
	if len(a.ChunkBlockRanges) != len(b.ChunkBlockRanges) {
		return false
	}
	for i := range a.ChunkBlockRanges {
		if a.ChunkBlockRanges[i].StartBlockNumber != b.ChunkBlockRanges[i].StartBlockNumber || a.ChunkBlockRanges[i].EndBlockNumber != b.ChunkBlockRanges[i].EndBlockNumber {
			return false
		}
	}
	return true
}
