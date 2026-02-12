package validium

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"

	ecies "github.com/scroll-tech/ecies-go/v2"
)

const (
	relayMessage                  = "relayMessage"
	relayMessageEncrypted         = "relayMessageEncrypted"
	finalizeDepositERC20          = "finalizeDepositERC20"
	finalizeDepositERC20Encrypted = "finalizeDepositERC20Encrypted"
)

var (
	BridgeABI abi.ABI

	// Legacy single key (deprecated, kept for backwards compatibility)
	// Thread safety: SetSequencerKey should only be called during node initialization,
	// before any concurrent access via GetKeyForMessage. Once set, it is read-only.
	SequencerKey *ecies.PrivateKey

	// Multi-key storage for key rotation support
	// Thread safety: All access to the fields below is protected by lock.
	// - Write operations (SetSequencerKeys, InitializeKeys) acquire write lock
	// - Read operations (GetKeyForMessage) acquire read lock
	// - Keys are loaded once at startup and then are read-only during operation
	sequencerKeys    map[uint64]*ecies.PrivateKey // Map from keyId → private key
	keyActivations   []keyActivation              // Sorted list of (msgIndex, keyId) for binary search
	configuredKeys   map[uint64]*ecies.PrivateKey // Keys from configuration
	lock             sync.RWMutex
	keysInitialized  bool
)

type keyActivation struct {
	msgIndex uint64
	keyId    uint64
}

func SetSequencerKey(key *ecies.PrivateKey) {
	lock.Lock()
	defer lock.Unlock()
	SequencerKey = key
	log.Warn("Sequencer key set (legacy mode)", "publicKey", key.PublicKey.Hex(true))
}

// SetSequencerKeys sets multiple sequencer keys for key rotation support.
// This should be called once during node initialization with all configured keys.
func SetSequencerKeys(keys map[uint64]*ecies.PrivateKey) {
	lock.Lock()
	defer lock.Unlock()
	configuredKeys = keys
	log.Info("Sequencer keys configured", "count", len(keys))
}

// GetKeyIdForMessage returns the keyId that should be used for the given queue index.
// Returns 0 in legacy mode or bootstrap mode.
//
// Thread-safe: Uses read lock for concurrent access during block production.
func GetKeyIdForMessage(queueIndex uint64) (uint64, error) {
	lock.RLock()
	defer lock.RUnlock()

	if !keysInitialized {
		// Legacy mode: always use keyId 0
		return 0, nil
	}

	if len(keyActivations) == 0 {
		// Bootstrap mode: use keyId 0
		return 0, nil
	}

	// Binary search: find the largest msgIndex <= queueIndex
	idx := sort.Search(len(keyActivations), func(i int) bool {
		return keyActivations[i].msgIndex > queueIndex
	})

	if idx > 0 {
		idx--
	}

	// Edge case: message before first key
	if keyActivations[idx].msgIndex > queueIndex {
		return 0, fmt.Errorf("no key available for queue index %d (first key starts at %d)", queueIndex, keyActivations[idx].msgIndex)
	}

	return keyActivations[idx].keyId, nil
}

// GetNextKeyRotationIndex returns the msgIndex at which the next key rotation occurs
// after the given queue index. Returns math.MaxUint64 if there is no subsequent rotation.
//
// This is used to efficiently find key rotation boundaries when collecting L1 messages.
// Thread-safe: Uses read lock for concurrent access during block production.
func GetNextKeyRotationIndex(queueIndex uint64) uint64 {
	lock.RLock()
	defer lock.RUnlock()

	if !keysInitialized || len(keyActivations) == 0 {
		// Legacy/bootstrap mode: no key rotations
		return ^uint64(0) // math.MaxUint64
	}

	// Binary search: find the first msgIndex > queueIndex
	idx := sort.Search(len(keyActivations), func(i int) bool {
		return keyActivations[i].msgIndex > queueIndex
	})

	// If we found a rotation boundary, return it
	if idx < len(keyActivations) {
		return keyActivations[idx].msgIndex
	}

	// No more rotations after this queue index
	return ^uint64(0) // math.MaxUint64
}

