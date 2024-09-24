package zktrie

import (
	"math/big"
	"sync"
)

type ZktrieDatabase interface {
	UpdatePreimage(preimage []byte, hashField *big.Int)
	Put(k, v []byte) error
	Get(key []byte) ([]byte, error)
}

type Database struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func (db *Database) UpdatePreimage([]byte, *big.Int) {}

func (db *Database) Put(k, v []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(k)] = v
	return nil
}

func (db *Database) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return entry, nil
	}
	return nil, ErrKeyNotFound

}

// Init flush db with batches of k/v without locking
func (db *Database) Init(k, v []byte) {
	db.db[string(k)] = v
}

func NewZkTrieMemoryDb() *Database {
	return &Database{
		db: make(map[string][]byte),
	}
}
