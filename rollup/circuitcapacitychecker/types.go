package circuitcapacitychecker

import (
	"errors"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/state"
)

var (
	ErrUnknown               = errors.New("unknown circuit capacity checker error")
	ErrTxRowUsageOverflow    = errors.New("tx row usage oveflow")
	ErrBlockRowUsageOverflow = errors.New("block row usage oveflow")
)

// ProofCache is cached in environment and holds data required for trace's storageProof and deletionProof
type ProofCache struct {
	AccountProof []hexutil.Bytes
	StorageProof map[string][]hexutil.Bytes
	StorageTrie  state.Trie
	TrieTracer   state.ZktrieProofTracer
}

func NewProofCache(stateDb *state.StateDB, addr common.Address) *ProofCache {
	var zktrieTracer state.ZktrieProofTracer
	trie, err := stateDb.GetStorageTrieForProof(addr)
	// notice storage trie can be non-existed if the account is not existed
	// we just use empty trie and Non-available tracer
	if err == nil {
		zktrieTracer = stateDb.NewProofTracer(trie)
	}

	return &ProofCache{
		StorageProof: make(map[string][]hexutil.Bytes),
		StorageTrie:  trie,
		TrieTracer:   zktrieTracer,
	}
}
