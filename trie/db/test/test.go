//nolint:gomnd,golint
package test

import (
	"testing"

	"github.com/iden3/go-merkletree/db"
	"github.com/stretchr/testify/assert"
)

// TestReturnKnownErrIfNotExists checks that the implementation of the
// db.Storage interface returns the expected error in the case that the value
// is not found
func TestReturnKnownErrIfNotExists(t *testing.T, sto db.Storage) {
	k := []byte("key")

	tx, err := sto.NewTx()
	assert.Nil(t, err)
	_, err = tx.Get(k)
	assert.EqualError(t, err, db.ErrNotFound.Error())
}

// TestStorageInsertGet checks that the implementation of the db.Storage
// interface behaves as expected
func TestStorageInsertGet(t *testing.T, sto db.Storage) {
	key := []byte("key")
	value := []byte("data")

	tx, err := sto.NewTx()
	assert.Nil(t, err)
	err = tx.Put(key, value)
	assert.Nil(t, err)
	v, err := tx.Get(key)
	assert.Nil(t, err)
	assert.Equal(t, value, v)
	assert.Nil(t, tx.Commit())

	tx, err = sto.NewTx()
	assert.Nil(t, err)
	v, err = tx.Get(key)
	assert.Nil(t, err)
	assert.Equal(t, value, v)
}

// TestStorageWithPrefix checks that the implementation of the db.Storage
// interface behaves as expected for the WithPrefix method
func TestStorageWithPrefix(t *testing.T, sto db.Storage) {
	k := []byte{9}

	sto1 := sto.WithPrefix([]byte{1})
	sto2 := sto.WithPrefix([]byte{2})

	// check within tx

	sto1tx, err := sto1.NewTx()
	assert.Nil(t, err)
	err = sto1tx.Put(k, []byte{4, 5, 6})
	assert.Nil(t, err)
	v1, err := sto1tx.Get(k)
	assert.Nil(t, err)
	assert.Equal(t, v1, []byte{4, 5, 6})
	assert.Nil(t, sto1tx.Commit())

	sto2tx, err := sto2.NewTx()
	assert.Nil(t, err)
	err = sto2tx.Put(k, []byte{8, 9})
	assert.Nil(t, err)
	v2, err := sto2tx.Get(k)
	assert.Nil(t, err)
	assert.Equal(t, v2, []byte{8, 9})
	assert.Nil(t, sto2tx.Commit())

	// check outside tx

	v1, err = sto1.Get(k)
	assert.Nil(t, err)
	assert.Equal(t, v1, []byte{4, 5, 6})

	v2, err = sto2.Get(k)
	assert.Nil(t, err)
	assert.Equal(t, v2, []byte{8, 9})
}

// TestIterate checks that the implementation of the db.Storage interface
// behaves as expected for the Iterate method
func TestIterate(t *testing.T, sto db.Storage) {
	r := []db.KV{}
	lister := func(k []byte, v []byte) (bool, error) {
		r = append(r, db.KV{K: db.Clone(k), V: db.Clone(v)})
		return true, nil
	}

	sto1 := sto.WithPrefix([]byte{1})
	err := sto1.Iterate(lister)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(r))

	sto1tx, _ := sto1.NewTx()
	err = sto1tx.Put([]byte{1}, []byte{4})
	assert.Nil(t, err)
	err = sto1tx.Put([]byte{2}, []byte{5})
	assert.Nil(t, err)
	err = sto1tx.Put([]byte{3}, []byte{6})
	assert.Nil(t, err)
	assert.Nil(t, sto1tx.Commit())

	sto2 := sto.WithPrefix([]byte{2})
	sto2tx, _ := sto2.NewTx()
	err = sto2tx.Put([]byte{1}, []byte{7})
	assert.Nil(t, err)
	err = sto2tx.Put([]byte{2}, []byte{8})
	assert.Nil(t, err)
	err = sto2tx.Put([]byte{3}, []byte{9})
	assert.Nil(t, err)
	assert.Nil(t, sto2tx.Commit())

	r = []db.KV{}
	err = sto1.Iterate(lister)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(r))
	assert.Equal(t, db.KV{K: []byte{1}, V: []byte{4}}, r[0])
	assert.Equal(t, db.KV{K: []byte{2}, V: []byte{5}}, r[1])
	assert.Equal(t, db.KV{K: []byte{3}, V: []byte{6}}, r[2])

	r = []db.KV{}
	err = sto2.Iterate(lister)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(r))
	assert.Equal(t, db.KV{K: []byte{1}, V: []byte{7}}, r[0])
	assert.Equal(t, db.KV{K: []byte{2}, V: []byte{8}}, r[1])
	assert.Equal(t, db.KV{K: []byte{3}, V: []byte{9}}, r[2])
}

// TestConcatTx checks that the implementation of the db.Storage interface
// behaves as expected
func TestConcatTx(t *testing.T, sto db.Storage) {
	k := []byte{9}

	sto1 := sto.WithPrefix([]byte{1})
	sto2 := sto.WithPrefix([]byte{2})

	// check within tx

	sto1tx, err := sto1.NewTx()
	if err != nil {
		panic(err)
	}
	err = sto1tx.Put(k, []byte{4, 5, 6})
	assert.Nil(t, err)
	sto2tx, err := sto2.NewTx()
	if err != nil {
		panic(err)
	}
	err = sto2tx.Put(k, []byte{8, 9})
	assert.Nil(t, err)

	err = sto1tx.Add(sto2tx)
	assert.Nil(t, err)
	assert.Nil(t, sto1tx.Commit())

	// check outside tx

	v1, err := sto1.Get(k)
	assert.Nil(t, err)
	assert.Equal(t, v1, []byte{4, 5, 6})

	v2, err := sto2.Get(k)
	assert.Nil(t, err)
	assert.Equal(t, v2, []byte{8, 9})
}

// TestList checks that the implementation of the db.Storage interface behaves
// as expected
func TestList(t *testing.T, sto db.Storage) {
	sto1 := sto.WithPrefix([]byte{1})
	r1, err := sto1.List(100)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(r1))

	sto1tx, _ := sto1.NewTx()
	err = sto1tx.Put([]byte{1}, []byte{4})
	assert.Nil(t, err)
	err = sto1tx.Put([]byte{2}, []byte{5})
	assert.Nil(t, err)
	err = sto1tx.Put([]byte{3}, []byte{6})
	assert.Nil(t, err)
	assert.Nil(t, sto1tx.Commit())

	sto2 := sto.WithPrefix([]byte{2})
	sto2tx, _ := sto2.NewTx()
	err = sto2tx.Put([]byte{1}, []byte{7})
	assert.Nil(t, err)
	err = sto2tx.Put([]byte{2}, []byte{8})
	assert.Nil(t, err)
	err = sto2tx.Put([]byte{3}, []byte{9})
	assert.Nil(t, err)
	assert.Nil(t, sto2tx.Commit())

	r, err := sto1.List(100)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(r))
	assert.Equal(t, r[0], db.KV{K: []byte{1}, V: []byte{4}})
	assert.Equal(t, r[1], db.KV{K: []byte{2}, V: []byte{5}})
	assert.Equal(t, r[2], db.KV{K: []byte{3}, V: []byte{6}})

	r, err = sto1.List(2)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(r))
	assert.Equal(t, r[0], db.KV{K: []byte{1}, V: []byte{4}})
	assert.Equal(t, r[1], db.KV{K: []byte{2}, V: []byte{5}})
}
