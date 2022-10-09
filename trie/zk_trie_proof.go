package trie

import (
	zktrie "github.com/scroll-tech/zktrie/trie"
)

// TODO: remove this hack
var magicHash []byte

func init() {
	hasher := newHasher(false)
	defer returnHasherToPool(hasher)
	magicHash = hasher.hashData(zktrie.ProofMagicBytes())
}
