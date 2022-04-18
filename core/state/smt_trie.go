// +build !oldTree

package state

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/trie"
)

// CopyTrie returns an independent copy of the given trie.
func (db *cachingDB) CopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case *trie.SecureBinaryTrie:
		return t.Copy()
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}
