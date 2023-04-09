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
			t.Fatal("Block number mismatch")
		}
	}
}

func newL1MessageTx(enqueueIndex uint64) types.L1MessageTx {
	return types.L1MessageTx{
		Nonce:  enqueueIndex,
		Gas:    0,
		To:     nil,
		Value:  big.NewInt(0),
		Data:   nil,
		Sender: &common.Address{},
	}
}

func TestReadWriteL1Message(t *testing.T) {
	enqueueIndex := uint64(123)
	msg := newL1MessageTx(enqueueIndex)
	db := NewMemoryDatabase()
	WriteL1Messages(db, []types.L1MessageTx{msg})

	got := ReadL1Message(db, enqueueIndex)
	if got == nil || got.Nonce != msg.Nonce {
		t.Fatal("L1 message mismatch", "got", got)
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

	it.Next()
	got := it.L1Message()
	if got.Nonce != 103 {
		t.Fatal("Invalid result", "nonce", got.Nonce)
	}

	it.Next()
	got = it.L1Message()
	if got.Nonce != 200 {
		t.Fatal("Invalid result", "nonce", got.Nonce)
	}

	it.Next()
	got = it.L1Message()
	if got.Nonce != 1000 {
		t.Fatal("Invalid result", "nonce", got.Nonce)
	}

	finished := it.Next()
	if finished {
		t.Fatal("Invalid result", "finished", finished)
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

	got := ReadL1MessagesInRange(db, 100, 199, false)

	if len(got) != 3 {
		t.Fatal("Invalid length", "length", len(got))
	}

	if got[0].Nonce != 100 || got[1].Nonce != 101 || got[2].Nonce != 103 {
		t.Fatal("Invalid result", "got", got)
	}
}

func TestReadWriteL1MessagesInBlock(t *testing.T) {
	hash := common.Hash{1}
	db := NewMemoryDatabase()

	WriteL1MessageRangeInL2Block(db, hash, L1MessageRangeInL2Block{
		FirstEnqueueIndex: 1,
		LastEnqueueIndex:  9,
	})

	got := ReadL1MessageRangeInL2Block(db, hash)

	if got == nil || got.FirstEnqueueIndex != 1 || got.LastEnqueueIndex != 9 {
		t.Fatal("Incorrect result", "got", got)
	}
}
