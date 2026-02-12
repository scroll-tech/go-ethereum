package rawdb

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestEncryptionKeyStorage(t *testing.T) {
	db := NewMemoryDatabase()

	// Test initial state - no keys synced
	assert.Equal(t, uint64(0), ReadHighestSyncedEncryptionKeyId(db))
	assert.Nil(t, ReadEncryptionKey(db, 0))

	// Write first key
	key0 := types.EncryptionKey{
		KeyId:    0,
		MsgIndex: 0,
		Key:      []byte{0x02, 0x03, 0x04}, // Mock 3-byte key for testing
	}
	WriteEncryptionKey(db, key0)

	// Verify key0 was written
	assert.Equal(t, uint64(0), ReadHighestSyncedEncryptionKeyId(db))
	retrieved := ReadEncryptionKey(db, 0)
	assert.NotNil(t, retrieved)
	assert.Equal(t, key0.KeyId, retrieved.KeyId)
	assert.Equal(t, key0.MsgIndex, retrieved.MsgIndex)
	assert.Equal(t, key0.Key, retrieved.Key)

	// Write second key
	key1 := types.EncryptionKey{
		KeyId:    1,
		MsgIndex: 100,
		Key:      []byte{0x03, 0x05, 0x07}, // Different key
	}
	WriteEncryptionKey(db, key1)

	// Verify highest synced ID updated
	assert.Equal(t, uint64(1), ReadHighestSyncedEncryptionKeyId(db))

	// Verify both keys can be read
	retrieved0 := ReadEncryptionKey(db, 0)
	assert.NotNil(t, retrieved0)
	assert.Equal(t, key0.KeyId, retrieved0.KeyId)
	assert.Equal(t, key0.MsgIndex, retrieved0.MsgIndex)

	retrieved1 := ReadEncryptionKey(db, 1)
	assert.NotNil(t, retrieved1)
	assert.Equal(t, key1.KeyId, retrieved1.KeyId)
	assert.Equal(t, key1.MsgIndex, retrieved1.MsgIndex)

	// Verify non-existent key returns nil
	assert.Nil(t, ReadEncryptionKey(db, 999))
}

func TestEncryptionKeyBatchWrite(t *testing.T) {
	db := NewMemoryDatabase()

	keys := []types.EncryptionKey{
		{
			KeyId:    0,
			MsgIndex: 0,
			Key:      []byte{0x02, 0x03, 0x04},
		},
		{
			KeyId:    1,
			MsgIndex: 50,
			Key:      []byte{0x03, 0x05, 0x07},
		},
		{
			KeyId:    2,
			MsgIndex: 200,
			Key:      []byte{0x04, 0x06, 0x08},
		},
	}

	WriteEncryptionKeys(db, keys)

	// Verify highest synced ID
	assert.Equal(t, uint64(2), ReadHighestSyncedEncryptionKeyId(db))

	// Verify all keys can be read
	for _, expected := range keys {
		retrieved := ReadEncryptionKey(db, expected.KeyId)
		assert.NotNil(t, retrieved)
		assert.Equal(t, expected.KeyId, retrieved.KeyId)
		assert.Equal(t, expected.MsgIndex, retrieved.MsgIndex)
		assert.Equal(t, expected.Key, retrieved.Key)
	}
}

func TestEncryptionKeyRLP(t *testing.T) {
	db := NewMemoryDatabase()

	key := types.EncryptionKey{
		KeyId:    5,
		MsgIndex: 1234,
		Key:      make([]byte, 33), // Typical compressed ECIES key size
	}
	// Fill with test data
	for i := range key.Key {
		key.Key[i] = byte(i)
	}

	WriteEncryptionKey(db, key)

	// Test RLP reading
	rlpData := ReadEncryptionKeyRLP(db, 5)
	assert.NotNil(t, rlpData)
	assert.NotEmpty(t, rlpData)

	// Non-existent key should return nil RLP
	assert.Nil(t, ReadEncryptionKeyRLP(db, 999))
}

func TestHighestSyncedEncryptionKeyId(t *testing.T) {
	db := NewMemoryDatabase()

	// Initial state
	assert.Equal(t, uint64(0), ReadHighestSyncedEncryptionKeyId(db))

	// Write directly (not through WriteEncryptionKey)
	WriteHighestSyncedEncryptionKeyId(db, 42)
	assert.Equal(t, uint64(42), ReadHighestSyncedEncryptionKeyId(db))

	// Update to higher value
	WriteHighestSyncedEncryptionKeyId(db, 100)
	assert.Equal(t, uint64(100), ReadHighestSyncedEncryptionKeyId(db))
}
