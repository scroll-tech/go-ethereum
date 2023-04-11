package rawdb

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

func TestReadWriteSyncedL1BlockNumber(t *testing.T) {
	blockNumbers := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()
	for _, num := range blockNumbers {
		WriteSyncedL1BlockNumber(db, num)
		got := ReadSyncedL1BlockNumber(db)

		if got == nil || *got != num {
			t.Fatal("Block number mismatch", "expected", num, "got", got)
		}
	}
}

func newL1MessageTx(enqueueIndex uint64) types.L1MessageTx {
	return types.L1MessageTx{
		Nonce:  enqueueIndex,
		Gas:    0,
		To:     &common.Address{},
		Value:  big.NewInt(0),
		Data:   nil,
		Sender: common.Address{},
	}
}

func TestReadWriteL1Message(t *testing.T) {
	enqueueIndex := uint64(123)
	msg := newL1MessageTx(enqueueIndex)
	db := NewMemoryDatabase()
	WriteL1Messages(db, []types.L1MessageTx{msg})
	got := ReadL1Message(db, enqueueIndex)
	if got == nil || got.Nonce != enqueueIndex {
		t.Fatal("L1 message mismatch", "expected", enqueueIndex, "got", got)
	}
}

func TestIterateL1Message(t *testing.T) {
	msgs := []types.L1MessageTx{
		newL1MessageTx(100),
		newL1MessageTx(101),
		newL1MessageTx(103),
		newL1MessageTx(200),
		newL1MessageTx(1000),
	}

	db := NewMemoryDatabase()
	WriteL1Messages(db, msgs)

	it := IterateL1MessagesFrom(db, 103)
	defer it.Release()

	for ii := 2; ii < len(msgs); ii++ {
		finished := !it.Next()
		if finished {
			t.Fatal("Iterator terminated early", "ii", ii)
		}

		got := it.L1Message()
		if got.Nonce != msgs[ii].Nonce {
			t.Fatal("Invalid result", "expected", msgs[ii].Nonce, "got", got.Nonce)
		}
	}

	finished := !it.Next()
	if !finished {
		t.Fatal("Iterator did not terminate")
	}
}

func TestReadL1MessageTxRange(t *testing.T) {
	msgs := []types.L1MessageTx{
		newL1MessageTx(100),
		newL1MessageTx(101),
		newL1MessageTx(103),
		newL1MessageTx(200),
		newL1MessageTx(1000),
	}

	db := NewMemoryDatabase()
	WriteL1Messages(db, msgs)

	got := ReadL1MessagesInRange(db, 101, 999, false)

	if len(got) != 3 {
		t.Fatal("Invalid length", "expected", 3, "got", len(got))
	}

	if got[0].Nonce != 101 || got[1].Nonce != 103 || got[2].Nonce != 200 {
		t.Fatal("Invalid result", "got", got)
	}
}

func TestReadWriteLastL1MessageInL2Block(t *testing.T) {
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
		WriteLastL1MessageInL2Block(db, l2BlockHash, num)
		got := ReadLastL1MessageInL2Block(db, l2BlockHash)

		if got == nil || *got != num {
			t.Fatal("Enqueue index mismatch", "expected", num, "got", got)
		}
	}
}