// GetKeyForMessage returns the decryption key for the given queue index.
// Uses binary search to find the key with the highest msgIndex <= queueIndex.
//
// Algorithm matches ScrollChainValidium.sol _getEncryptionKey (lines 432-445):
//   1. Binary search for largest msgIndex <= queueIndex
//   2. Return corresponding private key
//   3. This ensures sequencer uses same key as prover for validation
//
// Thread-safe: Uses read lock for concurrent access during block production.
func GetKeyForMessage(queueIndex uint64) (*ecies.PrivateKey, error) {
	lock.RLock()
	defer lock.RUnlock()

	if !keysInitialized {
		// Legacy mode: use single key
		if SequencerKey != nil {
			return SequencerKey, nil
		}
		return nil, fmt.Errorf("encryption keys not initialized")
	}

	if len(keyActivations) == 0 {
		return nil, fmt.Errorf("no encryption keys available")
	}

	// Binary search: find the largest msgIndex <= queueIndex
	idx := sort.Search(len(keyActivations), func(i int) bool {
		return keyActivations[i].msgIndex > queueIndex
	})

	if idx > 0 {
		idx--
	}

	// Edge case: message before first key
	if keyActivations[idx].msgIndex > queueIndex {
		return nil, fmt.Errorf("no key available for queue index %d (first key starts at %d)", queueIndex, keyActivations[idx].msgIndex)
	}

	keyId := keyActivations[idx].keyId
	key := sequencerKeys[keyId]
	if key == nil {
		return nil, fmt.Errorf("key not found for keyId %d", keyId)
	}

	return key, nil
}

// InitializeKeys loads encryption keys from database and initializes the key selection system.
// This should be called after the database is opened and after SetSequencerKeys.
//
// Bootstrap mode: If no keys are synced in the database yet, initializes with keyId 0
// starting at msgIndex 0. This allows fresh nodes to start without waiting for key sync.
//
// Normal mode: Loads all synced keys from database and validates that corresponding
// private keys are configured. Returns error if any synced key lacks a private key.
//
// Thread-safety: Acquires write lock during initialization. Must be called before
// any concurrent access via GetKeyForMessage.
func InitializeKeys(db ethdb.Database) error {
	lock.Lock()
	defer lock.Unlock()

	if configuredKeys == nil || len(configuredKeys) == 0 {
		log.Warn("No configured keys, using legacy single-key mode")
		keysInitialized = false
		return nil
	}

	highestKeyId := rawdb.ReadHighestSyncedEncryptionKeyId(db)

	// Bootstrap mode: no keys synced yet, use keyId 0 from msgIndex 0
	if highestKeyId == 0 {
		key0, exists := configuredKeys[0]
		if !exists {
			return fmt.Errorf("bootstrap mode requires keyId 0 to be configured")
		}

		sequencerKeys = make(map[uint64]*ecies.PrivateKey)
		sequencerKeys[0] = key0
		keyActivations = []keyActivation{{msgIndex: 0, keyId: 0}}
		keysInitialized = true

		log.Info("Encryption keys initialized (bootstrap mode)", "keyId", 0, "startMsgIndex", 0)
		return nil
	}

	// Load all keys from database
	sequencerKeys = make(map[uint64]*ecies.PrivateKey)
	keyActivations = []keyActivation{}

	for keyId := uint64(0); keyId <= highestKeyId; keyId++ {
		encKey := rawdb.ReadEncryptionKey(db, keyId)
		if encKey == nil {
			return fmt.Errorf("missing encryption key in database: keyId %d (expected continuous sequence 0..%d)", keyId, highestKeyId)
		}

		privateKey, exists := configuredKeys[keyId]
		if !exists {
			return fmt.Errorf("synced key keyId=%d has no corresponding private key in configuration", keyId)
		}

		sequencerKeys[keyId] = privateKey
		keyActivations = append(keyActivations, keyActivation{
			msgIndex: encKey.MsgIndex,
			keyId:    encKey.KeyId,
		})

		log.Info("Loaded encryption key from database", "keyId", keyId, "msgIndex", encKey.MsgIndex)
	}

	// Sort by msgIndex for binary search
	sort.Slice(keyActivations, func(i, j int) bool {
		return keyActivations[i].msgIndex < keyActivations[j].msgIndex
	})

	keysInitialized = true
	log.Info("Encryption keys initialized from database", "count", len(sequencerKeys), "highestKeyId", highestKeyId)
	return nil
}

