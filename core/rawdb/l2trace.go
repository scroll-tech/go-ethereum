package rawdb

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// ReadEvmTraces retrieves all the evmTraces belonging to a block.
func ReadEvmTraces(db ethdb.Reader, hash common.Hash) []*types.ExecutionResult {
	data, _ := db.Get(evmTracesKey(hash))
	if len(data) == 0 {
		return nil
	}
	var evmTraces []*types.ExecutionResult
	if err := rlp.DecodeBytes(data, &evmTraces); err != nil {
		log.Error("Failed to decode evmTraces", "err", err)
		return nil
	}
	return evmTraces
}

// WriteEvmTraces stores evmTrace list into leveldb.
func WriteEvmTraces(db ethdb.KeyValueWriter, hash common.Hash, evmTraces []*types.ExecutionResult) {
	bytes, err := rlp.EncodeToBytes(evmTraces)
	if err != nil {
		log.Crit("Failed to RLP encode evmTraces", "err", err)
	}
	db.Put(evmTracesKey(hash), bytes)
}

// DeleteEvmTraces removes all evmTraces with a block hash.
func DeleteEvmTraces(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(evmTracesKey(hash)); err != nil {
		log.Crit("Failed to delete evmTraces", "err", err)
	}
}
