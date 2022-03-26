package db

import (
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/scroll-tech/go-ethereum/ethdb"
)

// EthKVStorage implements the Storage interface
type EthKVStorage struct {
	db     ethdb.KeyValueStore
	prefix []byte
}

// EthKVStorageTx implements the Tx interface
type EthKVStorageTx struct {
	*EthKVStorage
	cache KvMap
}

func NewEthKVStorage(db ethdb.KeyValueStore) Storage {
	return &EthKVStorage{db, []byte{}}
}

// WithPrefix implements the method WithPrefix of the interface Storage
func (l *EthKVStorage) WithPrefix(prefix []byte) Storage {
	return &EthKVStorage{l.db, Concat(l.prefix, prefix)}
}

// NewTx implements the method NewTx of the interface Storage
func (l *EthKVStorage) NewTx() (Tx, error) {
	return &EthKVStorageTx{l, make(KvMap)}, nil
}

// Get retrieves a value from a key in the Storage
func (l *EthKVStorage) Get(key []byte) ([]byte, error) {
	concatKey := Concat(l.prefix, key[:])
	v, err := l.db.Get(concatKey)
	if err == leveldb.ErrNotFound {
		return nil, ErrNotFound
	}
	return v, err
}

// Iterate implements the method Iterate of the interface Storage
func (l *EthKVStorage) Iterate(f func([]byte, []byte) (bool, error)) error {
	iter := l.db.NewIterator(l.prefix, nil)
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

// Get retreives a value from a key in the interface Tx
func (tx *EthKVStorageTx) Get(key []byte) ([]byte, error) {
	var err error

	fullkey := Concat(tx.prefix, key)

	if value, ok := tx.cache.Get(fullkey); ok {
		return value, nil
	}

	value, err := tx.db.Get(fullkey)
	if err == leveldb.ErrNotFound {
		return nil, ErrNotFound
	}

	return value, err
}

// Put saves a key:value into the Storage
func (tx *EthKVStorageTx) Put(k, v []byte) error {
	tx.cache.Put(Concat(tx.prefix, k[:]), v)
	return nil
}

// Add implements the method Add of the interface Tx
func (tx *EthKVStorageTx) Add(atx Tx) error {
	ldbtx := atx.(*EthKVStorageTx)
	for _, v := range ldbtx.cache {
		tx.cache.Put(v.K, v.V)
	}
	return nil
}

// Commit implements the method Commit of the interface Tx
func (tx *EthKVStorageTx) Commit() error {
	batch := tx.db.NewBatch()
	for _, v := range tx.cache {
		batch.Put(v.K, v.V)
	}

	tx.cache = nil
	return batch.Write()
}

// Close implements the method Close of the interface Tx
func (tx *EthKVStorageTx) Close() {
	tx.cache = nil
}

// Close implements the method Close of the interface Storage
func (l *EthKVStorage) Close() {
	if err := l.db.Close(); err != nil {
		panic(err)
	}
}

// List implements the method List of the interface Storage
func (l *EthKVStorage) List(limit int) ([]KV, error) {
	ret := []KV{}
	err := l.Iterate(func(key []byte, value []byte) (bool, error) {
		ret = append(ret, KV{K: Clone(key), V: Clone(value)})
		if len(ret) == limit {
			return false, nil
		}
		return true, nil
	})
	return ret, err
}