func init() {
	abiJSON := `[{
		"name": "finalizeDepositERC20",
		"type": "function",
		"inputs": [
			{ "name": "token", "type": "address" },
			{ "name": "l2Token", "type": "address" },
			{ "name": "from", "type": "address" },
			{ "name": "to", "type": "address" },
			{ "name": "amount", "type": "uint256" },
			{ "name": "l2Data", "type": "bytes" }
		],
		"outputs": []
	}, {
		"name": "finalizeDepositERC20Encrypted",
		"type": "function",
		"inputs": [
			{ "name": "token", "type": "address" },
			{ "name": "l2Token", "type": "address" },
			{ "name": "from", "type": "address" },
			{ "name": "to", "type": "bytes" },
			{ "name": "amount", "type": "uint256" },
			{ "name": "l2Data", "type": "bytes" }
		],
		"outputs": []
	}, {
		"name": "relayMessage",
		"type": "function",
		"inputs": [
			{ "name": "sender", "type": "address" },
			{ "name": "target", "type": "address" },
			{ "name": "value", "type": "uint256" },
			{ "name": "messageNonce", "type": "uint256" },
			{ "name": "message", "type": "bytes" }
		],
		"outputs": []
	}, {
		"name": "relayMessageEncrypted",
		"type": "function",
		"inputs": [
			{ "name": "sender", "type": "address" },
			{ "name": "target", "type": "bytes" },
			{ "name": "value", "type": "uint256" },
			{ "name": "messageNonce", "type": "uint256" },
			{ "name": "message", "type": "bytes" }
		],
		"outputs": []
	}]`

	BridgeABI, _ = abi.JSON(bytes.NewReader([]byte(abiJSON)))
}

type relayMessageArgs struct {
	Sender       common.Address
	Target       common.Address
	Value        *big.Int
	MessageNonce *big.Int
	Message      []byte
}

type relayMessageEncryptedArgs struct {
	Sender       common.Address
	Target       []byte // encrypted address
	Value        *big.Int
	MessageNonce *big.Int
	Message      []byte
}

type finalizeDepositERC20Args struct {
	Token   common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	L2Data  []byte
}

type finalizeDepositERC20EncryptedArgs struct {
	Token   common.Address
	L2Token common.Address
	From    common.Address
	To      []byte // encrypted address
	Amount  *big.Int
	L2Data  []byte
}

// unpackArguments unpacks the payload into the provided struct.
// The payload should not include the function selector (first 4 bytes).
func unpackArguments(methodName string, target interface{}, payload []byte) error {
	method, ok := BridgeABI.Methods[methodName]
	if !ok {
		return fmt.Errorf("method %q not found in ABI", methodName)
	}
	values, err := method.Inputs.Unpack(payload)
	if err != nil {
		return fmt.Errorf("unpack error: %w", err)
	}
	return method.Inputs.Copy(target, values)
}

// packFunctionCall packs the struct into abi-encoded function call data.
// The resulting data includes the function selector (first 4 bytes).
func packFunctionCall(methodName string, args ...interface{}) ([]byte, error) {
	return BridgeABI.Pack(methodName, args...)
}

func (v *relayMessageArgs) packFunctionCall() ([]byte, error) {
	return packFunctionCall(relayMessage, v.Sender, v.Target, v.Value, v.MessageNonce, v.Message)
}

func (v *relayMessageEncryptedArgs) packFunctionCall() ([]byte, error) {
	return packFunctionCall(relayMessageEncrypted, v.Sender, v.Target, v.Value, v.MessageNonce, v.Message)
}

func (v *finalizeDepositERC20Args) packFunctionCall() ([]byte, error) {
	return packFunctionCall(finalizeDepositERC20, v.Token, v.L2Token, v.From, v.To, v.Amount, v.L2Data)
}

func (v *finalizeDepositERC20EncryptedArgs) packFunctionCall() ([]byte, error) {
	return packFunctionCall(finalizeDepositERC20Encrypted, v.Token, v.L2Token, v.From, v.To, v.Amount, v.L2Data)
}

// decrypt converts a finalizeDepositERC20EncryptedArgs to finalizeDepositERC20Args by decrypting the To field.
func (v *finalizeDepositERC20EncryptedArgs) decrypt() (*finalizeDepositERC20Args, error) {
	decrypted, err := decryptAddress(v.To)
	if err != nil {
		return nil, err
	}

	return &finalizeDepositERC20Args{
		Token:   v.Token,
		L2Token: v.L2Token,
		From:    v.From,
		To:      decrypted,
		Amount:  v.Amount,
		L2Data:  v.L2Data,
	}, nil
}

