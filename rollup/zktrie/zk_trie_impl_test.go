package zktrie

import (
	"bytes"
	"math/big"
	"os"
	"runtime"
	"sync"
	"testing"

	"github.com/iden3/go-iden3-crypto/constants"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/trie"
	"github.com/scroll-tech/go-ethereum/trie/trienode"
	"github.com/stretchr/testify/assert"
)

func setupENV() {
	InitHashScheme(func(arr []*big.Int, domain *big.Int) (*big.Int, error) {
		lcEff := big.NewInt(65536)
		sum := domain
		for _, bi := range arr {
			nbi := new(big.Int).Mul(bi, bi)
			sum = sum.Mul(sum, sum)
			sum = sum.Mul(sum, lcEff)
			sum = sum.Add(sum, nbi)
		}
		return sum.Mod(sum, Q), nil
	})
}

func TestMain(m *testing.M) {
	InitHashScheme(func(arr []*big.Int, domain *big.Int) (*big.Int, error) {
		lcEff := big.NewInt(65536)
		sum := domain
		for _, bi := range arr {
			nbi := new(big.Int).Mul(bi, bi)
			sum = sum.Mul(sum, sum)
			sum = sum.Mul(sum, lcEff)
			sum = sum.Add(sum, nbi)
		}
		return sum.Mod(sum, Q), nil
	})
	os.Exit(m.Run())
}

func newTestingMerkle(t *testing.T) (*ZkTrie, *trie.Database) {
	db := trie.NewDatabase(rawdb.NewMemoryDatabase(), &trie.Config{
		ChildResolver: ChildResolver{},
	})
	return newTestingMerkleWithDb(t, common.Hash{}, db)
}

func newTestingMerkleWithDb(t *testing.T, root common.Hash, db *trie.Database) (*ZkTrie, *trie.Database) {
	maxLevels := NodeKeyValidBytes * 8
	mt, err := NewZkTrie(trie.TrieID(root), db)
	if err != nil {
		t.Fatal(err)
		return nil, nil
	}
	mt.Debug = true
	assert.Equal(t, maxLevels, mt.MaxLevels())
	return mt, db
}

func TestMerkleTree_Init(t *testing.T) {
	maxLevels := 248
	t.Run("Test NewZkTrieImpl", func(t *testing.T) {
		mt, _ := newTestingMerkle(t)
		mtRoot, err := mt.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero.Bytes(), mtRoot.Bytes())
	})

	t.Run("Test NewZkTrieImplWithRoot with zero hash root", func(t *testing.T) {
		mt, _ := newTestingMerkle(t)
		mtRoot, err := mt.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero.Bytes(), mtRoot.Bytes())
	})

	t.Run("Test NewZkTrieImplWithRoot with non-zero hash root and node exists", func(t *testing.T) {
		mt1, db := newTestingMerkle(t)
		mt1Root, err := mt1.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero.Bytes(), mt1Root.Bytes())
		err = mt1.TryUpdate([]byte{1}, 1, []Byte32{{byte(1)}})
		assert.NoError(t, err)
		mt1Root, err = mt1.Root()
		assert.NoError(t, err)
		assert.Equal(t, "2bbb5391bce512d6d0e02e2162bf7f0eb8ec6df806f9284ec5c3242193409553", mt1Root.Hex())
		rootHash, nodeSet, err := mt1.Commit(false)
		assert.NoError(t, err)
		assert.NoError(t, db.Update(rootHash, common.Hash{}, 0, trienode.NewWithNodeSet(nodeSet), nil))
		assert.NoError(t, db.Commit(rootHash, false))

		mt2, _ := newTestingMerkleWithDb(t, rootHash, db)
		assert.Equal(t, maxLevels, mt2.maxLevels)
		mt2Root, err := mt2.Root()
		assert.NoError(t, err)
		assert.Equal(t, "2bbb5391bce512d6d0e02e2162bf7f0eb8ec6df806f9284ec5c3242193409553", mt2Root.Hex())
	})
}

