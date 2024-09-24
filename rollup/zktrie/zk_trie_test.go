package zktrie

import (
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

func TestNewZkTrie(t *testing.T) {
	root := Byte32{}
	db := NewZkTrieMemoryDb()
	zkTrie, err := NewZkTrie(root, db)
	assert.NoError(t, err)
	assert.Equal(t, HashZero.Bytes(), zkTrie.Hash())
	assert.Equal(t, HashZero.Bytes(), zkTrie.Tree().rootKey.Bytes())

	root = Byte32{1}
	zkTrie, err = NewZkTrie(root, db)
	assert.Equal(t, ErrKeyNotFound, err)
	assert.Nil(t, zkTrie)
}

func TestZkTrie_GetUpdateDelete(t *testing.T) {
	root := Byte32{}
	db := NewZkTrieMemoryDb()
	zkTrie, err := NewZkTrie(root, db)
	assert.NoError(t, err)

	val, err := zkTrie.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Nil(t, val)
	assert.Equal(t, HashZero.Bytes(), zkTrie.Hash())

	err = zkTrie.TryUpdate([]byte("key"), 1, []Byte32{{1}})
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x23, 0x36, 0x5e, 0xbd, 0x71, 0xa7, 0xad, 0x35, 0x65, 0xdd, 0x24, 0x88, 0x47, 0xca, 0xe8, 0xe8, 0x8, 0x21, 0x15, 0x62, 0xc6, 0x83, 0xdb, 0x8, 0x4f, 0x5a, 0xfb, 0xd1, 0xb0, 0x3d, 0x4c, 0xb5}, zkTrie.Hash())

	val, err = zkTrie.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, (&Byte32{1}).Bytes(), val)

	err = zkTrie.TryDelete([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, HashZero.Bytes(), zkTrie.Hash())

	val, err = zkTrie.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Nil(t, val)
}

func TestZkTrie_Copy(t *testing.T) {
	root := Byte32{}
	db := NewZkTrieMemoryDb()
	zkTrie, err := NewZkTrie(root, db)
	assert.NoError(t, err)

	zkTrie.TryUpdate([]byte("key"), 1, []Byte32{{1}})

	copyTrie := zkTrie.Copy()
	val, err := copyTrie.TryGet([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, (&Byte32{1}).Bytes(), val)
}

func TestZkTrie_ProveAndProveWithDeletion(t *testing.T) {
	root := Byte32{}
	db := NewZkTrieMemoryDb()
	zkTrie, err := NewZkTrie(root, db)
	assert.NoError(t, err)

	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	for i, keyStr := range keys {
		key := make([]byte, 32)
		copy(key, []byte(keyStr))

		err := zkTrie.TryUpdate(key, uint32(i+1), []Byte32{{byte(uint32(i + 1))}})
		assert.NoError(t, err)

		writeNode := func(n *Node) error {
			return nil
		}

		onHit := func(n *Node, sib *Node) {}

		k, err := ToSecureKey(key)
		assert.NoError(t, err)

		for j := 0; j <= i; j++ {
			err = zkTrie.ProveWithDeletion(NewHashFromBigInt(k).Bytes(), uint(j), writeNode, onHit)
			assert.NoError(t, err)

			err = zkTrie.Prove(NewHashFromBigInt(k).Bytes(), uint(j), writeNode)
			assert.NoError(t, err)
		}
	}
}