// decrypt converts a relayMessageEncryptedArgs to relayMessageArgs by decrypting the Target field.
func (v relayMessageEncryptedArgs) decrypt() (relayMessageArgs, error) {
	decrypted, err := decryptAddress(v.Target)
	if err != nil {
		return relayMessageArgs{}, err
	}

	return relayMessageArgs{
		Sender:       v.Sender,
		Target:       decrypted,
		Value:        v.Value,
		MessageNonce: v.MessageNonce,
		Message:      v.Message,
	}, nil
}

// decryptAddress decrypts the given encrypted address using the configured sequencer key.
// This is the legacy function that uses the single SequencerKey.
func decryptAddress(data []byte) (common.Address, error) {
	raw, err := ecies.Decrypt(SequencerKey, data)
	if err != nil {
		log.Warn("failed to decrypt address", "data", common.Bytes2Hex(data), "error", err)
		return common.Address{}, errors.New("decryption failed")
	}

	if len(raw) != common.AddressLength {
		log.Warn("decrypted data is not address", "data", common.Bytes2Hex(data), "plaintext", common.Bytes2Hex(raw))
		return common.Address{}, errors.New("decryption failed")
	}

	address := common.BytesToAddress(raw)
	log.Info("Successfully decrypted address", "address", address)
	return address, nil
}

// decryptAddressWithQueueIndex decrypts the given encrypted address using the appropriate key for the queue index.
func decryptAddressWithQueueIndex(data []byte, queueIndex uint64) (common.Address, error) {
	key, err := GetKeyForMessage(queueIndex)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get key for queue index %d: %w", queueIndex, err)
	}

	raw, err := ecies.Decrypt(key, data)
	if err != nil {
		log.Warn("failed to decrypt address", "queueIndex", queueIndex, "data", common.Bytes2Hex(data), "error", err)
		return common.Address{}, errors.New("decryption failed")
	}

	if len(raw) != common.AddressLength {
		log.Warn("decrypted data is not address", "queueIndex", queueIndex, "data", common.Bytes2Hex(data), "plaintext", common.Bytes2Hex(raw))
		return common.Address{}, errors.New("decryption failed")
	}

	address := common.BytesToAddress(raw)
	log.Info("Successfully decrypted address", "queueIndex", queueIndex, "address", address)
	return address, nil
}

// decryptWithQueueIndex converts a finalizeDepositERC20EncryptedArgs to finalizeDepositERC20Args by decrypting the To field using queue-specific key.
func (v *finalizeDepositERC20EncryptedArgs) decryptWithQueueIndex(queueIndex uint64) (*finalizeDepositERC20Args, error) {
	decrypted, err := decryptAddressWithQueueIndex(v.To, queueIndex)
	if err != nil {
		return nil, err
	}

	return &finalizeDepositERC20Args{
		Token:   v.Token,
		L2Token: v.L2Token,
		From:    v.From,
		To:      decrypted,
		Amount:  v.Amount,
		L2Data:  v.L2Data,
	}, nil
}

// decryptWithQueueIndex converts a relayMessageEncryptedArgs to relayMessageArgs by decrypting the Target field using queue-specific key.
func (v relayMessageEncryptedArgs) decryptWithQueueIndex(queueIndex uint64) (relayMessageArgs, error) {
	decrypted, err := decryptAddressWithQueueIndex(v.Target, queueIndex)
	if err != nil {
		return relayMessageArgs{}, err
	}

	return relayMessageArgs{
		Sender:       v.Sender,
		Target:       decrypted,
		Value:        v.Value,
		MessageNonce: v.MessageNonce,
		Message:      v.Message,
	}, nil
}