func TestMerkleTree_AddUpdateGetWord(t *testing.T) {
	mt, _ := newTestingMerkle(t)

	testData := []struct {
		key        byte
		initialVal byte
		updatedVal byte
	}{
		{1, 2, 7},
		{3, 4, 8},
		{5, 6, 9},
	}

	for _, td := range testData {
		err := mt.TryUpdate([]byte{td.key}, 1, []Byte32{{td.initialVal}})
		assert.NoError(t, err)

		node, err := mt.GetLeafNode([]byte{td.key})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(node.ValuePreimage))
		assert.Equal(t, (&Byte32{td.initialVal})[:], node.ValuePreimage[0][:])
	}

	for _, td := range testData {
		err := mt.TryUpdate([]byte{td.key}, 1, []Byte32{{td.updatedVal}})
		assert.NoError(t, err)

		node, err := mt.GetLeafNode([]byte{td.key})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(node.ValuePreimage))
		assert.Equal(t, (&Byte32{td.updatedVal})[:], node.ValuePreimage[0][:])
	}

	_, err := mt.GetLeafNode([]byte{100})
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestMerkleTree_Deletion(t *testing.T) {
	t.Run("Check root consistency", func(t *testing.T) {
		var err error
		mt, _ := newTestingMerkle(t)
		hashes := make([]*Hash, 7)
		hashes[0], err = mt.Root()
		assert.NoError(t, err)

		for i := 0; i < 6; i++ {
			err := mt.TryUpdate([]byte{byte(i)}, 1, []Byte32{{byte(i)}})
			assert.NoError(t, err)
			hashes[i+1], err = mt.Root()
			assert.NoError(t, err)
		}

		for i := 5; i >= 0; i-- {
			err := mt.TryDelete([]byte{byte(i)})
			assert.NoError(t, err)
			root, err := mt.Root()
			assert.NoError(t, err)
			assert.Equal(t, hashes[i], root, i)
		}
	})
}

func TestZkTrieImpl_Add(t *testing.T) {
	k1 := NewByte32FromBytes([]byte{1})
	k2 := NewByte32FromBytes([]byte{2})
	k3 := NewByte32FromBytes([]byte{3})

	kvMap := map[*Byte32]*Byte32{
		k1: NewByte32FromBytes([]byte{1}),
		k2: NewByte32FromBytes([]byte{2}),
		k3: NewByte32FromBytes([]byte{3}),
	}

	t.Run("Add 1 and 2 in different orders", func(t *testing.T) {
		orders := [][]*Byte32{
			{k1, k2},
			{k2, k1},
		}

		roots := make([]*Hash, len(orders))
		for i, order := range orders {
			mt, _ := newTestingMerkle(t)
			for _, key := range order {
				value := kvMap[key]
				err := mt.TryUpdate(key.Bytes(), 1, []Byte32{*value})
				assert.NoError(t, err)
			}
			var err error
			roots[i], err = mt.Root()
			assert.NoError(t, err)
		}

		assert.Equal(t, "23f0807c95a8a6be167ca512f850b0b9f7349b033ae0be8e7caf0553c13eee16", roots[0].Hex())
		assert.Equal(t, roots[0], roots[1])
	})

	t.Run("Add 1, 2, 3 in different orders", func(t *testing.T) {
		orders := [][]*Byte32{
			{k1, k2, k3},
			{k1, k3, k2},
			{k2, k1, k3},
			{k2, k3, k1},
			{k3, k1, k2},
			{k3, k2, k1},
		}

		roots := make([]*Hash, len(orders))
		for i, order := range orders {
			mt, _ := newTestingMerkle(t)
			for _, key := range order {
				value := kvMap[key]
				err := mt.TryUpdate(key.Bytes(), 1, []Byte32{*value})
				assert.NoError(t, err)
			}
			var err error
			roots[i], err = mt.Root()
			assert.NoError(t, err)
		}

		for i := 1; i < len(roots); i++ {
			assert.Equal(t, "17927c39184cb91ef9b105e42c8cdda845daf7f936309f665c7cc1beabbec191", roots[0].Hex())
			assert.Equal(t, roots[0], roots[i])
		}
	})
}

