package rawdb

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// WriteSyncedL1BlockNumber writes the highest synced L1 block number to the database.
func WriteSyncedL1BlockNumber(db ethdb.KeyValueWriter, L1BlockNumber uint64) {
	value := big.NewInt(0).SetUint64(L1BlockNumber).Bytes()

	if err := db.Put(syncedL1BlockNumberKey, value); err != nil {
		log.Crit("Failed to update synced L1 block number", "err", err)
	}
}

// ReadSyncedL1BlockNumber retrieves the highest synced L1 block number.
func ReadSyncedL1BlockNumber(db ethdb.Reader) *uint64 {
	data, err := db.Get(syncedL1BlockNumberKey)
	if err != nil {
		log.Crit("Failed to read synced L1 block number from database", "err", err)
	}
	if len(data) == 0 {
		return nil
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("Unexpected synced L1 block number in database", "number", number)
	}

	value := number.Uint64()
	return &value
}

// WriteL1Message writes an L1 message to the database.
func WriteL1Message(db ethdb.KeyValueWriter, l1Msg types.L1MessageTx) {
	bytes, err := rlp.EncodeToBytes(l1Msg)
	if err != nil {
		log.Crit("Failed to RLP encode L1 message", "err", err)
	}
	enqueueIndex := l1Msg.Nonce
	if err := db.Put(L1MessageKey(enqueueIndex), bytes); err != nil {
		log.Crit("Failed to store L1 message", "err", err)
	}
}

// WriteL1Messages writes an array of L1 messages to the database.
func WriteL1Messages(db ethdb.KeyValueWriter, l1Msgs []types.L1MessageTx) {
	for _, msg := range l1Msgs {
		WriteL1Message(db, msg)
	}
}

// WriteL1MessagesBatch writes an array of L1 messages to the database in a single batch.
func WriteL1MessagesBatch(db ethdb.Batcher, l1Msgs []types.L1MessageTx) {
	batch := db.NewBatch()
	WriteL1Messages(batch, l1Msgs)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to store L1 message batch", "err", err)
	}
}

// ReadL1MessageRLP retrieves an L1 message in its raw RLP database encoding.
func ReadL1MessageRLP(db ethdb.Reader, enqueueIndex uint64) rlp.RawValue {
	data, err := db.Get(L1MessageKey(enqueueIndex))
	if err != nil {
		log.Crit("Failed to load L1 message", "enqueueIndex", enqueueIndex, "err", err)
	}
	return data
}

// ReadL1Message retrieves the L1 message corresponding to the enqueue index.
func ReadL1Message(db ethdb.Reader, enqueueIndex uint64) *types.L1MessageTx {
	data := ReadL1MessageRLP(db, enqueueIndex)
	if len(data) == 0 {
		return nil
	}
	l1Msg := new(types.L1MessageTx)
	if err := rlp.Decode(bytes.NewReader(data), l1Msg); err != nil {
		log.Crit("Invalid L1 message RLP", "enqueueIndex", enqueueIndex, "data", data, "err", err)
	}
	return l1Msg
}

// L1MessageIterator is a wrapper around ethdb.Iterator that
// allows us to iterate over L1 messages in the database. It
// implements an interface similar to ethdb.Iterator.
type L1MessageIterator struct {
	inner     ethdb.Iterator
	keyLength int
}

// IterateL1MessagesFrom creates an L1MessageIterator that iterates over
// all L1 message in the database starting at the provided enqueue index.
func IterateL1MessagesFrom(db ethdb.Iteratee, fromEnqueueIndex uint64) L1MessageIterator {
	start := encodeEnqueueIndex(fromEnqueueIndex)
	it := db.NewIterator(L1MessagePrefix, start)
	keyLength := len(L1MessagePrefix) + 8

	return L1MessageIterator{
		inner:     it,
		keyLength: keyLength,
	}
}

