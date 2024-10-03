package trie

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// EmptyRoot indicate what root is for an empty trie, it depends on its underlying implement (zktrie or common trie)
func (db *Database) EmptyRoot() common.Hash {
	if db.IsUsingZktrie() {
		return types.EmptyZkTrieRootHash
	} else {
		return types.EmptyRootHash
	}
}
