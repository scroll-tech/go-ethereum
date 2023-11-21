package rawdb

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestReadWriteL1BlockHashesSyncedBlockNumber(t *testing.T) {
	blockNumbers := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()
	for _, num := range blockNumbers {
		WriteL1BlockHashesSyncedBlockNumber(db, num)
		got := ReadL1BlockHashesSyncedL1BlockNumber(db)

		if got == nil || *got != num {
			t.Fatal("Block number mismatch", "expected", num, "got", got)
		}
	}
}

func newL1BlockHashesTx(lastAppliedL1BLockNum uint64, blockHashesRange []common.Hash) types.L1BlockHashesTx {
	return types.L1BlockHashesTx{
		LastAppliedL1Block: lastAppliedL1BLockNum,
		BlockHashesRange:   blockHashesRange,
		To:                 &common.Address{},
		Data:               []byte{},
		Sender:             common.Address{},
	}
}

func TestReadWriteL1BlockNumberForL2Block(t *testing.T) {
	inputs := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()
	for _, num := range inputs {
		l2BlockHash := common.Hash{byte(num)}
		WriteFirstL1BlockNumberNotInL2Block(db, l2BlockHash, num)
		got := ReadFirstL1BlockNumberNotInL2Block(db, l2BlockHash)

		if got == nil || *got != num {
			t.Fatal("Enqueue index mismatch", "expected", num, "got", got)
		}
	}
}

func TestReadWriteL1BlockHashesTxForL2BlockHash(t *testing.T) {
	l2BlockHash := common.Hash{byte(255)}
	lastAppliedL1BlockNum := uint64(123)
	tx := newL1BlockHashesTx(lastAppliedL1BlockNum, []common.Hash{})

	db := NewMemoryDatabase()
	WriteL1BlockHashesTxForL2BlockHash(db, l2BlockHash, tx)

	got := ReadL1BlockHashesTxForL2BlockHash(db, l2BlockHash)
	assert.Equal(t, tx, *got)
}

func TestReadL1BlockHashes(t *testing.T) {
	blockRangeHash := []common.Hash{
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000080"),
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000081"),
	}
	expectLastAppliedL1BlockNumber := uint64(1)

	db := NewMemoryDatabase()
	WriteL1BlockNumberHashes(db, blockRangeHash, 0)

	result, lastApplied := ReadL1BlockHashes(db, 0, 10)

	assert.Equal(t, expectLastAppliedL1BlockNumber, lastApplied)
	assert.Equal(t, blockRangeHash, result)
}
