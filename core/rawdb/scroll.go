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

//
func WriteSyncedL1BlockNumber(db ethdb.KeyValueWriter, blockNumber uint64) {
	value := new(big.Int).SetUint64(blockNumber).Bytes()
	if blockNumber == 0 {
		value = []byte{0}
	}
	if err := db.Put(syncedL1BlockNumberKey, value); err != nil {
		log.Crit("Failed to synced L1 block number", "err", err)
	}
}

//
func ReadSyncedL1BlockNumber(db ethdb.Reader) *uint64 {
	data, _ := db.Get(syncedL1BlockNumberKey)
	if len(data) == 0 {
		return nil
	}
	ret := new(big.Int).SetBytes(data).Uint64()
	return &ret
}

//
func WriteL1Message(db ethdb.KeyValueWriter, msg *types.L1MessageTx) {
	bytes, err := rlp.EncodeToBytes(msg)
	if err != nil {
		log.Crit("Failed to RLP encode L1 message", "err", err)
	}
	enqueueIndex := msg.Nonce
	if err := db.Put(L1MessageKey(enqueueIndex), bytes); err != nil {
		log.Crit("Failed to store L1 message", "err", err)
	}
}

//
// TODO: consider writing messages in batches
func WriteL1Messages(db ethdb.KeyValueWriter, msgs []types.L1MessageTx) {
	for _, msg := range msgs {
		WriteL1Message(db, &msg)
	}
}

//
func ReadL1MessageRLP(db ethdb.Reader, enqueueIndex uint64) rlp.RawValue {
	data, err := db.Get(L1MessageKey(enqueueIndex))
	if err != nil {
		log.Crit("Failed to load L1 message", "enqueueIndex", enqueueIndex, "err", err)
	}
	return data
}

//
func ReadL1Message(db ethdb.Reader, enqueueIndex uint64) *types.L1MessageTx {
	data := ReadL1MessageRLP(db, enqueueIndex)
	if len(data) == 0 {
		return nil
	}
	msg := new(types.L1MessageTx)
	if err := rlp.Decode(bytes.NewReader(data), msg); err != nil {
		log.Crit("Invalid L1 message RLP", "enqueueIndex", enqueueIndex, "err", err)
	}
	return msg
}

type L1MessageIterator struct {
	inner     ethdb.Iterator
	keyLength int
}

func IterateL1MessagesFrom(db ethdb.Iteratee, from uint64) L1MessageIterator {
	start := encodeEnqueueIndex(from)
	it := db.NewIterator(L1MessagePrefix, start)
	keyLength := len(L1MessagePrefix) + 8

	return L1MessageIterator{
		inner:     it,
		keyLength: keyLength,
	}
}

func (it *L1MessageIterator) Next() bool {
	for it.inner.Next() {
		key := it.inner.Key()
		if len(key) == it.keyLength {
			return true
		}
	}
	return false
}

func (it *L1MessageIterator) EnqueueIndex() uint64 {
	key := it.inner.Key()
	enqueueIndex := binary.BigEndian.Uint64(key[len(L1MessagePrefix) : len(L1MessagePrefix)+8])
	return enqueueIndex
}

func (it *L1MessageIterator) L1Message() types.L1MessageTx {
	data := it.inner.Value()
	msg := types.L1MessageTx{}
	if err := rlp.DecodeBytes(data, &msg); err != nil {
		log.Crit("Invalid L1 message RLP", "err", err)
	}
	return msg
}

func (it *L1MessageIterator) Release() {
	it.inner.Release()
}

//
func ReadLMessagesInRange(db ethdb.Iteratee, first, last uint64) []types.L1MessageTx {
	msgs := make([]types.L1MessageTx, 0, last-first+1)
	it := IterateL1MessagesFrom(db, first)
	defer it.Release()

	for it.Next() {
		if it.EnqueueIndex() > last {
			break
		}
		msgs = append(msgs, it.L1Message())
	}

	return msgs
}

type L1MessagesInBlock struct {
	FirstEnqueueIndex uint64
	LastEnqueueIndex  uint64
}

//
func WriteL1MessagesInBlock(db ethdb.KeyValueWriter, hash common.Hash, entry L1MessagesInBlock) {
	bytes, err := rlp.EncodeToBytes(entry)
	if err != nil {
		log.Crit("Failed to RLP encode L1 messages in block", "err", err)
	}
	if err := db.Put(L1MessagesInBlockKey(hash), bytes); err != nil {
		log.Crit("Failed to store L1 messages in block", "hash", hash, "err", err)
	}
}

//
func ReadL1MessagesInBlock(db ethdb.Reader, hash common.Hash) *L1MessagesInBlock {
	data, _ := db.Get(L1MessagesInBlockKey(hash))
	if len(data) == 0 {
		return nil
	}
	var entry L1MessagesInBlock
	if err := rlp.DecodeBytes(data, &entry); err != nil {
		log.Error("Invalid L1 messages in block RLP", "hash", hash, "blob", data, "err", err)
		return nil
	}
	return &entry
}
