package zktrie

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// we do not need zktrie impl anymore, only made a wrapper for adapting testing
type zkTrieImplTestWrapper struct {
	*ZkTrieImpl
}

func newZkTrieImpl(storage ZktrieDatabase, maxLevels int) (*zkTrieImplTestWrapper, error) {
	return newZkTrieImplWithRoot(storage, &HashZero, maxLevels)
}

// NewZkTrieImplWithRoot loads a new ZkTrieImpl. If in the storage already exists one
// will open that one, if not, will create a new one.
func newZkTrieImplWithRoot(storage ZktrieDatabase, root *Hash, maxLevels int) (*zkTrieImplTestWrapper, error) {
	impl, err := NewZkTrieImplWithRoot(storage, root, maxLevels)
	if err != nil {
		return nil, err
	}

	return &zkTrieImplTestWrapper{impl}, nil
}

func (mt *zkTrieImplTestWrapper) AddWord(kPreimage, vPreimage *Byte32) error {

	if v, _ := mt.TryGet(kPreimage[:]); v != nil {
		return ErrEntryIndexAlreadyExists
	}

	return mt.ZkTrieImpl.TryUpdate(NewHashFromBytes(kPreimage[:]), 1, []Byte32{*vPreimage})
}

func (mt *zkTrieImplTestWrapper) GetLeafNodeByWord(kPreimage *Byte32) (*Node, error) {
	return mt.ZkTrieImpl.GetLeafNode(NewHashFromBytes(kPreimage[:]))
}

func (mt *zkTrieImplTestWrapper) UpdateWord(kPreimage, vPreimage *Byte32) error {
	return mt.ZkTrieImpl.TryUpdate(NewHashFromBytes(kPreimage[:]), 1, []Byte32{*vPreimage})
}

func (mt *zkTrieImplTestWrapper) DeleteWord(kPreimage *Byte32) error {
	return mt.ZkTrieImpl.TryDelete(NewHashFromBytes(kPreimage[:]))
}

func (mt *zkTrieImplTestWrapper) TryGet(key []byte) ([]byte, error) {
	return mt.ZkTrieImpl.TryGet(NewHashFromBytes(key))
}

func newTestingMerkle(t *testing.T, numLevels int) *zkTrieImplTestWrapper {
	mt, err := newZkTrieImpl(NewZkTrieMemoryDb(), numLevels)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	mt.Debug = true
	assert.Equal(t, numLevels, mt.MaxLevels())
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
		err = mt1.TryUpdate(NewHashFromBytes([]byte{1}), 1, []Byte32{{byte(1)}})
		assert.NoError(t, err)
		mt1Root, err = mt1.Root()
		assert.NoError(t, err)
		assert.Equal(t, "0539c6b1cac741eb1e98b2c271733d1e6f0fad557228f6b039d894b0a627c8d9", mt1Root.Hex())
		assert.NoError(t, mt1.Commit())

		mt2, err := NewZkTrieImplWithRoot(db, mt1Root, maxLevels)
		assert.NoError(t, err)
		assert.Equal(t, maxLevels, mt2.maxLevels)
		mt2Root, err := mt2.Root()
		assert.NoError(t, err)
		assert.Equal(t, "0539c6b1cac741eb1e98b2c271733d1e6f0fad557228f6b039d894b0a627c8d9", mt2Root.Hex())
	})

	t.Run("Test NewZkTrieImplWithRoot with non-zero hash root and node does not exist", func(t *testing.T) {
		root := NewHashFromBytes([]byte{1, 2, 3, 4, 5})

		mt, err := NewZkTrieImplWithRoot(db, root, maxLevels)
		assert.Error(t, err)
		assert.Nil(t, mt)
	})
}

