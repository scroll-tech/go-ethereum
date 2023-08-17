package rawdb

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// mutex used to avoid concurrent updates of NumSkippedL1Messages
var mu sync.Mutex

// WriteNumSkippedL1Messages writes the number of skipped L1 messages to the database.
func WriteNumSkippedL1Messages(db ethdb.KeyValueWriter, numSkipped uint64) {
	value := big.NewInt(0).SetUint64(numSkipped).Bytes()

	if err := db.Put(numSkippedL1MessagesKey, value); err != nil {
		log.Crit("Failed to update the number of skipped L1 messages", "err", err)
	}
}

// ReadNumSkippedL1Messages retrieves the number of skipped messages.
func ReadNumSkippedL1Messages(db ethdb.Reader) uint64 {
	data, err := db.Get(numSkippedL1MessagesKey)
	if err != nil && isNotFoundErr(err) {
		return 0
	}
	if err != nil {
		log.Crit("Failed to read number of skipped L1 messages from database", "err", err)
	}
	if len(data) == 0 {
		return 0
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("Unexpected number of skipped L1 messages in database", "number", number)
	}
	return number.Uint64()
}

// SkippedTransaction stores the transaction object, along with the skip reason and block context.
type SkippedTransaction struct {
	// Tx is the skipped transaction.
	// We store the tx itself because otherwise geth will discard it after skipping.
	Tx *types.Transaction

	// Reason is the skip reason.
	Reason string

	// BlockNumber is the number of the block in which this transaction was skipped.
	BlockNumber uint64

	// BlockNumber is the hash of the block in which this transaction was skipped or nil.
	BlockHash *common.Hash
}

// WriteSkippedTransaction writes a skipped transaction to the database.
func WriteSkippedTransaction(db ethdb.KeyValueWriter, tx *types.Transaction, reason string, blockNumber uint64, blockHash *common.Hash) {
	// workaround: RLP decoding fails if this is nil
	if blockHash == nil {
		blockHash = &common.Hash{}
	}
	stx := SkippedTransaction{Tx: tx, Reason: reason, BlockNumber: blockNumber, BlockHash: blockHash}
	bytes, err := rlp.EncodeToBytes(stx)
	if err != nil {
		log.Crit("Failed to RLP encode skipped transaction", "err", err)
	}
	if err := db.Put(SkippedTransactionKey(tx.Hash()), bytes); err != nil {
		log.Crit("Failed to store skipped transaction", "hash", tx.Hash().String(), "err", err)
	}
}

// ReadSkippedTransactionRLP retrieves a skipped transaction in its raw RLP database encoding.
func ReadSkippedTransactionRLP(db ethdb.Reader, txHash common.Hash) rlp.RawValue {
	data, err := db.Get(SkippedTransactionKey(txHash))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load skipped transaction", "hash", txHash.String(), "err", err)
	}
	return data
}

// ReadSkippedTransaction retrieves a skipped transaction by its hash, along with its skipped reason.
func ReadSkippedTransaction(db ethdb.Reader, txHash common.Hash) *SkippedTransaction {
	data := ReadSkippedTransactionRLP(db, txHash)
	if len(data) == 0 {
		return nil
	}
	var stx SkippedTransaction
	if err := rlp.Decode(bytes.NewReader(data), &stx); err != nil {
		log.Crit("Invalid skipped transaction RLP", "hash", txHash.String(), "data", data, "err", err)
	}
	if stx.BlockHash != nil && *stx.BlockHash == (common.Hash{}) {
		stx.BlockHash = nil
	}
	return &stx
}

// WriteSkippedL1MessageHash writes the hash of a skipped L1 message to the database.
func WriteSkippedL1MessageHash(db ethdb.KeyValueWriter, index uint64, txHash common.Hash) {
	if err := db.Put(SkippedL1MessageHashKey(index), txHash[:]); err != nil {
		log.Crit("Failed to store skipped transaction index", "index", index, "hash", txHash.String(), "err", err)
	}
}

// ReadSkippedL1MessageHash retrieves the hash of a skipped L1 message by its index.
func ReadSkippedL1MessageHash(db ethdb.Reader, index uint64) *common.Hash {
	data, err := db.Get(SkippedL1MessageHashKey(index))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load skipped L1 message index index", "index", index, "err", err)
	}
	hash := common.BytesToHash(data)
	return &hash
}

// WriteSkippedL1Message writes a skipped L1 message to the database and also updates the count and lookup index.
// Note: The lookup index and count will include duplicates if there are chain reorgs.
func WriteSkippedL1Message(db ethdb.Database, tx *types.Transaction, reason string, blockNumber uint64, blockHash *common.Hash) {
	// this method is not accessed concurrently, but just to be sure...
	mu.Lock()
	defer mu.Unlock()

	index := ReadNumSkippedL1Messages(db)

	// update in a batch
	batch := db.NewBatch()
	WriteSkippedTransaction(db, tx, reason, blockNumber, blockHash)
	WriteSkippedL1MessageHash(db, index, tx.Hash())
	WriteNumSkippedL1Messages(db, index+1)

	// write to DB
	if err := batch.Write(); err != nil {
		log.Crit("Failed to store skipped L1 message", "hash", tx.Hash().String(), "err", err)
	}
}

// SkippedTransactionIterator is a wrapper around ethdb.Iterator that
// allows us to iterate over skipped L1 message hashes in the database.
// It implements an interface similar to ethdb.Iterator.
type SkippedTransactionIterator struct {
	inner     ethdb.Iterator
	db        ethdb.Reader
	keyLength int
}

// IterateSkippedTransactionsFrom creates a SkippedTransactionIterator that iterates
// over all skipped L1 message hashes in the database starting at the provided index.
func IterateSkippedTransactionsFrom(db ethdb.Database, index uint64) SkippedTransactionIterator {
	start := encodeBigEndian(index)
	it := db.NewIterator(skippedL1MessageHashPrefix, start)
	keyLength := len(skippedL1MessageHashPrefix) + 8

	return SkippedTransactionIterator{
		inner:     it,
		db:        db,
		keyLength: keyLength,
	}
}

// Next moves the iterator to the next key/value pair.
// It returns false when the iterator is exhausted.
// TODO: Consider reading items in batches.
func (it *SkippedTransactionIterator) Next() bool {
	for it.inner.Next() {
		key := it.inner.Key()
		if len(key) == it.keyLength {
			return true
		}
	}
	return false
}

// Index returns the index of the current skipped L1 message hash.
func (it *SkippedTransactionIterator) Index() uint64 {
	key := it.inner.Key()
	raw := key[len(skippedL1MessageHashPrefix) : len(skippedL1MessageHashPrefix)+8]
	index := binary.BigEndian.Uint64(raw)
	return index
}

// TransactionHash returns the current skipped L1 message hash.
func (it *SkippedTransactionIterator) TransactionHash() common.Hash {
	data := it.inner.Value()
	return common.BytesToHash(data)
}

// Release releases the associated resources.
func (it *SkippedTransactionIterator) Release() {
	it.inner.Release()
}
