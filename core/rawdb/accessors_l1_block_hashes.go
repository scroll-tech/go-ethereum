package rawdb

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"time"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rlp"
)

func WriteL1BlockHashesSyncedBlockNumber(db ethdb.KeyValueWriter, l1BlockNumber uint64) {
	value := big.NewInt(0).SetUint64(l1BlockNumber).Bytes()

	if err := db.Put(syncedL1BlockHashesTxBlockNumberKey, value); err != nil {
		log.Crit("Failed to update l1BlockHashes synced L1 block number", "err", err)
	}
}

// ReadL1BlockHashesSyncedL1BlockNumber retrieves the highest synced L1 block number.
func ReadL1BlockHashesSyncedL1BlockNumber(db ethdb.Reader) *uint64 {
	data, err := db.Get(syncedL1BlockHashesTxBlockNumberKey)
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to read synced L1BlockHashes block number from database", "err", err)
	}
	if len(data) == 0 {
		return nil
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("Unexpected synced L1BlockHashes block number in database", "number", number)
	}

	value := number.Uint64()
	return &value
}

func WriteFirstL1BlockNumberNotInL2Block(db ethdb.KeyValueWriter, l2BlockHash common.Hash, l1BlockNumber uint64) {
	if err := db.Put(FirstL1BlockNumberNotInL2Block(l2BlockHash), encodeBigEndian(l1BlockNumber)); err != nil {
		log.Crit("Failed to store first L1 BlockNumber not in L2 Block", "l2BlockHash", l2BlockHash, "l1BlockNumber", l1BlockNumber, "err", err)
	}
}

func ReadFirstL1BlockNumberNotInL2Block(db ethdb.Reader, l2BlockHash common.Hash) *uint64 {
	data, err := db.Get(FirstL1BlockNumberNotInL2Block(l2BlockHash))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to read first L1 BlockNumber not in L2 Block from database", "l2BlockHash", l2BlockHash, "err", err)
	}
	if len(data) == 0 {
		return nil
	}
	l1BlockNumber := binary.BigEndian.Uint64(data)
	return &l1BlockNumber
}

func WriteL1BlockHashesTxForL2BlockHash(db ethdb.KeyValueWriter, l2BlockHash common.Hash, l1BlockHashesTx types.L1BlockHashesTx) {
	bytes, err := rlp.EncodeToBytes(l1BlockHashesTx)
	if err != nil {
		log.Crit("Failed to RLP encode L1BlockHashesTx for L2BlockHash", "err", err)
	}
	if err := db.Put(L1BlockHashesTxForL2BlockHash(l2BlockHash), bytes); err != nil {
		log.Crit("Failed to store L1BlockHashesTx for L2BlockHash", "err", err)
	}
}

func ReadL1BlockHashesTxForL2BlockHash(db ethdb.Reader, l2BlockHash common.Hash) *types.L1BlockHashesTx {
	data := readL1BlockHashRLPL2BlockHash(db, l2BlockHash)
	if len(data) == 0 {
		return nil
	}
	l1BlockHashesTx := new(types.L1BlockHashesTx)
	if err := rlp.Decode(bytes.NewReader(data), l1BlockHashesTx); err != nil {
		log.Crit("Invalid L1BlockHashesTx RLP", "l2BlockHash", l2BlockHash, "data", data, "err", err)
	}
	return l1BlockHashesTx
}

func readL1BlockHashRLPL2BlockHash(db ethdb.Reader, l2BlockHash common.Hash) rlp.RawValue {
	data, err := db.Get(L1BlockHashesTxForL2BlockHash(l2BlockHash))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load L1BlockNumberHash", "l2BlockHash", l2BlockHash, "err", err)
	}
	return data
}

func WriteL1BlockNumberHashes(db ethdb.KeyValueWriter, l1BlockHashes []common.Hash, start uint64) {
	for i := 0; i < len(l1BlockHashes); i++ {
		log.Debug("Writing L1BlockNumberHash", "number", start+uint64(i), "hash", l1BlockHashes)
		writeL1BlockNumberHash(db, start+uint64(i), l1BlockHashes[i])
	}
}

func writeL1BlockNumberHash(db ethdb.KeyValueWriter, l1BlockNumber uint64, l1BlockHash common.Hash) {
	bytes, err := rlp.EncodeToBytes(l1BlockHash)
	if err != nil {
		log.Crit("Failed to RLP encode L1BlockHash", "err", err)
	}

	if err := db.Put(L1BlockNumberHashKey(l1BlockNumber), bytes); err != nil {
		log.Crit("Failed to store L1BlockNumberHash", "err", err)
	}
}

