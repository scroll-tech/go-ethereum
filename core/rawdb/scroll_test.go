package rawdb

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

func TestReadWriteSyncedL1BlockNumber(t *testing.T) {
	blockNumbers := []*big.Int{
		big.NewInt(0).SetUint64(1),
		big.NewInt(0).SetUint64(1 << 2),
		big.NewInt(0).SetUint64(1 << 8),
		big.NewInt(0).SetUint64(1 << 16),
		big.NewInt(0).SetUint64(1 << 32),
	}

	db := NewMemoryDatabase()
	for _, num := range blockNumbers {
		WriteSyncedL1BlockNumber(db, num)
		got := ReadSyncedL1BlockNumber(db)

		if num.Cmp(got) != 0 {
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
	WriteL1Message(db, &msg)

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

	got := ReadLMessagesInRange(db, 100, 199)

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

	WriteL1MessagesInBlock(db, hash, L1MessagesInBlock{
		FirstEnqueueIndex: 1,
		LastEnqueueIndex:  9,
	})

	got := ReadL1MessagesInBlock(db, hash)

	if got == nil || got.FirstEnqueueIndex != 1 || got.LastEnqueueIndex != 9 {
		t.Fatal("Incorrect result", "got", got)
	}
}