// decryptInnerCall decrypts the inner call of a relayMessageArgs if it is encrypted.
func (v *relayMessageArgs) decryptInnerCall() error {
	if len(v.Message) < 4 {
		return nil
	}

	selector := v.Message[:4]
	payload := v.Message[4:]

	switch {
	// payload is encrypted, we parse and decrypt it
	case bytes.Equal(selector, BridgeABI.Methods[finalizeDepositERC20Encrypted].ID):
		var encrypted finalizeDepositERC20EncryptedArgs
		if err := unpackArguments(finalizeDepositERC20Encrypted, &encrypted, payload); err != nil {
			return err
		} else if decrypted, err := encrypted.decrypt(); err != nil {
			return err
		} else if v.Message, err = decrypted.packFunctionCall(); err != nil { // overwrite the message with the decrypted call
			return err
		} else {
			return nil
		}

	// payload is not encrypted or otherwise unknown, nothing to do
	default:
		return nil
	}
}

// DecryptTxData decrypts the transaction data, handling both encrypted and unencrypted calls.
// This is the legacy function that uses the single SequencerKey.
func DecryptTxData(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return data, nil
	}

	// decrypt outer call
	selector := data[:4]
	payload := data[4:]
	var args relayMessageArgs

	switch {
	// payload is not encrypted, simply unpack it
	case bytes.Equal(selector, BridgeABI.Methods[relayMessage].ID):
		if err := unpackArguments(relayMessage, &args, payload); err != nil {
			return nil, err
		}

	// payload is encrypted, we parse and decrypt it
	case bytes.Equal(selector, BridgeABI.Methods[relayMessageEncrypted].ID):
		var encrypted relayMessageEncryptedArgs
		if err := unpackArguments(relayMessageEncrypted, &encrypted, payload); err != nil {
			return nil, err
		} else if args, err = encrypted.decrypt(); err != nil {
			return nil, err
		}

	// payload is unknown, nothing to do
	default:
		return data, nil
	}

	// decrypt inner call
	if err := args.decryptInnerCall(); err != nil {
		return nil, err
	} else if data, err = args.packFunctionCall(); err != nil { // overwrite the message with the decrypted call
		return nil, err
	} else {
		return data, nil
	}
}

// decryptInnerCallWithQueueIndex decrypts the inner call of a relayMessageArgs if it is encrypted, using queue-specific key.
func (v *relayMessageArgs) decryptInnerCallWithQueueIndex(queueIndex uint64) error {
	if len(v.Message) < 4 {
		return nil
	}

	selector := v.Message[:4]
	payload := v.Message[4:]

	switch {
	// payload is encrypted, we parse and decrypt it
	case bytes.Equal(selector, BridgeABI.Methods[finalizeDepositERC20Encrypted].ID):
		var encrypted finalizeDepositERC20EncryptedArgs
		if err := unpackArguments(finalizeDepositERC20Encrypted, &encrypted, payload); err != nil {
			return err
		} else if decrypted, err := encrypted.decryptWithQueueIndex(queueIndex); err != nil {
			return err
		} else if v.Message, err = decrypted.packFunctionCall(); err != nil { // overwrite the message with the decrypted call
			return err
		} else {
			return nil
		}

	// payload is not encrypted or otherwise unknown, nothing to do
	default:
		return nil
	}
}

// DecryptTxDataWithIndex decrypts the transaction data using the appropriate key for the given queue index.
// This function supports key rotation by selecting the correct decryption key based on the L1 message queue index.
func DecryptTxDataWithIndex(data []byte, queueIndex uint64) ([]byte, error) {
	if len(data) < 4 {
		return data, nil
	}

	// decrypt outer call
	selector := data[:4]
	payload := data[4:]
	var args relayMessageArgs

	switch {
	// payload is not encrypted, simply unpack it
	case bytes.Equal(selector, BridgeABI.Methods[relayMessage].ID):
		if err := unpackArguments(relayMessage, &args, payload); err != nil {
			return nil, err
		}

	// payload is encrypted, we parse and decrypt it
	case bytes.Equal(selector, BridgeABI.Methods[relayMessageEncrypted].ID):
		var encrypted relayMessageEncryptedArgs
		if err := unpackArguments(relayMessageEncrypted, &encrypted, payload); err != nil {
			return nil, err
		} else if args, err = encrypted.decryptWithQueueIndex(queueIndex); err != nil {
			return nil, err
		}

	// payload is unknown, nothing to do
	default:
		return data, nil
	}

	// decrypt inner call
	if err := args.decryptInnerCallWithQueueIndex(queueIndex); err != nil {
		return nil, err
	} else if data, err = args.packFunctionCall(); err != nil { // overwrite the message with the decrypted call
		return nil, err
	} else {
		return data, nil
	}
}
