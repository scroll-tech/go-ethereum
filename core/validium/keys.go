package validium

import (
	"fmt"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"

	ecies "github.com/scroll-tech/ecies-go/v2"
)

var (
	// Private keys storage for decryption
	// Thread safety: All access to sequencerKeys is protected by lock.
	// - Write operations (SetSequencerKeys) acquire write lock
	// - Read operations (GetKeyByPubKey) acquire read lock
	// - Keys are loaded once at startup and then are read-only during operation
	// - Keys are indexed by compressed public key hex (33 bytes compressed → hex string)
	sequencerKeys map[string]*ecies.PrivateKey // Map: compressed pubkey hex → private key
	lock          sync.RWMutex
)

// SetSequencerKeys sets sequencer keys for decryption.
// This should be called once during node initialization with all configured keys.
// Keys are indexed internally by their compressed public key hex for O(1) lookup during decryption.
func SetSequencerKeys(keys []*ecies.PrivateKey) {
	lock.Lock()
	defer lock.Unlock()

	sequencerKeys = make(map[string]*ecies.PrivateKey)
	for i, key := range keys {
		if key == nil {
			log.Warn("Skipping nil key in SetSequencerKeys", "index", i)
			continue
		}
		// Use compressed public key (33 bytes) as the map key
		compressedPubKey := key.PublicKey.Bytes(true)
		pubKeyHex := common.Bytes2Hex(compressedPubKey)
		sequencerKeys[pubKeyHex] = key
		log.Info("Sequencer key configured", "index", i, "pubKeyHex", pubKeyHex[:16]+"...")
	}
	log.Info("Sequencer keys configured", "count", len(sequencerKeys))
}

// GetKeyByPubKey retrieves the private key corresponding to the given compressed public key.
// Returns an error if no key is found for the given public key.
//
// This function is thread-safe and can be called concurrently during block production.
func GetKeyByPubKey(compressedPubKey []byte) (*ecies.PrivateKey, error) {
	if compressedPubKey == nil {
		return nil, fmt.Errorf("compressedPubKey cannot be nil")
	}

	lock.RLock()
	defer lock.RUnlock()

	pubKeyHex := common.Bytes2Hex(compressedPubKey)
	key, exists := sequencerKeys[pubKeyHex]
	if !exists {
		return nil, fmt.Errorf("no private key found for public key %s", pubKeyHex[:16]+"...")
	}

	return key, nil
}
