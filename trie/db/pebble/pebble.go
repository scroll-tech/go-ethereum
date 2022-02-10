package pebble

import (
	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/trie/db"
)

// Storage implements the db.Storage interface
type Storage struct {
	pdb    *pebble.DB
	prefix []byte
}

// StorageTx implements the db.Tx interface
type StorageTx struct {
	*Storage
	batch *pebble.Batch
}

// NewPebbleStorage returns a new Storage
func NewPebbleStorage(path string, errorIfMissing bool) (*Storage, error) {
	o := &pebble.Options{
		ErrorIfNotExists: errorIfMissing,
	}
	rdb, err := pebble.Open(path, o)
	if err != nil {
		return nil, err
	}
	return &Storage{rdb, []byte{}}, nil
}

// WithPrefix implements the method WithPrefix of the interface db.Storage
func (p *Storage) WithPrefix(prefix []byte) db.Storage {
	return &Storage{p.pdb, db.Concat(p.prefix, prefix)}
}

// NewTx implements the method NewTx of the interface db.Storage
func (p *Storage) NewTx() (db.Tx, error) {
	return &StorageTx{p, p.pdb.NewIndexedBatch()}, nil
}

// Get retreives a value from a key in the db.Storage
func (p *Storage) Get(key []byte) ([]byte, error) {
	v, closer, err := p.pdb.Get(db.Concat(p.prefix, key[:]))
	if err == pebble.ErrNotFound {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	err = closer.Close()
	return v, err
}

//nolint:lll
// https://github.com/cockroachdb/pebble/pull/923/files#diff-c2ade2f386c41794d5ebc57ee49b57a5fca8082e03255e5bff13977cbc061287R39
func keyUpperBound(b []byte) []byte {
	end := make([]byte, len(b))
	copy(end, b)
	for i := len(end) - 1; i >= 0; i-- {
		end[i] = end[i] + 1
		if end[i] != 0 {
			return end[:i+1]
		}
	}
	return nil // no upper-bound
}
func prefixIterOptions(prefix []byte) *pebble.IterOptions {
	return &pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: keyUpperBound(prefix),
	}
}

// Iterate implements the method Iterate of the interface db.Storage
func (p *Storage) Iterate(f func([]byte, []byte) (bool, error)) (err error) {
	// NewIter already provides a point-in-time view of the current DB
	// state, but if is used for long term (is not the case), should use an
	// iterator over an snapshot:
	// snapshot := p.pdb.NewSnapshot()
	// defer snapshot.Close()
	// iter := snapshot.NewIter(nil)
	iter := p.pdb.NewIter(prefixIterOptions(p.prefix))
	defer func() {
		err1 := iter.Close()
		if err != nil {
			return
		}
		err = err1
	}()

	for iter.First(); iter.Valid(); iter.Next() {
		localKey := iter.Key()[len(p.prefix):]
		if cont, err := f(localKey, iter.Value()); err != nil {
			return err
		} else if !cont {
			break
		}
	}
	return iter.Error()
}

// Get retreives a value from a key in the interface db.Tx
func (tx *StorageTx) Get(key []byte) ([]byte, error) {
	var err error

	fullkey := db.Concat(tx.prefix, key)

	v, closer, err := tx.batch.Get(fullkey)
	if err == pebble.ErrNotFound {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	err = closer.Close()
	return v, err
}

// Put saves a key:value into the db.Storage
func (tx *StorageTx) Put(k, v []byte) error {
	return tx.batch.Set(db.Concat(tx.prefix, k[:]), v, nil)
}

// Add implements the method Add of the interface db.Tx
func (tx *StorageTx) Add(atx db.Tx) error {
	patx := atx.(*StorageTx)
	return tx.batch.Apply(patx.batch, nil)
}

// Commit implements the method Commit of the interface db.Tx
func (tx *StorageTx) Commit() error {
	return tx.batch.Commit(nil)
}

// Close implements the method Close of the interface db.Tx
func (tx *StorageTx) Close() {
	_ = tx.batch.Close()
}

// Close implements the method Close of the interface db.Storage
func (p *Storage) Close() {
	if err := p.pdb.Close(); err != nil {
		panic(err)
	}
}

// Pebble is an extra method that returns the *pebble.DB
func (p *Storage) Pebble() *pebble.DB {
	return p.pdb
}

// List implements the method List of the interface db.Storage
func (p *Storage) List(limit int) ([]db.KV, error) {
	ret := []db.KV{}
	err := p.Iterate(func(key []byte, value []byte) (bool, error) {
		ret = append(ret, db.KV{K: db.Clone(key), V: db.Clone(value)})
		if len(ret) == limit {
			return false, nil
		}
		return true, nil
	})
	return ret, err
}
