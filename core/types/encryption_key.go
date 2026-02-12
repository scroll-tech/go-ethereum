package types

// EncryptionKey represents a sequencer encryption key registration from the ScrollChainValidium contract.
// The sequencer uses this to map L1 message queue indices to the appropriate ECIES decryption key.
type EncryptionKey struct {
	KeyId    uint64 // Key identifier matching keyId from NewEncryptionKey event
	MsgIndex uint64 // L1 message queue index when this key becomes active
	Key      []byte // 33-byte compressed ECIES public key
}
