package zktrie

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math/big"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/syndtr/goleveldb/leveldb"

	itrie "github.com/scroll-tech/zktrie/trie"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
)

var (
	memcacheCleanHitMeter   = metrics.NewRegisteredMeter("zktrie/memcache/clean/hit", nil)
	memcacheCleanMissMeter  = metrics.NewRegisteredMeter("zktrie/memcache/clean/miss", nil)
	memcacheCleanReadMeter  = metrics.NewRegisteredMeter("zktrie/memcache/clean/read", nil)
	memcacheCleanWriteMeter = metrics.NewRegisteredMeter("zktrie/memcache/clean/write", nil)

	memcacheDirtyHitMeter  = metrics.NewRegisteredMeter("zktrie/memcache/dirty/hit", nil)
	memcacheDirtyMissMeter = metrics.NewRegisteredMeter("zktrie/memcache/dirty/miss", nil)
	memcacheDirtyReadMeter = metrics.NewRegisteredMeter("zktrie/memcache/dirty/read", nil)
	//memcacheDirtyWriteMeter = metrics.NewRegisteredMeter("zktrie/memcache/dirty/write", nil)

	//memcacheFlushTimeTimer  = metrics.NewRegisteredResettingTimer("zktrie/memcache/flush/time", nil)
	//memcacheFlushNodesMeter = metrics.NewRegisteredMeter("zktrie/memcache/flush/nodes", nil)
	//memcacheFlushSizeMeter  = metrics.NewRegisteredMeter("zktrie/memcache/flush/size", nil)

	//memcacheGCTimeTimer  = metrics.NewRegisteredResettingTimer("zktrie/memcache/gc/time", nil)
	//memcacheGCNodesMeter = metrics.NewRegisteredMeter("zktrie/memcache/gc/nodes", nil)
	//memcacheGCSizeMeter  = metrics.NewRegisteredMeter("zktrie/memcache/gc/size", nil)

	//memcacheCommitTimeTimer  = metrics.NewRegisteredResettingTimer("zktrie/memcache/commit/time", nil)
	//memcacheCommitNodesMeter = metrics.NewRegisteredMeter("zktrie/memcache/commit/nodes", nil)
	//memcacheCommitSizeMeter  = metrics.NewRegisteredMeter("zktrie/memcache/commit/size", nil)
)

// ErrNotFound is used by the implementations of the interface db.Storage for
// when a key is not found in the storage
var ErrNotFound = errors.New("key not found")

// KV contains a key (K) and a value (V)
type KV struct {
	K []byte
	V []byte
}

// KvMap is a key-value map between a sha256 byte array hash, and a KV struct
type KvMap map[[sha256.Size]byte]KV

// Get retreives the value respective to a key from the KvMap
func (m KvMap) Get(k []byte) ([]byte, bool) {
	v, ok := m[sha256.Sum256(k)]
	return v.V, ok
}

// Put stores a key and a value in the KvMap
func (m KvMap) Put(k, v []byte) {
	m[sha256.Sum256(k)] = KV{k, v}
}

// Delete delete the value respective to a key from the KvMap
func (m KvMap) Delete(k []byte) {
	delete(m, sha256.Sum256(k))
}

// Concat concatenates arrays of bytes
func Concat(vs ...[]byte) []byte {
	var b bytes.Buffer
	for _, v := range vs {
		b.Write(v)
	}
	return b.Bytes()
}

// Clone clones a byte array into a new byte array
func Clone(b0 []byte) []byte {
	b1 := make([]byte, len(b0))
	copy(b1, b0)
	return b1
}

var (
	cachedNodeSize = int(reflect.TypeOf(KV{}).Size())
)

// Database Database adaptor imple zktrie.ZktrieDatbase
type Database struct {
	diskdb ethdb.KeyValueStore // Persistent storage for matured trie nodes

	cleans  *fastcache.Cache // GC friendly memory cache of clean node RLPs
	dirties KvMap

	preimages *preimageStore

	lock sync.RWMutex
}

