package leveldb

import (
	"github.com/iden3/go-merkletree/db"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Storage implements the db.Storage interface
type Storage struct {
	ldb    *leveldb.DB
	prefix []byte
}

// StorageTx implements the db.Tx interface
type StorageTx struct {
	*Storage
	cache db.KvMap
}

// NewLevelDbStorage returns a new Storage
func NewLevelDbStorage(path string, errorIfMissing bool) (*Storage, error) {
	o := &opt.Options{
		ErrorIfMissing: errorIfMissing,
	}
	ldb, err := leveldb.OpenFile(path, o)
	if err != nil {
		return nil, err
	}
	return &Storage{ldb, []byte{}}, nil
}

// WithPrefix implements the method WithPrefix of the interface db.Storage
func (l *Storage) WithPrefix(prefix []byte) db.Storage {
	return &Storage{l.ldb, db.Concat(l.prefix, prefix)}
}

// NewTx implements the method NewTx of the interface db.Storage
func (l *Storage) NewTx() (db.Tx, error) {
	return &StorageTx{l, make(db.KvMap)}, nil
}

// Get retreives a value from a key in the db.Storage
func (l *Storage) Get(key []byte) ([]byte, error) {
	v, err := l.ldb.Get(db.Concat(l.prefix, key[:]), nil)
	if err == errors.ErrNotFound {
		return nil, db.ErrNotFound
	}
	return v, err
}

// Iterate implements the method Iterate of the interface db.Storage
func (l *Storage) Iterate(f func([]byte, []byte) (bool, error)) error {
	// FIXME: Use the prefix!
	snapshot, err := l.ldb.GetSnapshot()
	if err != nil {
		return err
	}
	iter := snapshot.NewIterator(util.BytesPrefix(l.prefix), nil)
	defer iter.Release()
	for iter.Next() {
		localKey := iter.Key()[len(l.prefix):]
		if cont, err := f(localKey, iter.Value()); err != nil {
			return err
		} else if !cont {
			break
		}
	}
	iter.Release()
	return iter.Error()
}

// Get retreives a value from a key in the interface db.Tx
func (tx *StorageTx) Get(key []byte) ([]byte, error) {
	var err error

	fullkey := db.Concat(tx.prefix, key)

	if value, ok := tx.cache.Get(fullkey); ok {
		return value, nil
	}

	value, err := tx.ldb.Get(fullkey, nil)
	if err == errors.ErrNotFound {
		return nil, db.ErrNotFound
	}

	return value, err
}

// Put saves a key:value into the db.Storage
func (tx *StorageTx) Put(k, v []byte) error {
	tx.cache.Put(db.Concat(tx.prefix, k[:]), v)
	return nil
}

// Add implements the method Add of the interface db.Tx
func (tx *StorageTx) Add(atx db.Tx) error {
	ldbtx := atx.(*StorageTx)
	for _, v := range ldbtx.cache {
		tx.cache.Put(v.K, v.V)
	}
	return nil
}

// Commit implements the method Commit of the interface db.Tx
func (tx *StorageTx) Commit() error {
	var batch leveldb.Batch
	for _, v := range tx.cache {
		batch.Put(v.K, v.V)
	}

	tx.cache = nil
	return tx.ldb.Write(&batch, nil)
}

// Close implements the method Close of the interface db.Tx
func (tx *StorageTx) Close() {
	tx.cache = nil
}

// Close implements the method Close of the interface db.Storage
func (l *Storage) Close() {
	if err := l.ldb.Close(); err != nil {
		panic(err)
	}
}

// LevelDB is an extra method that returns the *leveldb.DB
func (l *Storage) LevelDB() *leveldb.DB {
	return l.ldb
}

// List implements the method List of the interface db.Storage
func (l *Storage) List(limit int) ([]db.KV, error) {
	ret := []db.KV{}
	err := l.Iterate(func(key []byte, value []byte) (bool, error) {
		ret = append(ret, db.KV{K: db.Clone(key), V: db.Clone(value)})
		if len(ret) == limit {
			return false, nil
		}
		return true, nil
	})
	return ret, err
}
