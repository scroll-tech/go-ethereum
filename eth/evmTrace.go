package eth

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/internal/ethapi"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// ReadEvmTraces retrieves all the evmTraces belonging to a block.
func (s *Ethereum) ReadEvmTraces(hash common.Hash) []*ethapi.ExecutionResult {
	data, _ := s.chainDb.Get(rawdb.EvmTracesKey(hash))
	if len(data) == 0 {
		return nil
	}
	var evmTraces []*ethapi.ExecutionResult
	if err := rlp.DecodeBytes(data, &evmTraces); err != nil {
		log.Error("Failed to decode evmTraces", "err", err)
	}
	return evmTraces
}

// WriteEvmTraces stores evmTrace list into leveldb.
func (s *Ethereum) WriteEvmTraces(hash common.Hash, evmTraces []*ethapi.ExecutionResult) error {
	bytes, err := rlp.EncodeToBytes(evmTraces)
	if err != nil {
		return err
	}
	return s.chainDb.Put(rawdb.EvmTracesKey(hash), bytes)
}

// DeleteEvmTraces removes all evmTraces with a block hash.
func (s *Ethereum) DeleteEvmTraces(hash common.Hash) error {
	return s.chainDb.Delete(rawdb.EvmTracesKey(hash))
}