func TestMerkleTree_AddUpdateGetWord(t *testing.T) {
	mt := newTestingMerkle(t, 10)

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
		err := mt.AddWord(NewByte32FromBytes([]byte{td.key}), &Byte32{td.initialVal})
		assert.NoError(t, err)

		node, err := mt.GetLeafNodeByWord(NewByte32FromBytes([]byte{td.key}))
		assert.NoError(t, err)
		assert.Equal(t, 1, len(node.ValuePreimage))
		assert.Equal(t, (&Byte32{td.initialVal})[:], node.ValuePreimage[0][:])
	}

	err := mt.AddWord(NewByte32FromBytes([]byte{5}), &Byte32{7})
	assert.Equal(t, ErrEntryIndexAlreadyExists, err)

	for _, td := range testData {
		err := mt.UpdateWord(NewByte32FromBytes([]byte{td.key}), &Byte32{td.updatedVal})
		assert.NoError(t, err)

		node, err := mt.GetLeafNodeByWord(NewByte32FromBytes([]byte{td.key}))
		assert.NoError(t, err)
		assert.Equal(t, 1, len(node.ValuePreimage))
		assert.Equal(t, (&Byte32{td.updatedVal})[:], node.ValuePreimage[0][:])
	}

	_, err = mt.GetLeafNodeByWord(&Byte32{100})
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestMerkleTree_Deletion(t *testing.T) {
	t.Run("Check root consistency", func(t *testing.T) {
		var err error
		mt := newTestingMerkle(t, 10)
		hashes := make([]*Hash, 7)
		hashes[0], err = mt.Root()
		assert.NoError(t, err)

		for i := 0; i < 6; i++ {
			err := mt.AddWord(NewByte32FromBytes([]byte{byte(i)}), &Byte32{byte(i)})
			assert.NoError(t, err)
			hashes[i+1], err = mt.Root()
			assert.NoError(t, err)
		}

		for i := 5; i >= 0; i-- {
			err := mt.DeleteWord(NewByte32FromBytes([]byte{byte(i)}))
			assert.NoError(t, err)
			root, err := mt.Root()
			assert.NoError(t, err)
			assert.Equal(t, hashes[i], root, i)
		}
	})

	t.Run("Check depth", func(t *testing.T) {
		mt := newTestingMerkle(t, 10)
		key1 := NewByte32FromBytes([]byte{67}) //0b1000011
		err := mt.AddWord(key1, &Byte32{67})
		assert.NoError(t, err)
		rootPhase1, err := mt.Root()
		assert.NoError(t, err)
		key2 := NewByte32FromBytes([]byte{131}) //0b10000011
		err = mt.AddWord(key2, &Byte32{131})
		assert.NoError(t, err)
		rootPhase2, err := mt.Root()
		assert.NoError(t, err)

		assertKeyDepth := func(key *Byte32, expectedDep int) {
			levelCnt := 0
			err := mt.Prove(NewHashFromBytes(key[:]), 0,
				func(*Node) error {
					levelCnt++
					return nil
				},
			)
			assert.NoError(t, err)
			assert.Equal(t, expectedDep, levelCnt)
		}

		assertKeyDepth(key1, 8)
		assertKeyDepth(key2, 8)

		err = mt.DeleteWord(key2)
		assert.NoError(t, err)

		assertKeyDepth(key1, 1)
		curRoot, err := mt.Root()
		assert.NoError(t, err)
		assert.Equal(t, rootPhase1, curRoot)

		err = mt.AddWord(key2, &Byte32{131})
		assert.NoError(t, err)
		curRoot, err = mt.Root()
		assert.NoError(t, err)
		assert.Equal(t, rootPhase2, curRoot)
		assertKeyDepth(key1, 8)

		// delete node with parent sibling (fail before a410f14)
		key3 := NewByte32FromBytes([]byte{19}) //0b10011
		err = mt.AddWord(key3, &Byte32{19})
		assert.NoError(t, err)

		err = mt.DeleteWord(key3)
		assert.NoError(t, err)
		assertKeyDepth(key1, 8)
		curRoot, err = mt.Root()
		assert.NoError(t, err)
		assert.Equal(t, rootPhase2, curRoot)

		key4 := NewByte32FromBytes([]byte{4}) //0b100, so it is 2 level node (fail before d1c735)
		err = mt.AddWord(key4, &Byte32{4})
		assert.NoError(t, err)

		assertKeyDepth(key4, 2)

		err = mt.DeleteWord(key4)
		assert.NoError(t, err)
		curRoot, err = mt.Root()
		assert.NoError(t, err)
		assert.Equal(t, rootPhase2, curRoot)
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
			mt := newTestingMerkle(t, 10)
			for _, key := range order {
				value := kvMap[key]
				err := mt.AddWord(key, value)
				assert.NoError(t, err)
			}
			var err error
			roots[i], err = mt.Root()
			assert.NoError(t, err)
		}

		assert.Equal(t, "225fe589e8cbdfe424a032e6e2fd1132762b20794cff61f0c70e8f757b6a0ed7", roots[0].Hex())
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
			mt := newTestingMerkle(t, 10)
			for _, key := range order {
				value := kvMap[key]
				err := mt.AddWord(key, value)
				assert.NoError(t, err)
			}
			var err error
			roots[i], err = mt.Root()
			assert.NoError(t, err)
		}

		for i := 1; i < len(roots); i++ {
			assert.Equal(t, "25aa478a6c8c3a7cab40b0c3a37f8ed6815ee575228f0ba8e77d1145191f9a34", roots[0].Hex())
			assert.Equal(t, roots[0], roots[i])
		}
	})

	t.Run("Add twice", func(t *testing.T) {
		keys := []*Byte32{k1, k2, k3}

		mt := newTestingMerkle(t, 10)
		for _, key := range keys {
			err := mt.AddWord(key, kvMap[key])
			assert.NoError(t, err)

			err = mt.AddWord(key, kvMap[key])
			assert.Equal(t, ErrEntryIndexAlreadyExists, err)
		}
	})
}

