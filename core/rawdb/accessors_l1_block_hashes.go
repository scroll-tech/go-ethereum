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

// L1BlockHashesTx

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

func WriteL1BlockHashesTx(db ethdb.KeyValueWriter, l1BlockHashesTx types.L1BlockHashesTx) {
	bytes, err := rlp.EncodeToBytes(l1BlockHashesTx)
	if err != nil {
		log.Crit("Failed to RLP encode L1BlockHashesTx", "err", err)
	}
	if err := db.Put(L1BlockHashesKey(l1BlockHashesTx.LastAppliedL1Block), bytes); err != nil {
		log.Crit("Failed to store L1BlockHashesTx", "err", err)
	}
}

func ReadL1BlockHashesTx(db ethdb.Reader, lastAppliedL1BlockNumber uint64) *types.L1BlockHashesTx {
	data := readL1BlockHashesTxRLP(db, lastAppliedL1BlockNumber)
	if len(data) == 0 {
		return nil
	}
	l1BlockHashesTx := new(types.L1BlockHashesTx)
	if err := rlp.Decode(bytes.NewReader(data), l1BlockHashesTx); err != nil {
		log.Crit("Invalid L1BlockHashesTx RLP", "lastAppliedL1BlockNumber", lastAppliedL1BlockNumber, "data", data, "err", err)
	}
	return l1BlockHashesTx
}

func readL1BlockHashesTxRLP(db ethdb.Reader, lastAppliedL1BlockNumber uint64) rlp.RawValue {
	data, err := db.Get(L1BlockHashesKey(lastAppliedL1BlockNumber))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load L1BlockHashesTx", "lastAppliedL1BlockNumber", lastAppliedL1BlockNumber, "err", err)
	}
	return data
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
	data := readL1BlockHashRLP(db, l1blockNumber)
	if len(data) == 0 {
		return common.Hash{}
	}
	l1blockHash := new(common.Hash)
	if err := rlp.Decode(bytes.NewReader(data), l1blockHash); err != nil {
		log.Crit("Invalid L1BlockNumberHash RLP", "l1BlockNumber", l1blockNumber, "data", data, "err", err)
	}
	return *l1blockHash
}

func readL1BlockHashRLP(db ethdb.Reader, l1BlockNumber uint64) rlp.RawValue {
	data, err := db.Get(L1BlockNumberHashKey(l1BlockNumber))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load L1BlockNumberHash", "l1BlockNumber", l1BlockNumber, "err", err)
	}
	return data
}
