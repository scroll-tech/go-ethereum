package trie

import (
	"github.com/syndtr/goleveldb/leveldb"
)

// EthKVStorageTx implements the Tx interface
// A dummy implementation for legacy interface reason
type EthKVStorageTx struct {
	storage *EthKVStorage
}

// Get retreives a value from a key in the interface Tx
func (tx *EthKVStorageTx) Get(key []byte) ([]byte, error) {
	return tx.storage.Get(key)
}

// Put saves a key:value into the Storage
func (tx *EthKVStorageTx) Put(k, v []byte) error {
	return tx.storage.Put(k, v)
}

// Commit implements the method Commit of the interface Tx
func (tx *EthKVStorageTx) Commit() error {
	return nil
}

// EthKVStorage implements the Storage interface
type EthKVStorage struct {
	db *Database
	//cache  KvMap
	prefix []byte
}

// Close implements the method Close of the interface Tx
func (tx *EthKVStorageTx) Close() {
}

func NewEthKVStorage(db *Database) Storage {
	return &EthKVStorage{db, []byte{}}
}

/*
func NewEthKVStorageWithCache(db ethdb.KeyValueStore, cache KvMap) Storage {
	return &EthKVStorage{db, cache, []byte{}}
}
*/
// WithPrefix implements the method WithPrefix of the interface Storage
func (l *EthKVStorage) WithPrefix(prefix []byte) Storage {
	return &EthKVStorage{l.db, Concat(l.prefix, prefix)}
}

// NewTx implements the method NewTx of the interface Storage
func (l *EthKVStorage) NewTx() (Tx, error) {
	return &EthKVStorageTx{l}, nil
}

// Put saves a key:value into the Storage
func (l *EthKVStorage) Put(k, v []byte) error {
	l.db.lock.Lock()
	l.db.rawDirties.Put(Concat(l.prefix, k[:]), v)
	l.db.lock.Unlock()
	return nil
}

// Get retrieves a value from a key in the Storage
func (l *EthKVStorage) Get(key []byte) ([]byte, error) {
	concatKey := Concat(l.prefix, key[:])
	l.db.lock.RLock()
	value, ok := l.db.rawDirties.Get(concatKey)
	l.db.lock.RUnlock()
	if ok {
		return value, nil
	}
	v, err := l.db.diskdb.Get(concatKey)
	if err == leveldb.ErrNotFound {
		return nil, ErrNotFound
	}
	return v, err
}

// Iterate implements the method Iterate of the interface Storage
func (l *EthKVStorage) Iterate(f func([]byte, []byte) (bool, error)) error {
	iter := l.db.diskdb.NewIterator(l.prefix, nil)
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

// Close implements the method Close of the interface Storage
func (l *EthKVStorage) Close() {
	// FIXME: is this correct?
	if err := l.db.diskdb.Close(); err != nil {
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