func ReadL1BlockHashesRange(db ethdb.Reader, from uint64, to uint64) []byte {
	var result []byte
	for i := from; i <= to; i++ {
		result = append(result, readL1BlockNumberHash(db, i).Bytes()...)
	}

	return result
}

func readL1BlockNumberHash(db ethdb.Reader, l1blockNumber uint64) common.Hash {
	data := readL1BlockNumberRLP(db, l1blockNumber)
	if len(data) == 0 {
		return common.Hash{}
	}
	l1blockHash := new(common.Hash)
	if err := rlp.Decode(bytes.NewReader(data), l1blockHash); err != nil {
		log.Crit("Invalid L1BlockNumberHash RLP", "l1BlockNumber", l1blockNumber, "data", data, "err", err)
	}
	return *l1blockHash
}

func readL1BlockNumberRLP(db ethdb.Reader, l1BlockNumber uint64) rlp.RawValue {
	data, err := db.Get(L1BlockNumberHashKey(l1BlockNumber))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load L1BlockNumberHash", "l1BlockNumber", l1BlockNumber, "err", err)
	}
	return data
}

var (
	// L1 message iterator metrics
	iteratorBlockHashesNextCalledCounter      = metrics.NewRegisteredCounter("rawdb/l1_block_hashes/iterator/next_called", nil)
	iteratorBlockHashesInnerNextCalledCounter = metrics.NewRegisteredCounter("rawdb/l1_block_hashes/iterator/inner_next_called", nil)
	iteratorBlockHashesLengthMismatchCounter  = metrics.NewRegisteredCounter("rawdb/l1_block_hashes/iterator/length_mismatch", nil)
	iteratorBlockHashesNextDurationTimer      = metrics.NewRegisteredTimer("rawdb/l1_block_hashes/iterator/next_time", nil)
	iteratorBlockHashesL1BlockHashSizeGauge   = metrics.NewRegisteredGauge("rawdb/l1_block_hashes/size", nil)
)

type L1BlockHashesIterator struct {
	inner          ethdb.Iterator
	keyLength      int
	maxBlockNumber uint64
}

func IterateL1BlockHashesFrom(db ethdb.Database, from uint64) L1BlockHashesIterator {
	start := encodeBigEndian(from)
	it := db.NewIterator(l1BlockPrefix, start)
	keyLength := len(l1BlockPrefix) + 8
	maxBlock := ReadL1BlockHashesSyncedL1BlockNumber(db)
	maxBlockNumber := from

	if maxBlock != nil {
		maxBlockNumber = *maxBlock
	}

	return L1BlockHashesIterator{
		inner:          it,
		keyLength:      keyLength,
		maxBlockNumber: maxBlockNumber,
	}
}

// Next moves the iterator to the next key/value pair.
// It returns false when the iterator is exhausted.
// TODO: Consider reading items in batches.
func (it *L1BlockHashesIterator) Next() bool {
	iteratorBlockHashesNextCalledCounter.Inc(1)

	defer func(t0 time.Time) {
		iteratorBlockHashesNextDurationTimer.Update(time.Since(t0))
	}(time.Now())

	for it.inner.Next() {
		iteratorBlockHashesInnerNextCalledCounter.Inc(1)

		key := it.inner.Key()
		if len(key) == it.keyLength {
			return true
		} else {
			iteratorBlockHashesLengthMismatchCounter.Inc(1)
		}
	}
	return false
}

func (it *L1BlockHashesIterator) L1BlockHash() common.Hash {
	data := it.inner.Value()

	l1blockHash := new(common.Hash)
	if err := rlp.Decode(bytes.NewReader(data), l1blockHash); err != nil {
		log.Crit("Invalid L1BlockNumberHash RLP", "data", data, "err", err)
	}
	return *l1blockHash
}

// Release releases the associated resources.
func (it *L1BlockHashesIterator) Release() {
	it.inner.Release()
}

func ReadL1BlockHashes(db ethdb.Database, startIndex, maxCount uint64) ([]common.Hash, uint64) {
	blockHashes := make([]common.Hash, 0, maxCount)
	it := IterateL1BlockHashesFrom(db, startIndex)
	defer it.Release()

	index := startIndex
	count := maxCount

	for count > 0 && it.Next() {
		blockHash := it.L1BlockHash()

		blockHashes = append(blockHashes, blockHash)
		index += 1
		count -= 1

		iteratorBlockHashesL1BlockHashSizeGauge.Update(int64(unsafe.Sizeof(blockHash) + uintptr(cap(blockHash)))) // TODO(l1blockhashes)

		// TODO: check to stop if it.maxBlockNumber == blockhash number
	}

	if len(blockHashes) == 0 && startIndex == 0 {
		return blockHashes, 0
	}

	return blockHashes, startIndex + uint64(len(blockHashes)) - uint64(1)
}