// Next moves the iterator to the next key/value pair.
// It returns whether the iterator is exhausted.
func (it *L1MessageIterator) Next() bool {
	for it.inner.Next() {
		key := it.inner.Key()
		if len(key) == it.keyLength {
			return true
		}
	}
	return false
}

// EnqueueIndex returns the enqueue index of the current L1 message.
func (it *L1MessageIterator) EnqueueIndex() uint64 {
	key := it.inner.Key()
	raw := key[len(L1MessagePrefix) : len(L1MessagePrefix)+8]
	enqueueIndex := binary.BigEndian.Uint64(raw)
	return enqueueIndex
}

// L1Message returns the current L1 message.
func (it *L1MessageIterator) L1Message() types.L1MessageTx {
	data := it.inner.Value()
	l1Msg := types.L1MessageTx{}
	if err := rlp.DecodeBytes(data, &l1Msg); err != nil {
		log.Crit("Invalid L1 message RLP", "data", data, "err", err)
	}
	return l1Msg
}

// Release releases the associated resources.
func (it *L1MessageIterator) Release() {
	it.inner.Release()
}

// ReadL1MessagesInRange retrieves all L1 messages between two enqueue indices (inclusive).
// The resulting array is ordered by the L1 message enqueue index.
func ReadL1MessagesInRange(db ethdb.Iteratee, firstEnqueueIndex, lastEnqueueIndex uint64, checkRange bool) []types.L1MessageTx {
	if firstEnqueueIndex > lastEnqueueIndex {
		return nil
	}

	expectedCount := lastEnqueueIndex - firstEnqueueIndex + 1
	msgs := make([]types.L1MessageTx, 0, expectedCount)
	it := IterateL1MessagesFrom(db, firstEnqueueIndex)
	defer it.Release()

	for it.Next() {
		if it.EnqueueIndex() > lastEnqueueIndex {
			break
		}
		msgs = append(msgs, it.L1Message())
	}

	if checkRange && uint64(len(msgs)) != expectedCount {
		log.Crit("Missing or unordered L1 messages in database",
			"firstEnqueueIndex", firstEnqueueIndex,
			"lastEnqueueIndex", lastEnqueueIndex,
			"count", len(msgs),
		)
	}

	return msgs
}

// L1MessageRangeInL2Block stores the range of L1 messages included
// in some L2 block. The sync layer is expected to verify that the
// L2 block includes these L1 messages contiguously.
type L1MessageRangeInL2Block struct {
	FirstEnqueueIndex uint64
	LastEnqueueIndex  uint64
}

// WriteL1MessageRangeInL2Block writes the L1 message range included in an
// L2 block into the database. The L2 block is identified by its block hash.
func WriteL1MessageRangeInL2Block(db ethdb.KeyValueWriter, l2BlockHash common.Hash, msgRange L1MessageRangeInL2Block) {
	bytes, err := rlp.EncodeToBytes(msgRange)
	if err != nil {
		log.Crit("Failed to RLP encode L1MessageRangeInL2Block", "range", msgRange, "err", err)
	}
	if err := db.Put(L1MessageRangeInL2BlockKey(l2BlockHash), bytes); err != nil {
		log.Crit("Failed to store L1MessageRangeInL2Block", "hash", l2BlockHash, "err", err)
	}
}

// ReadL1MessageRangeInL2Block retrieves the range of L1 messages included in an L2 block.
func ReadL1MessageRangeInL2Block(db ethdb.Reader, l2BlockHash common.Hash) *L1MessageRangeInL2Block {
	data, err := db.Get(L1MessageRangeInL2BlockKey(l2BlockHash))
	if err != nil {
		log.Crit("Failed to read L1MessageRangeInL2Block from database", "err", err)
	}
	if len(data) == 0 {
		return nil
	}
	var msgRange L1MessageRangeInL2Block
	if err := rlp.DecodeBytes(data, &msgRange); err != nil {
		log.Crit("Invalid L1MessageRangeInL2Block RLP", "hash", l2BlockHash, "data", data, "err", err)
	}
	return &msgRange
}
