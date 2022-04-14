package trie

import (
	"bytes"
	"crypto/sha256"
	"errors"
)

// ErrNotFound is used by the implementations of the interface db.Storage for
// when a key is not found in the storage
var ErrNotFound = errors.New("key not found")

// Storage is the interface that defines the methods for the storage used in
// the merkletree. Examples of the interface implementation can be found at
// db/memory and db/leveldb directories.
type Storage interface {
	NewTx() (Tx, error)
	WithPrefix(prefix []byte) Storage
	Get([]byte) ([]byte, error)
	List(int) ([]KV, error)
	Close()
	Iterate(func([]byte, []byte) (bool, error)) error
}

// Tx is the interface that defines the methods for the db transaction used in
// the merkletree storage. Examples of the interface implementation can be
// found at db/memory and db/leveldb directories.
type Tx interface {
	// Get retreives the value for the given key
	// looking first in the content of the Tx, and
	// then into the content of the Storage
	Get([]byte) ([]byte, error)
	// Put sets the key & value into the Tx
	Put(k, v []byte) error
	Commit() error
	Close()
}

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
