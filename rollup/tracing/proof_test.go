package tracing

import (
	"bytes"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
	"github.com/scroll-tech/go-ethereum/trie"
	"github.com/stretchr/testify/assert"
)

func newTestingMerkle(t *testing.T) (*trie.ZkTrie, *trie.Database) {
	db := trie.NewDatabase(rawdb.NewMemoryDatabase(), &trie.Config{})
	return newTestingMerkleWithDb(t, common.Hash{}, db)
}

func newTestingMerkleWithDb(t *testing.T, root common.Hash, db *trie.Database) (*trie.ZkTrie, *trie.Database) {
	maxLevels := trie.NodeKeyValidBytes * 8
	mt, err := trie.NewZkTrie(trie.TrieID(root), db)
	if err != nil {
		t.Fatal(err)
		return nil, nil
	}
	mt.Debug = true
	assert.Equal(t, maxLevels, mt.MaxLevels())
	return mt, db
}

// Tests that new "proof trace" feature
func TestProofWithDeletion(t *testing.T) {
	mt, _ := newTestingMerkle(t)
	key1 := bytes.Repeat([]byte("b"), 32)
	key2 := bytes.Repeat([]byte("c"), 32)
	err := mt.TryUpdate(
		key1,
		1,
		[]trie.Byte32{*trie.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("v"), 32))},
	)
	assert.NoError(t, err)
	err = mt.TryUpdate(
		key2,
		1,
		[]trie.Byte32{*trie.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("n"), 32))},
	)
	assert.NoError(t, err)

	proof := memorydb.New()
	proofTracer := NewProofTracer(mt)

	err = proofTracer.Prove(key1, proof)
	assert.NoError(t, err)
	nd, err := mt.TryGet(key2)
	assert.NoError(t, err)

	key4 := bytes.Repeat([]byte("x"), 32)
	err = proofTracer.Prove(key4, proof)
	assert.NoError(t, err)
	//assert.Equal(t, len(sibling1), len(delTracer.GetProofs()))

	siblings, err := proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(siblings))

	proofTracer.MarkDeletion(key1)
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(siblings))
	l := len(siblings[0])
	// a hacking to grep the value part directly from the encoded leaf node,
	// notice the sibling of key `k*32`` is just the leaf of key `m*32`
	assert.Equal(t, siblings[0][l-33:l-1], nd)

	// Marking a key that is currently not hit (but terminated by an empty node)
	// also causes it to be added to the deletion proof
	proofTracer.MarkDeletion(key4)
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(siblings))

	key3 := bytes.Repeat([]byte("x"), 32)
	err = mt.TryUpdate(
		key3,
		1,
		[]trie.Byte32{*trie.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("z"), 32))},
	)
	assert.NoError(t, err)

	proofTracer = NewProofTracer(mt)
	err = proofTracer.Prove(key1, proof)
	assert.NoError(t, err)
	err = proofTracer.Prove(key4, proof)
	assert.NoError(t, err)

	proofTracer.MarkDeletion(key1)
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(siblings))

	proofTracer.MarkDeletion(key4)
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(siblings))

	// one of the siblings is just leaf for key2, while
	// another one must be a middle node
	match1 := bytes.Equal(siblings[0][l-33:l-1], nd)
	match2 := bytes.Equal(siblings[1][l-33:l-1], nd)
	assert.True(t, match1 || match2)
	assert.False(t, match1 && match2)
}