func TestZkTrieImpl_Update(t *testing.T) {
	k1 := NewByte32FromBytes([]byte{1})
	k2 := NewByte32FromBytes([]byte{2})
	k3 := NewByte32FromBytes([]byte{3})

	t.Run("Update 1", func(t *testing.T) {
		mt1 := newTestingMerkle(t, 10)
		err := mt1.AddWord(k1, NewByte32FromBytes([]byte{1}))
		assert.NoError(t, err)
		root1, err := mt1.Root()
		assert.NoError(t, err)

		mt2 := newTestingMerkle(t, 10)
		err = mt2.AddWord(k1, NewByte32FromBytes([]byte{2}))
		assert.NoError(t, err)
		err = mt2.UpdateWord(k1, NewByte32FromBytes([]byte{1}))
		assert.NoError(t, err)
		root2, err := mt2.Root()
		assert.NoError(t, err)

		assert.Equal(t, root1, root2)
	})

	t.Run("Update 2", func(t *testing.T) {
		mt1 := newTestingMerkle(t, 10)
		err := mt1.AddWord(k1, NewByte32FromBytes([]byte{1}))
		assert.NoError(t, err)
		err = mt1.AddWord(k2, NewByte32FromBytes([]byte{2}))
		assert.NoError(t, err)
		root1, err := mt1.Root()
		assert.NoError(t, err)

		mt2 := newTestingMerkle(t, 10)
		err = mt2.AddWord(k1, NewByte32FromBytes([]byte{1}))
		assert.NoError(t, err)
		err = mt2.AddWord(k2, NewByte32FromBytes([]byte{3}))
		assert.NoError(t, err)
		err = mt2.UpdateWord(k2, NewByte32FromBytes([]byte{2}))
		assert.NoError(t, err)
		root2, err := mt2.Root()
		assert.NoError(t, err)

		assert.Equal(t, root1, root2)
	})

	t.Run("Update 1, 2, 3", func(t *testing.T) {
		mt1 := newTestingMerkle(t, 10)
		mt2 := newTestingMerkle(t, 10)
		keys := []*Byte32{k1, k2, k3}
		for i, key := range keys {
			err := mt1.AddWord(key, NewByte32FromBytes([]byte{byte(i)}))
			assert.NoError(t, err)
		}
		for i, key := range keys {
			err := mt2.AddWord(key, NewByte32FromBytes([]byte{byte(i + 3)}))
			assert.NoError(t, err)
		}
		for i, key := range keys {
			err := mt1.UpdateWord(key, NewByte32FromBytes([]byte{byte(i + 6)}))
			assert.NoError(t, err)
			err = mt2.UpdateWord(key, NewByte32FromBytes([]byte{byte(i + 6)}))
			assert.NoError(t, err)
		}

		root1, err := mt1.Root()
		assert.NoError(t, err)
		root2, err := mt2.Root()
		assert.NoError(t, err)

		assert.Equal(t, root1, root2)
	})

	t.Run("Update same value", func(t *testing.T) {
		mt := newTestingMerkle(t, 10)
		keys := []*Byte32{k1, k2, k3}
		for _, key := range keys {
			err := mt.AddWord(key, NewByte32FromBytes([]byte{1}))
			assert.NoError(t, err)
			err = mt.UpdateWord(key, NewByte32FromBytes([]byte{1}))
			assert.NoError(t, err)
			node, err := mt.GetLeafNodeByWord(key)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(node.ValuePreimage))
			assert.Equal(t, NewByte32FromBytes([]byte{1}).Bytes(), node.ValuePreimage[0][:])
		}
	})

	t.Run("Update non-existent word", func(t *testing.T) {
		mt := newTestingMerkle(t, 10)
		err := mt.UpdateWord(k1, NewByte32FromBytes([]byte{1}))
		assert.NoError(t, err)
		node, err := mt.GetLeafNodeByWord(k1)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(node.ValuePreimage))
		assert.Equal(t, NewByte32FromBytes([]byte{1}).Bytes(), node.ValuePreimage[0][:])
	})
}