func TestZkTrieImpl_Update(t *testing.T) {
	k1 := []byte{1}
	k2 := []byte{2}
	k3 := []byte{3}

	t.Run("Update 1", func(t *testing.T) {
		mt1, _ := newTestingMerkle(t)
		err := mt1.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		root1, err := mt1.Root()
		assert.NoError(t, err)

		mt2, _ := newTestingMerkle(t)
		err = mt2.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{2})})
		assert.NoError(t, err)
		err = mt2.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		root2, err := mt2.Root()
		assert.NoError(t, err)

		assert.Equal(t, root1, root2)
	})

	t.Run("Update 2", func(t *testing.T) {
		mt1, _ := newTestingMerkle(t)
		err := mt1.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		err = mt1.TryUpdate(k2, 1, []Byte32{*NewByte32FromBytes([]byte{2})})
		assert.NoError(t, err)
		root1, err := mt1.Root()
		assert.NoError(t, err)

		mt2, _ := newTestingMerkle(t)
		err = mt2.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		err = mt2.TryUpdate(k2, 1, []Byte32{*NewByte32FromBytes([]byte{3})})
		assert.NoError(t, err)
		err = mt2.TryUpdate(k2, 1, []Byte32{*NewByte32FromBytes([]byte{2})})
		assert.NoError(t, err)
		root2, err := mt2.Root()
		assert.NoError(t, err)

		assert.Equal(t, root1, root2)
	})

	t.Run("Update 1, 2, 3", func(t *testing.T) {
		mt1, _ := newTestingMerkle(t)
		mt2, _ := newTestingMerkle(t)
		keys := [][]byte{k1, k2, k3}
		for i, key := range keys {
			err := mt1.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{byte(i)})})
			assert.NoError(t, err)
		}
		for i, key := range keys {
			err := mt2.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{byte(i + 3)})})
			assert.NoError(t, err)
		}
		for i, key := range keys {
			err := mt1.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{byte(i + 6)})})
			assert.NoError(t, err)
			err = mt2.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{byte(i + 6)})})
			assert.NoError(t, err)
		}

		root1, err := mt1.Root()
		assert.NoError(t, err)
		root2, err := mt2.Root()
		assert.NoError(t, err)

		assert.Equal(t, root1, root2)
	})

	t.Run("Update same value", func(t *testing.T) {
		mt, _ := newTestingMerkle(t)
		keys := [][]byte{k1, k2, k3}
		for _, key := range keys {
			err := mt.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
			assert.NoError(t, err)
			err = mt.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
			assert.NoError(t, err)
			node, err := mt.GetLeafNode(key)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(node.ValuePreimage))
			assert.Equal(t, NewByte32FromBytes([]byte{1}).Bytes(), node.ValuePreimage[0][:])
		}
	})

	t.Run("Update non-existent word", func(t *testing.T) {
		mt, _ := newTestingMerkle(t)
		err := mt.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		node, err := mt.GetLeafNode(k1)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(node.ValuePreimage))
		assert.Equal(t, NewByte32FromBytes([]byte{1}).Bytes(), node.ValuePreimage[0][:])
	})
}

