package core

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/trie"
)

func decodeProof(proofs []string, db *memorydb.Database, onNode func(*trie.Node)) {
	for _, nodestr := range proofs {
		buf, err := hexutil.Decode(nodestr)
		if err == nil {
			err = trie.DecodeSMTProof(buf, db, onNode)
		}
		if err != nil {
			log.Warn("decode proof string fail", "error", err)
		}
	}
}

// Fill smtproof field for execResult
func (bc *BlockChain) writeSMTProof(state *state.StateDB, execResult *types.ExecutionResult) error {

	underlayerDb := memorydb.New()

	_, err := trie.NewSecure(
		*execResult.Storage.RootBefore,
		trie.NewDatabase(underlayerDb),
	)

	if err != nil {
		return fmt.Errorf("smt create failure: %s", err)
	}

	storage := execResult.Storage
	//accounts := make(map[string]*types.StateAccount)
	handleAccount := func(n *trie.Node) {
		//n.Type
	}

	// start with from/to's data
	decodeProof(storage.ProofFrom, underlayerDb, handleAccount)
	decodeProof(storage.ProofTo, underlayerDb, handleAccount)

	// now trace every OP which could cause changes on state:
	//for i, log := execResult.StructLogs {

	//}

	return nil
}
