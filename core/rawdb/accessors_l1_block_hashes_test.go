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

func TestReadWriteL1BlockHashesTx(t *testing.T) {
	from := uint64(122)
	lastAppliedL1BlockNum := uint64(123)
	blockRangeHash := []common.Hash{
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000080"),
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000081"),
	}
	tx := newL1BlockHashesTx(lastAppliedL1BlockNum, blockRangeHash)

	db := NewMemoryDatabase()
	WriteL1BlockHashesTx(db, tx, from)
	got := ReadL1BlockHashesTx(db, lastAppliedL1BlockNum)
	assert.Equal(t, tx, *got)

	for i := uint64(0); i <= lastAppliedL1BlockNum-from; i++ {
		hash := readL1BlockNumberHash(db, from+i)
		assert.Equal(t, blockRangeHash[i], hash, "l1 block number hash mismatch")
	}
}

func TestReadWriteL1BlockNumberHash(t *testing.T) {
	from := uint64(122)
	lastAppliedL1BlockNum := uint64(123)
	firstHash := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000080")
	secondHash := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000081")
	blockRangeHash := []common.Hash{
		firstHash,
		secondHash,
	}
	concatHashes := append(firstHash.Bytes(), secondHash.Bytes()...)

	db := NewMemoryDatabase()
	for i := uint64(0); i <= lastAppliedL1BlockNum-from; i++ {
		blockNum := from + i
		blockHash := blockRangeHash[i]
		writeL1BlockNumberHash(db, blockNum, blockHash)

		assert.Equal(t, blockHash, readL1BlockNumberHash(db, blockNum))
	}

	assert.Equal(t, concatHashes, ReadL1BlockHashesRange(db, from, lastAppliedL1BlockNum))
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
		WriteL1BlockNumberForL2Block(db, l2BlockHash, num)
		got := ReadL1BlockNumberForL2Block(db, l2BlockHash)

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