func TestZkTrieImpl_Delete(t *testing.T) {
	k1 := NewByte32FromBytes([]byte{1})
	k2 := NewByte32FromBytes([]byte{2})
	k3 := NewByte32FromBytes([]byte{3})
	k4 := NewByte32FromBytes([]byte{4})

	t.Run("Test deletion leads to empty tree", func(t *testing.T) {
		emptyMT := newTestingMerkle(t, 10)
		emptyMTRoot, err := emptyMT.Root()
		assert.NoError(t, err)

		mt1 := newTestingMerkle(t, 10)
		err = mt1.AddWord(k1, NewByte32FromBytes([]byte{1}))
		assert.NoError(t, err)
		err = mt1.DeleteWord(k1)
		assert.NoError(t, err)
		mt1Root, err := mt1.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero, *mt1Root)
		assert.Equal(t, emptyMTRoot, mt1Root)

		keys := []*Byte32{k1, k2, k3, k4}
		mt2 := newTestingMerkle(t, 10)
		for _, key := range keys {
			err := mt2.AddWord(key, NewByte32FromBytes([]byte{1}))
			assert.NoError(t, err)
		}
		for _, key := range keys {
			err := mt2.DeleteWord(key)
			assert.NoError(t, err)
		}
		mt2Root, err := mt2.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero, *mt2Root)
		assert.Equal(t, emptyMTRoot, mt2Root)

		mt3 := newTestingMerkle(t, 10)
		for _, key := range keys {
			err := mt3.AddWord(key, NewByte32FromBytes([]byte{1}))
			assert.NoError(t, err)
		}
		for i := len(keys) - 1; i >= 0; i-- {
			err := mt3.DeleteWord(keys[i])
			assert.NoError(t, err)
		}
		mt3Root, err := mt3.Root()
		assert.NoError(t, err)
		assert.Equal(t, HashZero, *mt3Root)
		assert.Equal(t, emptyMTRoot, mt3Root)
	})

	t.Run("Test equivalent trees after deletion", func(t *testing.T) {
		keys := []*Byte32{k1, k2, k3, k4}

		mt1 := newTestingMerkle(t, 10)
		for i, key := range keys {
			err := mt1.AddWord(key, NewByte32FromBytes([]byte{byte(i + 1)}))
			assert.NoError(t, err)
		}
		err := mt1.DeleteWord(k1)
		assert.NoError(t, err)
		err = mt1.DeleteWord(k2)
		assert.NoError(t, err)

		mt2 := newTestingMerkle(t, 10)
		err = mt2.AddWord(k3, NewByte32FromBytes([]byte{byte(3)}))
		assert.NoError(t, err)
		err = mt2.AddWord(k4, NewByte32FromBytes([]byte{byte(4)}))
		assert.NoError(t, err)

		mt1Root, err := mt1.Root()
		assert.NoError(t, err)
		mt2Root, err := mt2.Root()
		assert.NoError(t, err)

		assert.Equal(t, mt1Root, mt2Root)

		mt3 := newTestingMerkle(t, 10)
		for i, key := range keys {
			err := mt3.AddWord(key, NewByte32FromBytes([]byte{byte(i + 1)}))
			assert.NoError(t, err)
		}
		err = mt3.DeleteWord(k1)
		assert.NoError(t, err)
		err = mt3.DeleteWord(k3)
		assert.NoError(t, err)
		mt4 := newTestingMerkle(t, 10)
		err = mt4.AddWord(k2, NewByte32FromBytes([]byte{2}))
		assert.NoError(t, err)
		err = mt4.AddWord(k4, NewByte32FromBytes([]byte{4}))
		assert.NoError(t, err)

		mt3Root, err := mt3.Root()
		assert.NoError(t, err)
		mt4Root, err := mt4.Root()
		assert.NoError(t, err)

		assert.Equal(t, mt3Root, mt4Root)
	})

	t.Run("Test repeat deletion", func(t *testing.T) {
		mt := newTestingMerkle(t, 10)
		err := mt.AddWord(k1, NewByte32FromBytes([]byte{1}))
		assert.NoError(t, err)
		err = mt.DeleteWord(k1)
		assert.NoError(t, err)
		err = mt.DeleteWord(k1)
		assert.Equal(t, ErrKeyNotFound, err)
	})

	t.Run("Test deletion of non-existent node", func(t *testing.T) {
		mt := newTestingMerkle(t, 10)
		err := mt.DeleteWord(k1)
		assert.Equal(t, ErrKeyNotFound, err)
	})
}