func TestZkTrieImpl_Delete(t *testing.T) {
	k1 := []byte{1}
	k2 := []byte{2}
	k3 := []byte{3}
	k4 := []byte{4}

	t.Run("Test deletion leads to empty tree", func(t *testing.T) {
		emptyMT, _ := newTestingMerkle(t)
		emptyMTRoot, err := emptyMT.Root()
		assert.NoError(t, err)

		mt1, _ := newTestingMerkle(t)
		err = mt1.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		err = mt1.TryDelete(k1)
		assert.NoError(t, err)
		mt1Root, err := mt1.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero, *mt1Root)
		assert.Equal(t, emptyMTRoot, mt1Root)

		keys := [][]byte{k1, k2, k3, k4}
		mt2, _ := newTestingMerkle(t)
		for _, key := range keys {
			err := mt2.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
			assert.NoError(t, err)
		}
		for _, key := range keys {
			err := mt2.TryDelete(key)
			assert.NoError(t, err)
		}
		mt2Root, err := mt2.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero, *mt2Root)
		assert.Equal(t, emptyMTRoot, mt2Root)

		mt3, _ := newTestingMerkle(t)
		for _, key := range keys {
			err := mt3.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
			assert.NoError(t, err)
		}
		for i := len(keys) - 1; i >= 0; i-- {
			err := mt3.TryDelete(keys[i])
			assert.NoError(t, err)
		}
		mt3Root, err := mt3.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero, *mt3Root)
		assert.Equal(t, emptyMTRoot, mt3Root)
	})

	t.Run("Test equivalent trees after deletion", func(t *testing.T) {
		keys := [][]byte{k1, k2, k3, k4}

		mt1, _ := newTestingMerkle(t)
		for i, key := range keys {
			err := mt1.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{byte(i + 1)})})
			assert.NoError(t, err)
		}
		err := mt1.TryDelete(k1)
		assert.NoError(t, err)
		err = mt1.TryDelete(k2)
		assert.NoError(t, err)

		mt2, _ := newTestingMerkle(t)
		err = mt2.TryUpdate(k3, 1, []Byte32{*NewByte32FromBytes([]byte{byte(3)})})
		assert.NoError(t, err)
		err = mt2.TryUpdate(k4, 1, []Byte32{*NewByte32FromBytes([]byte{byte(4)})})
		assert.NoError(t, err)

		mt1Root, err := mt1.Root()
		assert.NoError(t, err)
		mt2Root, err := mt2.Root()
		assert.NoError(t, err)

		assert.Equal(t, mt1Root, mt2Root)

		mt3, _ := newTestingMerkle(t)
		for i, key := range keys {
			err := mt3.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{byte(i + 1)})})
			assert.NoError(t, err)
		}
		err = mt3.TryDelete(k1)
		assert.NoError(t, err)
		err = mt3.TryDelete(k3)
		assert.NoError(t, err)
		mt4, _ := newTestingMerkle(t)
		err = mt4.TryUpdate(k2, 1, []Byte32{*NewByte32FromBytes([]byte{2})})
		assert.NoError(t, err)
		err = mt4.TryUpdate(k4, 1, []Byte32{*NewByte32FromBytes([]byte{4})})
		assert.NoError(t, err)

		mt3Root, err := mt3.Root()
		assert.NoError(t, err)
		mt4Root, err := mt4.Root()
		assert.NoError(t, err)

		assert.Equal(t, mt3Root, mt4Root)
	})

	t.Run("Test repeat deletion", func(t *testing.T) {
		mt, _ := newTestingMerkle(t)
		err := mt.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		err = mt.TryDelete(k1)
		assert.NoError(t, err)
		err = mt.TryDelete(k1)
		assert.Equal(t, ErrKeyNotFound, err)
	})

	t.Run("Test deletion of non-existent node", func(t *testing.T) {
		mt, _ := newTestingMerkle(t)
		err := mt.TryDelete(k1)
		assert.Equal(t, ErrKeyNotFound, err)
	})
}

func TestMerkleTree_BuildAndVerifyZkTrieProof(t *testing.T) {
	zkTrie, _ := newTestingMerkle(t)

	testData := []struct {
		key   *big.Int
		value byte
	}{
		{big.NewInt(1), 2},
		{big.NewInt(3), 4},
		{big.NewInt(5), 6},
		{big.NewInt(7), 8},
		{big.NewInt(9), 10},
	}

	nonExistentKey := big.NewInt(11)

	for _, td := range testData {
		err := zkTrie.TryUpdate([]byte{byte(td.key.Int64())}, 1, []Byte32{{td.value}})
		assert.NoError(t, err)
	}
	_, err := zkTrie.Root()
	assert.NoError(t, err)

	t.Run("Test with existent key", func(t *testing.T) {
		for _, td := range testData {

			node, err := zkTrie.GetLeafNode([]byte{byte(td.key.Int64())})
			assert.NoError(t, err)
			assert.Equal(t, 1, len(node.ValuePreimage))
			assert.Equal(t, (&Byte32{td.value})[:], node.ValuePreimage[0][:])
			proof, node, err := BuildZkTrieProof(zkTrie.rootKey, td.key, 10, zkTrie.GetNode)
			assert.NoError(t, err)

			valid := VerifyProofZkTrie(zkTrie.rootKey, proof, node)
			assert.True(t, valid)
		}
	})

	t.Run("Test with non-existent key", func(t *testing.T) {
		proof, node, err := BuildZkTrieProof(zkTrie.rootKey, nonExistentKey, 10, zkTrie.GetNode)
		assert.NoError(t, err)
		assert.False(t, proof.Existence)
		valid := VerifyProofZkTrie(zkTrie.rootKey, proof, node)
		assert.True(t, valid)
		nodeAnother, err := zkTrie.GetLeafNode([]byte{byte(big.NewInt(1).Int64())})
		assert.NoError(t, err)
		valid = VerifyProofZkTrie(zkTrie.rootKey, proof, nodeAnother)
		assert.False(t, valid)

		hash, err := proof.Verify(node.nodeHash)
		assert.NoError(t, err)
		assert.Equal(t, hash[:], zkTrie.rootKey[:])
	})
}