// Config defines all necessary options for database.
type Config struct {
	Cache     int  // Memory allowance (MB) to use for caching trie nodes in memory
	Preimages bool // Flag whether the preimage of trie key is recorded
}

func NewDatabase(diskdb ethdb.KeyValueStore) *Database {
	return NewDatabaseWithConfig(diskdb, nil)
}

func NewDatabaseWithConfig(diskdb ethdb.KeyValueStore, config *Config) *Database {
	var cleans *fastcache.Cache
	if config != nil && config.Cache > 0 {
		cleans = fastcache.New(config.Cache * 1024 * 1024)
	}
	db := &Database{
		diskdb:  diskdb,
		cleans:  cleans,
		dirties: make(KvMap),
	}
	// enable preimage in default
	if config == nil || config.Preimages {
		db.preimages = newPreimageStore(diskdb)
	}
	return db
}

// Put saves a key:value into the Storage
func (db *Database) Put(k, v []byte) error {
	db.lock.Lock()
	db.dirties.Put(k, v)
	db.lock.Unlock()
	return nil
}

// Get retrieves a value from a key in the Storage
func (db *Database) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	value, ok := db.dirties.Get(key)
	db.lock.RUnlock()
	if ok {
		return value, nil
	}

	if db.cleans != nil {
		if enc := db.cleans.Get(nil, key); enc != nil {
			memcacheCleanHitMeter.Mark(1)
			memcacheCleanReadMeter.Mark(int64(len(enc)))
			return enc, nil
		}
	}

	v, err := db.diskdb.Get(key)
	if err == leveldb.ErrNotFound {
		return nil, itrie.ErrKeyNotFound
	}
	if db.cleans != nil {
		db.cleans.Set(key, v)
		memcacheCleanMissMeter.Mark(1)
		memcacheCleanWriteMeter.Mark(int64(len(v)))
	}
	return v, err
}

func (db *Database) UpdatePreimage(preimage []byte, hashField *big.Int) {
	if db.preimages != nil { // Ugly direct check but avoids the below write lock
		// we must copy the input key
		db.preimages.insertPreimage(map[common.Hash][]byte{common.BytesToHash(hashField.Bytes()): common.CopyBytes(preimage)})
	}
}

// Iterate implements the method Iterate of the interface Storage
func (db *Database) Iterate(f func([]byte, []byte) (bool, error)) error {
	iter := db.diskdb.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		localKey := iter.Key()
		if cont, err := f(localKey, iter.Value()); err != nil {
			return err
		} else if !cont {
			break
		}
	}
	iter.Release()
	return iter.Error()
}

// Nodes retrieves the hashes of all the nodes cached within the memory database.
// This method is extremely expensive and should only be used to validate internal
// states in test code.
func (db *Database) Nodes() []common.Hash {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var hashes = make([]common.Hash, 0, len(db.dirties))
	for _, kv := range db.dirties {
		hash := NodeHashFromStoreKey(kv.K)
		if hash != (common.Hash{}) { // Special case for "root" references/nodes
			hashes = append(hashes, hash)
		}
	}
	return hashes
}

func (db *Database) Reference(child common.Hash, parent common.Hash) {
	//TODO:
}

func (db *Database) Dereference(root common.Hash) {
	// mimic the logic of database garbage collection behaviour
	//TODO:

	db.lock.Lock()
	defer db.lock.Unlock()

	storeKey := StoreHashFromNodeHash(root)
	db.dirties.Delete(storeKey[:])

	log.Debug("Dereferenced trie from memory database", "root", root)
}

// Close implements the method Close of the interface Storage
func (db *Database) Close() {
	// FIXME: is this correct?
	if err := db.diskdb.Close(); err != nil {
		panic(err)
	}
}

