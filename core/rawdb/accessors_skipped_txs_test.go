package rawdb

import (
	"math/big"
	"sync"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

func TestReadWriteNumSkippedL1Messages(t *testing.T) {
	blockNumbers := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()
	for _, num := range blockNumbers {
		WriteNumSkippedL1Messages(db, num)
		got := ReadNumSkippedL1Messages(db)

		if got != num {
			t.Fatal("Num L1 messages mismatch", "expected", num, "got", got)
		}
	}
}

func newTestTransaction(queueIndex uint64) *types.Transaction {
	l1msg := types.L1MessageTx{
		QueueIndex: queueIndex,
		Gas:        0,
		To:         &common.Address{},
		Value:      big.NewInt(0),
		Data:       nil,
		Sender:     common.Address{},
	}
	return types.NewTx(&l1msg)
}

func TestReadWriteSkippedTransaction(t *testing.T) {
	tx := newTestTransaction(123)
	db := NewMemoryDatabase()
	WriteSkippedTransaction(db, tx, "random reason", 1, &common.Hash{1})
	got := ReadSkippedTransaction(db, tx.Hash())
	if got == nil || got.Tx.Hash() != tx.Hash() || got.Reason != "random reason" || got.BlockNumber != 1 || got.BlockHash == nil || *got.BlockHash != (common.Hash{1}) {
		t.Fatal("Skipped transaction mismatch", "got", got)
	}
}

func TestReadWriteSkippedL1Message(t *testing.T) {
	tx := newTestTransaction(123)
	db := NewMemoryDatabase()
	WriteSkippedL1Message(db, tx, "random reason", 1, &common.Hash{1})
	got := ReadSkippedTransaction(db, tx.Hash())
	if got == nil || got.Tx.Hash() != tx.Hash() || got.Reason != "random reason" || got.BlockNumber != 1 || got.BlockHash == nil || *got.BlockHash != (common.Hash{1}) {
		t.Fatal("Skipped transaction mismatch", "got", got)
	}
	count := ReadNumSkippedL1Messages(db)
	if count != 1 {
		t.Fatal("Skipped transaction count mismatch", "expected", 1, "got", count)
	}
	hash := ReadSkippedL1MessageHash(db, 0)
	if hash == nil || *hash != tx.Hash() {
		t.Fatal("Skipped L1 message hash mismatch", "expected", tx.Hash(), "got", hash)
	}
}

func TestSkippedL1MessageConcurrentUpdate(t *testing.T) {
	count := 20
	tx := newTestTransaction(123)
	db := NewMemoryDatabase()
	var wg sync.WaitGroup
	for ii := 0; ii < count; ii++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			WriteSkippedL1Message(db, tx, "random reason", 1, &common.Hash{1})
		}()
	}
	wg.Wait()
	got := ReadNumSkippedL1Messages(db)
	if got != uint64(count) {
		t.Fatal("Skipped transaction count mismatch", "expected", count, "got", got)
	}
}

func TestIterateSkippedL1Messages(t *testing.T) {
	db := NewMemoryDatabase()

	txs := []*types.Transaction{
		newTestTransaction(1),
		newTestTransaction(2),
		newTestTransaction(3),
		newTestTransaction(4),
		newTestTransaction(5),
	}

	for _, tx := range txs {
		WriteSkippedL1Message(db, tx, "random reason", 1, &common.Hash{1})
	}

	// simulate skipped L2 tx that's not included in the index
	l2tx := newTestTransaction(6)
	WriteSkippedTransaction(db, l2tx, "random reason", 1, &common.Hash{1})

	it := IterateSkippedTransactionsFrom(db, 2)
	defer it.Release()

	for ii := 2; ii < len(txs); ii++ {
		finished := !it.Next()
		if finished {
			t.Fatal("Iterator terminated early", "ii", ii)
		}

		index := it.Index()
		if index != uint64(ii) {
			t.Fatal("Invalid skipped L1 message index", "expected", ii, "got", index)
		}

		hash := it.TransactionHash()
		if hash != txs[ii].Hash() {
			t.Fatal("Invalid skipped L1 message hash", "expected", txs[ii].Hash(), "got", hash)
		}
	}

	finished := !it.Next()
	if !finished {
		t.Fatal("Iterator did not terminate")
	}
}