func TestMerkleTree_GraphViz(t *testing.T) {
	mt, _ := newTestingMerkle(t)

	var buffer bytes.Buffer
	err := mt.GraphViz(&buffer, nil)
	assert.NoError(t, err)
	assert.Equal(t, "--------\nGraphViz of the ZkTrie with RootHash 0\ndigraph hierarchy {\nnode [fontname=Monospace,fontsize=10,shape=box]\n}\nEnd of GraphViz of the ZkTrie with RootHash 0\n--------\n", buffer.String())
	buffer.Reset()

	key1 := []byte{1} //0b1
	err = mt.TryUpdate(key1, 1, []Byte32{{1}})
	assert.NoError(t, err)
	key2 := []byte{3} //0b11
	err = mt.TryUpdate(key2, 1, []Byte32{{3}})
	assert.NoError(t, err)

	err = mt.GraphViz(&buffer, nil)
	assert.NoError(t, err)
	assert.Equal(t, "--------\nGraphViz of the ZkTrie with RootHash 10951270817330706114198641949214391028137561893123097337637233896895686724291\ndigraph hierarchy {\nnode [fontname=Monospace,fontsize=10,shape=box]\n\"10951270...\" -> {\"16038355...\" \"19780429...\"}\n\"16038355...\" [style=filled];\n\"19780429...\" [style=filled];\n}\nEnd of GraphViz of the ZkTrie with RootHash 10951270817330706114198641949214391028137561893123097337637233896895686724291\n--------\n", buffer.String())
	buffer.Reset()
}

func TestZkTrie_GetUpdateDelete(t *testing.T) {
	mt, _ := newTestingMerkle(t)
	val, err := mt.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Nil(t, val)
	assert.Equal(t, common.Hash{}, mt.Hash())

	err = mt.TryUpdate([]byte("key"), 1, []Byte32{{1}})
	assert.NoError(t, err)
	expected := common.BytesToHash([]byte{0x23, 0x36, 0x5e, 0xbd, 0x71, 0xa7, 0xad, 0x35, 0x65, 0xdd, 0x24, 0x88, 0x47, 0xca, 0xe8, 0xe8, 0x8, 0x21, 0x15, 0x62, 0xc6, 0x83, 0xdb, 0x8, 0x4f, 0x5a, 0xfb, 0xd1, 0xb0, 0x3d, 0x4c, 0xb5})
	assert.Equal(t, expected, mt.Hash())

	val, err = mt.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, (&Byte32{1}).Bytes(), val)

	err = mt.TryDelete([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, common.Hash{}, mt.Hash())

	val, err = mt.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Nil(t, val)
}

func TestZkTrie_Copy(t *testing.T) {
	mt, _ := newTestingMerkle(t)

	mt.TryUpdate([]byte("key"), 1, []Byte32{{1}})

	copyTrie := mt.Copy()
	val, err := copyTrie.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, (&Byte32{1}).Bytes(), val)
}

func TestZkTrie_ProveAndProveWithDeletion(t *testing.T) {
	mt, _ := newTestingMerkle(t)

	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	for i, keyStr := range keys {
		key := make([]byte, 32)
		copy(key, []byte(keyStr))

		err := mt.TryUpdate(key, uint32(i+1), []Byte32{{byte(uint32(i + 1))}})
		assert.NoError(t, err)

		writeNode := func(n *Node) error {
			return nil
		}

		k, err := ToSecureKey(key)
		assert.NoError(t, err)

		for j := 0; j <= i; j++ {
			err = mt.ProveWithDeletion(NewHashFromBigInt(k).Bytes(), uint(j), writeNode, nil)
			assert.NoError(t, err)
		}
	}
}

