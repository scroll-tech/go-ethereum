package zktrie

import (
	"bytes"
	"math/big"
	"os"
	"testing"

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

func newTestingMerkle(t *testing.T) *ZkTrieImpl {
	maxLevels := NodeKeyValidBytes * 8
	mt, err := NewZkTrieImplWithRoot(NewZkTrieMemoryDb(), &HashZero, maxLevels)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	mt.Debug = true
	assert.Equal(t, maxLevels, mt.MaxLevels())
	return mt
}

func TestMerkleTree_Init(t *testing.T) {
	maxLevels := 248
	db := NewZkTrieMemoryDb()

	t.Run("Test NewZkTrieImpl", func(t *testing.T) {
		mt, err := NewZkTrieImpl(db, maxLevels)
		assert.NoError(t, err)
		mtRoot, err := mt.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero.Bytes(), mtRoot.Bytes())
	})

	t.Run("Test NewZkTrieImplWithRoot with zero hash root", func(t *testing.T) {
		mt, err := NewZkTrieImplWithRoot(db, &HashZero, maxLevels)
		assert.NoError(t, err)
		mtRoot, err := mt.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero.Bytes(), mtRoot.Bytes())
	})

	t.Run("Test NewZkTrieImplWithRoot with non-zero hash root and node exists", func(t *testing.T) {
		mt1, err := NewZkTrieImplWithRoot(db, &HashZero, maxLevels)
		assert.NoError(t, err)
		mt1Root, err := mt1.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero.Bytes(), mt1Root.Bytes())
		err = mt1.TryUpdate([]byte{1}, 1, []Byte32{{byte(1)}})
		assert.NoError(t, err)
		mt1Root, err = mt1.Root()
		assert.NoError(t, err)
		assert.Equal(t, "2bbb5391bce512d6d0e02e2162bf7f0eb8ec6df806f9284ec5c3242193409553", mt1Root.Hex())
		assert.NoError(t, mt1.Commit())

		mt2, err := NewZkTrieImplWithRoot(db, mt1Root, maxLevels)
		assert.NoError(t, err)
		assert.Equal(t, maxLevels, mt2.maxLevels)
		mt2Root, err := mt2.Root()
		assert.NoError(t, err)
		assert.Equal(t, "2bbb5391bce512d6d0e02e2162bf7f0eb8ec6df806f9284ec5c3242193409553", mt2Root.Hex())
	})

	t.Run("Test NewZkTrieImplWithRoot with non-zero hash root and node does not exist", func(t *testing.T) {
		root := NewHashFromBytes([]byte{1, 2, 3, 4, 5})

		mt, err := NewZkTrieImplWithRoot(db, root, maxLevels)
		assert.Error(t, err)
		assert.Nil(t, mt)
	})
}

func TestMerkleTree_AddUpdateGetWord(t *testing.T) {
	mt := newTestingMerkle(t)

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
		mt := newTestingMerkle(t)
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
			mt := newTestingMerkle(t)
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
			mt := newTestingMerkle(t)
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
		mt1 := newTestingMerkle(t)
		err := mt1.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		root1, err := mt1.Root()
		assert.NoError(t, err)

		mt2 := newTestingMerkle(t)
		err = mt2.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{2})})
		assert.NoError(t, err)
		err = mt2.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		root2, err := mt2.Root()
		assert.NoError(t, err)

		assert.Equal(t, root1, root2)
	})

	t.Run("Update 2", func(t *testing.T) {
		mt1 := newTestingMerkle(t)
		err := mt1.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		err = mt1.TryUpdate(k2, 1, []Byte32{*NewByte32FromBytes([]byte{2})})
		assert.NoError(t, err)
		root1, err := mt1.Root()
		assert.NoError(t, err)

		mt2 := newTestingMerkle(t)
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
		mt1 := newTestingMerkle(t)
		mt2 := newTestingMerkle(t)
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
		mt := newTestingMerkle(t)
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
		mt := newTestingMerkle(t)
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
		emptyMT := newTestingMerkle(t)
		emptyMTRoot, err := emptyMT.Root()
		assert.NoError(t, err)

		mt1 := newTestingMerkle(t)
		err = mt1.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		err = mt1.TryDelete(k1)
		assert.NoError(t, err)
		mt1Root, err := mt1.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero, *mt1Root)
		assert.Equal(t, emptyMTRoot, mt1Root)

		keys := [][]byte{k1, k2, k3, k4}
		mt2 := newTestingMerkle(t)
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

		mt3 := newTestingMerkle(t)
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

		mt1 := newTestingMerkle(t)
		for i, key := range keys {
			err := mt1.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{byte(i + 1)})})
			assert.NoError(t, err)
		}
		err := mt1.TryDelete(k1)
		assert.NoError(t, err)
		err = mt1.TryDelete(k2)
		assert.NoError(t, err)

		mt2 := newTestingMerkle(t)
		err = mt2.TryUpdate(k3, 1, []Byte32{*NewByte32FromBytes([]byte{byte(3)})})
		assert.NoError(t, err)
		err = mt2.TryUpdate(k4, 1, []Byte32{*NewByte32FromBytes([]byte{byte(4)})})
		assert.NoError(t, err)

		mt1Root, err := mt1.Root()
		assert.NoError(t, err)
		mt2Root, err := mt2.Root()
		assert.NoError(t, err)

		assert.Equal(t, mt1Root, mt2Root)

		mt3 := newTestingMerkle(t)
		for i, key := range keys {
			err := mt3.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes([]byte{byte(i + 1)})})
			assert.NoError(t, err)
		}
		err = mt3.TryDelete(k1)
		assert.NoError(t, err)
		err = mt3.TryDelete(k3)
		assert.NoError(t, err)
		mt4 := newTestingMerkle(t)
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
		mt := newTestingMerkle(t)
		err := mt.TryUpdate(k1, 1, []Byte32{*NewByte32FromBytes([]byte{1})})
		assert.NoError(t, err)
		err = mt.TryDelete(k1)
		assert.NoError(t, err)
		err = mt.TryDelete(k1)
		assert.Equal(t, ErrKeyNotFound, err)
	})

	t.Run("Test deletion of non-existent node", func(t *testing.T) {
		mt := newTestingMerkle(t)
		err := mt.TryDelete(k1)
		assert.Equal(t, ErrKeyNotFound, err)
	})
}

