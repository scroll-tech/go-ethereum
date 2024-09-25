package zktrie

import (
	"math/big"
	"sync"
)

type MemDatabase struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func (db *MemDatabase) UpdatePreimage([]byte, *big.Int) {}

func (db *MemDatabase) Put(k, v []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(k)] = v
	return nil
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return entry, nil
	}
	return nil, ErrKeyNotFound

}

// Init flush db with batches of k/v without locking
func (db *MemDatabase) Init(k, v []byte) {
	db.db[string(k)] = v
}

func NewZkTrieMemoryDb() *MemDatabase {
	return &MemDatabase{
		db: make(map[string][]byte),
	}
}