func TestMerkleTree_BuildAndVerifyZkTrieProof(t *testing.T) {
	zkTrie := newTestingMerkle(t, 10)

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
		err := zkTrie.AddWord(NewByte32FromBytes([]byte{byte(td.key.Int64())}), &Byte32{td.value})
		assert.NoError(t, err)
	}

	t.Run("Test with existent key", func(t *testing.T) {
		for _, td := range testData {

			node, err := zkTrie.GetLeafNodeByWord(NewByte32FromBytes([]byte{byte(td.key.Int64())}))
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
		nodeAnother, err := zkTrie.GetLeafNodeByWord(NewByte32FromBytes([]byte{byte(big.NewInt(1).Int64())}))
		assert.NoError(t, err)
		valid = VerifyProofZkTrie(zkTrie.rootKey, proof, nodeAnother)
		assert.False(t, valid)

		hash, err := proof.Verify(node.nodeHash)
		assert.NoError(t, err)
		assert.Equal(t, hash[:], zkTrie.rootKey[:])
	})
}

func TestMerkleTree_GraphViz(t *testing.T) {
	mt := newTestingMerkle(t, 10)

	var buffer bytes.Buffer
	err := mt.GraphViz(&buffer, nil)
	assert.NoError(t, err)
	assert.Equal(t, "--------\nGraphViz of the ZkTrieImpl with RootHash 0\ndigraph hierarchy {\nnode [fontname=Monospace,fontsize=10,shape=box]\n}\nEnd of GraphViz of the ZkTrieImpl with RootHash 0\n--------\n", buffer.String())
	buffer.Reset()

	key1 := NewByte32FromBytes([]byte{1}) //0b1
	err = mt.AddWord(key1, &Byte32{1})
	assert.NoError(t, err)
	key2 := NewByte32FromBytes([]byte{3}) //0b11
	err = mt.AddWord(key2, &Byte32{3})
	assert.NoError(t, err)

	err = mt.GraphViz(&buffer, nil)
	assert.NoError(t, err)
	assert.Equal(t, "--------\nGraphViz of the ZkTrieImpl with RootHash 18814328259272153650095812929528579893472885385393031263032639585810677019057\ndigraph hierarchy {\nnode [fontname=Monospace,fontsize=10,shape=box]\n\"18814328...\" -> {\"empty0\" \"36062889...\"}\n\"empty0\" [style=dashed,label=0];\n\"36062889...\" -> {\"23636458...\" \"20814118...\"}\n\"23636458...\" [style=filled];\n\"20814118...\" [style=filled];\n}\nEnd of GraphViz of the ZkTrieImpl with RootHash 18814328259272153650095812929528579893472885385393031263032639585810677019057\n--------\n", buffer.String())
	buffer.Reset()
}