// List implements the method List of the interface Storage
func (db *Database) List(limit int) ([]KV, error) {
	ret := []KV{}
	err := db.Iterate(func(key []byte, value []byte) (bool, error) {
		ret = append(ret, KV{K: Clone(key), V: Clone(value)})
		if len(ret) == limit {
			return false, nil
		}
		return true, nil
	})
	return ret, err
}

func (db *Database) Commit(node common.Hash, report bool, callback func(common.Hash)) error {
	batch := db.diskdb.NewBatch()

	db.lock.Lock()
	for _, v := range db.dirties {
		batch.Put(v.K, v.V)
	}
	for k := range db.dirties {
		delete(db.dirties, k)
	}
	db.lock.Unlock()
	if err := batch.Write(); err != nil {
		return err
	}
	batch.Reset()

	if (node == common.Hash{}) {
		return nil
	}

	if db.preimages != nil {
		db.preimages.commit(true)
	}
	return nil
}

// DiskDB retrieves the persistent storage backing the trie database.
func (db *Database) DiskDB() ethdb.KeyValueStore {
	return db.diskdb
}

// EmptyRoot indicate what root is for an empty trie
func (db *Database) EmptyRoot() common.Hash {
	return emptyRoot
}

// saveCache saves clean state cache to given directory path
// using specified CPU cores.
func (db *Database) saveCache(dir string, threads int) error {
	//TODO: impelement it?
	return nil
}

// SaveCachePeriodically atomically saves fast cache data to the given dir with
// the specified interval. All dump operation will only use a single CPU core.
func (db *Database) SaveCachePeriodically(dir string, interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			db.saveCache(dir, 1)
		case <-stopCh:
			return
		}
	}
}

func (db *Database) Size() (common.StorageSize, common.StorageSize) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var imgSize common.StorageSize = 0
	if db.preimages != nil {
		imgSize = db.preimages.size()
	}
	return common.StorageSize(len(db.dirties) * cachedNodeSize), imgSize
}

func (db *Database) SaveCache(dir string) error {
	return db.saveCache(dir, runtime.GOMAXPROCS(0))
}

func (db *Database) Node(hash common.Hash) ([]byte, error) {
	if hash == (common.Hash{}) {
		return itrie.NewEmptyNode().CanonicalValue(), nil
	}
	key := StoreHashFromNodeHash(hash)[:]
	// Retrieve the node from the clean cache if available
	if db.cleans != nil {
		if enc := db.cleans.Get(nil, key); enc != nil {
			memcacheCleanHitMeter.Mark(1)
			memcacheCleanReadMeter.Mark(int64(len(enc)))
			return enc, nil
		}
	}
	// Retrieve the node from the dirty cache if available
	db.lock.RLock()
	dirty, _ := db.dirties.Get(key)
	db.lock.RUnlock()

	if dirty != nil {
		memcacheDirtyHitMeter.Mark(1)
		memcacheDirtyReadMeter.Mark(int64(len(dirty)))
		return dirty, nil
	}
	memcacheDirtyMissMeter.Mark(1)

	// Content unavailable in memory, attempt to retrieve from disk
	enc := rawdb.ReadZKTrieNode(db.diskdb, hash)
	if len(enc) != 0 {
		if db.cleans != nil {
			db.cleans.Set(key, enc)
			memcacheCleanMissMeter.Mark(1)
			memcacheCleanWriteMeter.Mark(int64(len(enc)))
		}
		return enc, nil
	}
	return nil, errors.New("not found")
}

// Cap iteratively flushes old but still referenced trie nodes until the total
// memory usage goes below the given threshold.
//
// Note, this method is a non-synchronized mutator. It is unsafe to call this
// concurrently with other mutators.
func (db *Database) Cap(size common.StorageSize) error {
	//TODO: implement it when database is refactor
	return db.Commit(common.Hash{}, true, nil)
}

func (db *Database) Has(key []byte) (bool, error) {
	val, err := db.Get(key)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}
