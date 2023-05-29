package zktrie

import (
	itrie "github.com/scroll-tech/zktrie/trie"
	itypes "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/poseidon"
)

func init() {
	itypes.InitHashScheme(poseidon.HashFixed)
}

func reverseBytesInPlace(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

func StoreHashFromNodeHash(node common.Hash) *itypes.Hash {
	h := new(itypes.Hash)
	copy(h[:], node[:])
	reverseBytesInPlace(h[:])
	return h
}

func NodeHashFromStoreKey(key []byte) common.Hash {
	h := common.Hash{}
	copy(h[:], key)
	reverseBytesInPlace(h[:])
	return h
}

// NodeStoreHash represent the db key of node content for storing
func NodeStoreHash(blob []byte) (*itypes.Hash, error) {
	node, err := itrie.NewNodeFromBytes(blob)
	if err != nil {
		return nil, err
	}

	hash, err := node.NodeHash()
	if err != nil {
		return nil, err
	}

	return hash, nil
}

// NodeHash represent the hash of node content
func NodeHash(blob []byte) (common.Hash, error) {
	hash, err := NodeStoreHash(blob)
	if err != nil {
		return common.Hash{}, err
	}

	var h common.Hash
	copy(h[:], hash[:])
	reverseBytesInPlace(h[:])
	return h, nil
}