func newHashFromHex(h string) (*Hash, error) {
	return NewHashFromCheckedBytes(common.FromHex(h))
}

func TestHashParsers(t *testing.T) {
	h0 := NewHashFromBigInt(big.NewInt(0))
	assert.Equal(t, "0", h0.String())
	h1 := NewHashFromBigInt(big.NewInt(1))
	assert.Equal(t, "1", h1.String())
	h10 := NewHashFromBigInt(big.NewInt(10))
	assert.Equal(t, "10", h10.String())

	h7l := NewHashFromBigInt(big.NewInt(1234567))
	assert.Equal(t, "1234567", h7l.String())
	h8l := NewHashFromBigInt(big.NewInt(12345678))
	assert.Equal(t, "12345678...", h8l.String())

	b, ok := new(big.Int).SetString("4932297968297298434239270129193057052722409868268166443802652458940273154854", 10) //nolint:lll
	assert.True(t, ok)
	h := NewHashFromBigInt(b)
	assert.Equal(t, "4932297968297298434239270129193057052722409868268166443802652458940273154854", h.BigInt().String()) //nolint:lll
	assert.Equal(t, "49322979...", h.String())
	assert.Equal(t, "0ae794eb9c3d8bbb9002e993fc2ed301dcbd2af5508ed072c375e861f1aa5b26", h.Hex())

	b1, err := NewBigIntFromHashBytes(b.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, new(big.Int).SetBytes(b.Bytes()).String(), b1.String())

	b2, err := NewHashFromCheckedBytes(b.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, b.String(), b2.BigInt().String())

	h2, err := newHashFromHex(h.Hex())
	assert.Nil(t, err)
	assert.Equal(t, h, h2)
	_, err = newHashFromHex("0x12")
	assert.NotNil(t, err)

	// check limits
	a := new(big.Int).Sub(constants.Q, big.NewInt(1))
	testHashParsers(t, a)
	a = big.NewInt(int64(1))
	testHashParsers(t, a)
}

func testHashParsers(t *testing.T, a *big.Int) {
	h := NewHashFromBigInt(a)
	assert.Equal(t, a, h.BigInt())
	hFromBytes, err := NewHashFromCheckedBytes(h.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, h, hFromBytes)
	assert.Equal(t, a, hFromBytes.BigInt())
	assert.Equal(t, a.String(), hFromBytes.BigInt().String())
	hFromHex, err := newHashFromHex(h.Hex())
	assert.Nil(t, err)
	assert.Equal(t, h, hFromHex)

	aBIFromHBytes, err := NewBigIntFromHashBytes(h.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, a, aBIFromHBytes)
	assert.Equal(t, new(big.Int).SetBytes(a.Bytes()).String(), aBIFromHBytes.String())
}

