package rawdb

import (
	"bytes"
	"math/big"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// WriteEncryptionKey writes an encryption key to the database.
// Keys should be written in order by keyId for consistency.
func WriteEncryptionKey(db ethdb.KeyValueWriter, key types.EncryptionKey) {
	bytes, err := rlp.EncodeToBytes(key)
	if err != nil {
		log.Crit("Failed to RLP encode encryption key", "keyId", key.KeyId, "err", err)
	}
	if err := db.Put(encryptionKeyKey(key.KeyId), bytes); err != nil {
		log.Crit("Failed to store encryption key", "keyId", key.KeyId, "err", err)
	}

	WriteHighestSyncedEncryptionKeyId(db, key.KeyId)

	log.Info("Successfully wrote encryption key to database",
		"keyId", key.KeyId,
		"msgIndex", key.MsgIndex,
		"keyLength", len(key.Key))
}

// WriteEncryptionKeys writes an array of encryption keys to the database.
// Note: pass a db of type `ethdb.Batcher` to batch writes in memory.
func WriteEncryptionKeys(db ethdb.KeyValueWriter, keys []types.EncryptionKey) {
	for _, key := range keys {
		WriteEncryptionKey(db, key)
	}
}

// ReadEncryptionKeyRLP retrieves an encryption key in its raw RLP database encoding.
func ReadEncryptionKeyRLP(db ethdb.Reader, keyId uint64) rlp.RawValue {
	data, err := db.Get(encryptionKeyKey(keyId))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load encryption key", "keyId", keyId, "err", err)
	}
	return data
}

// ReadEncryptionKey retrieves the encryption key corresponding to the keyId.
func ReadEncryptionKey(db ethdb.Reader, keyId uint64) *types.EncryptionKey {
	data := ReadEncryptionKeyRLP(db, keyId)
	if len(data) == 0 {
		return nil
	}
	key := new(types.EncryptionKey)
	if err := rlp.Decode(bytes.NewReader(data), key); err != nil {
		log.Crit("Invalid encryption key RLP", "keyId", keyId, "data", data, "err", err)
	}
	return key
}

// WriteHighestSyncedEncryptionKeyId writes the highest synced encryption key ID to the database.
func WriteHighestSyncedEncryptionKeyId(db ethdb.KeyValueWriter, keyId uint64) {
	value := big.NewInt(0).SetUint64(keyId).Bytes()

	if err := db.Put(highestSyncedEncryptionKeyId, value); err != nil {
		log.Crit("Failed to update highest synced encryption key ID", "err", err)
	}
}

// ReadHighestSyncedEncryptionKeyId retrieves the highest synced encryption key ID.
// Returns 0 if no keys have been synced yet (bootstrap mode).
func ReadHighestSyncedEncryptionKeyId(db ethdb.Reader) uint64 {
	data, err := db.Get(highestSyncedEncryptionKeyId)
	if err != nil && isNotFoundErr(err) {
		return 0
	}
	if err != nil {
		log.Crit("Failed to read highest synced encryption key ID from database", "err", err)
	}
	if len(data) == 0 {
		return 0
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("Unexpected highest synced encryption key ID in database", "number", number)
	}

	return number.Uint64()
}
