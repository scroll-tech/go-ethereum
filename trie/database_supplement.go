package trie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// EmptyRoot indicate what root is for an empty trie, it depends on its underlying implement (zktrie or common trie)
func (db *Database) EmptyRoot() common.Hash {
	if db.IsUsingZktrie() {
		return common.Hash{}
	} else {
		return types.EmptyRootHash
	}
}