func TestMerkleTree_AddUpdateGetWord_2(t *testing.T) {
	mt, _ := newTestingMerkle(t)
	err := mt.TryUpdate([]byte{1}, 1, []Byte32{{2}})
	assert.Nil(t, err)
	err = mt.TryUpdate([]byte{3}, 1, []Byte32{{4}})
	assert.Nil(t, err)
	err = mt.TryUpdate([]byte{5}, 1, []Byte32{{6}})
	assert.Nil(t, err)

	mt.GetLeafNode([]byte{1})
	node, err := mt.GetLeafNode([]byte{1})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&Byte32{2})[:], node.ValuePreimage[0][:])
	node, err = mt.GetLeafNode([]byte{3})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&Byte32{4})[:], node.ValuePreimage[0][:])
	node, err = mt.GetLeafNode([]byte{5})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&Byte32{6})[:], node.ValuePreimage[0][:])

	err = mt.TryUpdate([]byte{1}, 1, []Byte32{{7}})
	assert.Nil(t, err)
	err = mt.TryUpdate([]byte{3}, 1, []Byte32{{8}})
	assert.Nil(t, err)
	err = mt.TryUpdate([]byte{5}, 1, []Byte32{{9}})
	assert.Nil(t, err)

	node, err = mt.GetLeafNode([]byte{1})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&Byte32{7})[:], node.ValuePreimage[0][:])
	node, err = mt.GetLeafNode([]byte{3})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&Byte32{8})[:], node.ValuePreimage[0][:])
	node, err = mt.GetLeafNode([]byte{5})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&Byte32{9})[:], node.ValuePreimage[0][:])
	_, err = mt.GetLeafNode([]byte{100})
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestMerkleTree_UpdateAccount(t *testing.T) {
	mt, _ := newTestingMerkle(t)

	acc1 := &types.StateAccount{
		Nonce:            1,
		Balance:          big.NewInt(10000000),
		Root:             common.HexToHash("22fb59aa5410ed465267023713ab42554c250f394901455a3366e223d5f7d147"),
		KeccakCodeHash:   common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
		PoseidonCodeHash: common.HexToHash("0c0a77f6e063b4b62eb7d9ed6f427cf687d8d0071d751850cfe5d136bc60d3ab").Bytes(),
		CodeSize:         0,
	}
	value, flag := acc1.MarshalFields()
	accValue := []Byte32{}
	for _, v := range value {
		accValue = append(accValue, *NewByte32FromBytes(v.Bytes()))
	}
	err := mt.TryUpdate(common.HexToAddress("0x05fDbDfaE180345C6Cff5316c286727CF1a43327").Bytes(), flag, accValue)
	assert.Nil(t, err)

	acc2 := &types.StateAccount{
		Nonce:            5,
		Balance:          big.NewInt(50000000),
		Root:             common.HexToHash("0"),
		KeccakCodeHash:   common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
		PoseidonCodeHash: common.HexToHash("05d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
		CodeSize:         5,
	}
	value, flag = acc2.MarshalFields()
	accValue = []Byte32{}
	for _, v := range value {
		accValue = append(accValue, *NewByte32FromBytes(v.Bytes()))
	}
	err = mt.TryUpdate(common.HexToAddress("0x4cb1aB63aF5D8931Ce09673EbD8ae2ce16fD6571").Bytes(), flag, accValue)
	assert.Nil(t, err)

	bt, err := mt.TryGet(common.HexToAddress("0x05fDbDfaE180345C6Cff5316c286727CF1a43327").Bytes())
	assert.Nil(t, err)

	acc, err := types.UnmarshalStateAccount(bt)
	assert.Nil(t, err)
	assert.Equal(t, acc1.Nonce, acc.Nonce)
	assert.Equal(t, acc1.Balance.Uint64(), acc.Balance.Uint64())
	assert.Equal(t, acc1.Root.Bytes(), acc.Root.Bytes())
	assert.Equal(t, acc1.KeccakCodeHash, acc.KeccakCodeHash)
	assert.Equal(t, acc1.PoseidonCodeHash, acc.PoseidonCodeHash)
	assert.Equal(t, acc1.CodeSize, acc.CodeSize)

	bt, err = mt.TryGet(common.HexToAddress("0x4cb1aB63aF5D8931Ce09673EbD8ae2ce16fD6571").Bytes())
	assert.Nil(t, err)

	acc, err = types.UnmarshalStateAccount(bt)
	assert.Nil(t, err)
	assert.Equal(t, acc2.Nonce, acc.Nonce)
	assert.Equal(t, acc2.Balance.Uint64(), acc.Balance.Uint64())
	assert.Equal(t, acc2.Root.Bytes(), acc.Root.Bytes())
	assert.Equal(t, acc2.KeccakCodeHash, acc.KeccakCodeHash)
	assert.Equal(t, acc2.PoseidonCodeHash, acc.PoseidonCodeHash)
	assert.Equal(t, acc2.CodeSize, acc.CodeSize)

	bt, err = mt.TryGet(common.HexToAddress("0x8dE13967F19410A7991D63c2c0179feBFDA0c261").Bytes())
	assert.Nil(t, err)
	assert.Nil(t, bt)

	err = mt.TryDelete(common.HexToAddress("0x05fDbDfaE180345C6Cff5316c286727CF1a43327").Bytes())
	assert.Nil(t, err)

	bt, err = mt.TryGet(common.HexToAddress("0x05fDbDfaE180345C6Cff5316c286727CF1a43327").Bytes())
	assert.Nil(t, err)
	assert.Nil(t, bt)

	err = mt.TryDelete(common.HexToAddress("0x4cb1aB63aF5D8931Ce09673EbD8ae2ce16fD6571").Bytes())
	assert.Nil(t, err)

	bt, err = mt.TryGet(common.HexToAddress("0x4cb1aB63aF5D8931Ce09673EbD8ae2ce16fD6571").Bytes())
	assert.Nil(t, err)
	assert.Nil(t, bt)
}

func TestDecodeSMTProof(t *testing.T) {
	node, err := DecodeSMTProof(magicSMTBytes)
	assert.NoError(t, err)
	assert.Nil(t, node)

	k1 := NewHashFromBytes([]byte{1, 2, 3, 4, 5})
	k2 := NewHashFromBytes([]byte{6, 7, 8, 9, 0})
	origNode := NewParentNode(NodeTypeBranch_0, k1, k2)
	node, err = DecodeSMTProof(origNode.Value())
	assert.NoError(t, err)
	assert.Equal(t, origNode.Value(), node.Value())
}

func TestZktrieGetKey(t *testing.T) {
	t.Skip("get key is not implemented")
	trie, _ := newTestingMerkle(t)
	key := []byte("0a1b2c3d4e5f6g7h8i9j0a1b2c3d4e5f")
	value := []byte("9j8i7h6g5f4e3d2c1b0a9j8i7h6g5f4e")
	trie.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes(value)})

	kPreimage := NewByte32FromBytesPaddingZero(key)
	kHash, err := kPreimage.Hash()
	assert.Nil(t, err)
	if k := trie.GetKey(kHash.Bytes()); !bytes.Equal(k, key) {
		t.Errorf("GetKey returned %q, want %q", k, key)
	}
}