func TestMerkleTree_BuildAndVerifyZkTrieProof(t *testing.T) {
	zkTrie := newTestingMerkle(t)

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

	getNode := func(hash *Hash) (*Node, error) {
		node, err := zkTrie.GetNode(hash)
		if err != nil {
			return nil, err
		}
		return node, nil
	}

	for _, td := range testData {
		err := zkTrie.TryUpdate([]byte{byte(td.key.Int64())}, 1, []Byte32{{td.value}})
		assert.NoError(t, err)
	}

	t.Run("Test with existent key", func(t *testing.T) {
		for _, td := range testData {

			node, err := zkTrie.GetLeafNode([]byte{byte(td.key.Int64())})
			assert.NoError(t, err)
			assert.Equal(t, 1, len(node.ValuePreimage))
			assert.Equal(t, (&Byte32{td.value})[:], node.ValuePreimage[0][:])
			assert.NoError(t, zkTrie.Commit())

			proof, node, err := BuildZkTrieProof(zkTrie.rootKey, td.key, 10, getNode)
			assert.NoError(t, err)

			valid := VerifyProofZkTrie(zkTrie.rootKey, proof, node)
			assert.True(t, valid)
		}
	})

	t.Run("Test with non-existent key", func(t *testing.T) {
		proof, node, err := BuildZkTrieProof(zkTrie.rootKey, nonExistentKey, 10, getNode)
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
	mt := newTestingMerkle(t)

	var buffer bytes.Buffer
	err := mt.GraphViz(&buffer, nil)
	assert.NoError(t, err)
	assert.Equal(t, "--------\nGraphViz of the ZkTrieImpl with RootHash 0\ndigraph hierarchy {\nnode [fontname=Monospace,fontsize=10,shape=box]\n}\nEnd of GraphViz of the ZkTrieImpl with RootHash 0\n--------\n", buffer.String())
	buffer.Reset()

	key1 := []byte{1} //0b1
	err = mt.TryUpdate(key1, 1, []Byte32{{1}})
	assert.NoError(t, err)
	key2 := []byte{3} //0b11
	err = mt.TryUpdate(key2, 1, []Byte32{{3}})
	assert.NoError(t, err)

	err = mt.GraphViz(&buffer, nil)
	assert.NoError(t, err)
	assert.Equal(t, "--------\nGraphViz of the ZkTrieImpl with RootHash 10951270817330706114198641949214391028137561893123097337637233896895686724291\ndigraph hierarchy {\nnode [fontname=Monospace,fontsize=10,shape=box]\n\"10951270...\" -> {\"16038355...\" \"19780429...\"}\n\"16038355...\" [style=filled];\n\"19780429...\" [style=filled];\n}\nEnd of GraphViz of the ZkTrieImpl with RootHash 10951270817330706114198641949214391028137561893123097337637233896895686724291\n--------\n", buffer.String())
	buffer.Reset()
}

func TestZkTrie_GetUpdateDelete(t *testing.T) {
	mt := newTestingMerkle(t)
	val, err := mt.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Nil(t, val)
	assert.Equal(t, HashZero.Bytes(), mt.Hash())

	err = mt.TryUpdate([]byte("key"), 1, []Byte32{{1}})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x23, 0x36, 0x5e, 0xbd, 0x71, 0xa7, 0xad, 0x35, 0x65, 0xdd, 0x24, 0x88, 0x47, 0xca, 0xe8, 0xe8, 0x8, 0x21, 0x15, 0x62, 0xc6, 0x83, 0xdb, 0x8, 0x4f, 0x5a, 0xfb, 0xd1, 0xb0, 0x3d, 0x4c, 0xb5}, mt.Hash())

	val, err = mt.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, (&Byte32{1}).Bytes(), val)

	err = mt.TryDelete([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, HashZero.Bytes(), mt.Hash())

	val, err = mt.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Nil(t, val)
}

func TestZkTrie_Copy(t *testing.T) {
	mt := newTestingMerkle(t)

	mt.TryUpdate([]byte("key"), 1, []Byte32{{1}})

	copyTrie := mt.Copy()
	val, err := copyTrie.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, (&Byte32{1}).Bytes(), val)
}

func TestZkTrie_ProveAndProveWithDeletion(t *testing.T) {
	mt := newTestingMerkle(t)

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
			err = mt.Prove(NewHashFromBigInt(k).Bytes(), uint(j), writeNode)
			assert.NoError(t, err)
		}
	}
}