func TestZkTrieConcurrency(t *testing.T) {
	// Create an initial trie and copy if for concurrent access
	trie, _ := newTestingMerkle(t)

	threads := runtime.NumCPU()
	tries := make([]*ZkTrie, threads)
	for i := 0; i < threads; i++ {
		tries[i] = trie.Copy()
	}
	// Start a batch of goroutines interactng with the trie
	pend := new(sync.WaitGroup)
	pend.Add(threads)
	for i := 0; i < threads; i++ {
		go func(index int) {
			defer pend.Done()

			for j := byte(0); j < 255; j++ {
				// Map the same data under multiple keys
				key, val := common.LeftPadBytes([]byte{byte(index), 1, j}, 32), bytes.Repeat([]byte{j}, 32)
				tries[index].TryUpdate(key, 1, []Byte32{*NewByte32FromBytes(val)})

				key, val = common.LeftPadBytes([]byte{byte(index), 2, j}, 32), bytes.Repeat([]byte{j}, 32)
				tries[index].TryUpdate(key, 1, []Byte32{*NewByte32FromBytes(val)})

				// Add some other data to inflate the trie
				for k := byte(3); k < 13; k++ {
					key, val = common.LeftPadBytes([]byte{byte(index), k, j}, 32), bytes.Repeat([]byte{k, j}, 16)
					tries[index].TryUpdate(key, 1, []Byte32{*NewByte32FromBytes(val)})
				}
			}
			tries[index].Commit(false)
		}(i)
	}
	// Wait for all threads to finish
	pend.Wait()
}

func TestZkTrieDelete(t *testing.T) {
	trie1, _ := newTestingMerkle(t)

	var count int = 6
	var hashes []common.Hash
	hashes = append(hashes, trie1.Hash())
	for i := 0; i < count; i++ {
		err := trie1.TryUpdate([]byte{byte(i)}, 1, []Byte32{{byte(i)}})
		assert.NoError(t, err)
		hashes = append(hashes, trie1.Hash())
	}

	for i := count - 1; i >= 0; i-- {
		v, err := trie1.TryGet([]byte{byte(i)})
		assert.NoError(t, err)
		assert.NotEmpty(t, v)
		err = trie1.TryDelete([]byte{byte(i)})
		assert.NoError(t, err)
		hash := trie1.Hash()
		assert.Equal(t, hashes[i].Hex(), hash.Hex())
	}
}
