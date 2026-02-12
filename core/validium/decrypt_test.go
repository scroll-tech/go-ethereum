package validium

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"

	ecies "github.com/scroll-tech/ecies-go/v2"
)

func TestDepositERC20NotEncrypted(t *testing.T) {
	data := common.Hex2Bytes("8ef1332e000000000000000000000000f1af3b23de0a5ca3cab7261cb0061c0d779a5c7b00000000000000000000000033b60d5dd260d453cac3782b0bdc01ce84672142000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e9cd600000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e48431f5c1000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb4800000000000000000000000006efdbff2a14a7c8e15944d1f4a48f9f95f663a4000000000000000000000000f4e147db314947fc1275a8cbb6cde48c510cd8cf0000000000000000000000003a6a724595184dda4be69db1ce726f2ac3d66b870000000000000000000000000000000000000000000000000000000783a8a06400000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	newData, err := DecryptTxData(data)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, newData) {
		t.Errorf("expected %s, got %s", data, newData)
	}
}

func TestDecryptRelayMessage(t *testing.T) {
	defer resetKeyState()

	testKey, _ := ecies.NewPrivateKeyFromHex("32d11e92cdb5ed666faa2ec639a03c63dfd730b6ae41f1306a59f1d1e9201b59")
	SetSequencerKey(testKey)

	// Address encrypted using eciesjs.
	target := common.HexToAddress("127b15f37acbeaa4188a3388689445ae892787bc")
	targetEncrypted := common.Hex2Bytes("047228ab6bed95b93ceef9a64da739375952d1c53cf3dd9a20b76821d74f393dd1a8cd854afd67d8ad5dc0672314c5c059d7b3ba1479fe1efc522b3049351969da362cabc57d7a99446e4341ad6fce79a38b6ed3cc1bb6b51858660b3f365cf45ceea21a1ea08aa65a4107d8cc27c5462153321e27")

	// Assume we see a "relayMessageEncrypted(...)" call with no payload.
	encryptedArgs := relayMessageEncryptedArgs{
		Sender:       common.Address{},
		Target:       targetEncrypted,
		Value:        big.NewInt(0),
		MessageNonce: big.NewInt(0),
		Message:      []byte{},
	}

	encryptedCall, err := encryptedArgs.packFunctionCall()
	if err != nil {
		t.Fatal(err)
	}

	// Process call, decrypt payload.
	decryptedCall, err := DecryptTxData(encryptedCall)
	if err != nil {
		t.Fatal(err)
	}

	var decryptedArgs relayMessageArgs
	err = unpackArguments("relayMessage", &decryptedArgs, decryptedCall[4:])
	if err != nil {
		t.Fatal(err)
	}

	if decryptedArgs.Target != target {
		t.Errorf("expected target %s, got %s", target.Hex(), decryptedArgs.Target.Hex())
	}
}

func TestDecryptDepositERC20Message(t *testing.T) {
	defer resetKeyState()

	testKey, _ := ecies.NewPrivateKeyFromHex("32d11e92cdb5ed666faa2ec639a03c63dfd730b6ae41f1306a59f1d1e9201b59")
	SetSequencerKey(testKey)

	// Address encrypted using eciesjs.
	target := common.HexToAddress("127b15f37acbeaa4188a3388689445ae892787bc")
	targetEncrypted := common.Hex2Bytes("047228ab6bed95b93ceef9a64da739375952d1c53cf3dd9a20b76821d74f393dd1a8cd854afd67d8ad5dc0672314c5c059d7b3ba1479fe1efc522b3049351969da362cabc57d7a99446e4341ad6fce79a38b6ed3cc1bb6b51858660b3f365cf45ceea21a1ea08aa65a4107d8cc27c5462153321e27")

	// Assume we see a "relayMessage(...)" call with "finalizeDepositERC20Encrypted(...)" payload.
	encryptedInnerArgs := finalizeDepositERC20EncryptedArgs{
		Token:   common.Address{},
		L2Token: common.Address{},
		From:    common.Address{},
		To:      targetEncrypted,
		Amount:  big.NewInt(0),
		L2Data:  []byte{},
	}

	encryptedInnerCall, err := encryptedInnerArgs.packFunctionCall()
	if err != nil {
		t.Fatal(err)
	}

	encryptedOuterArgs := relayMessageArgs{
		Sender:       common.Address{},
		Target:       common.Address{}, // normally this would be the erc20 gateway address
		Value:        big.NewInt(0),
		MessageNonce: big.NewInt(0),
		Message:      encryptedInnerCall,
	}

	encryptedOuterCall, err := encryptedOuterArgs.packFunctionCall()
	if err != nil {
		t.Fatal(err)
	}

	// Process call, decrypt payload.
	decryptedOuterCall, err := DecryptTxData(encryptedOuterCall)
	if err != nil {
		t.Fatal(err)
	}

	var decryptedOuterArgs relayMessageArgs
	err = unpackArguments("relayMessage", &decryptedOuterArgs, decryptedOuterCall[4:])
	if err != nil {
		t.Fatal(err)
	}

	var decryptedInnerArgs finalizeDepositERC20Args
	err = unpackArguments("finalizeDepositERC20", &decryptedInnerArgs, decryptedOuterArgs.Message[4:])
	if err != nil {
		t.Fatal(err)
	}

	if decryptedInnerArgs.To != target {
		t.Errorf("expected target %s, got %s", target.Hex(), decryptedOuterArgs.Target.Hex())
	}
}

// Tests for key rotation support

// resetKeyState resets global key state to avoid test pollution
func resetKeyState() {
	lock.Lock()
	defer lock.Unlock()
	SequencerKey = nil
	sequencerKeys = nil
	keyActivations = nil
	configuredKeys = nil
	keysInitialized = false
}

func TestGetKeyForMessage_SingleKey(t *testing.T) {
	defer resetKeyState()

	// Setup: single key starting at msgIndex 0
	key0, _ := ecies.GenerateKey()
	lock.Lock()
	sequencerKeys = map[uint64]*ecies.PrivateKey{0: key0}
	keyActivations = []keyActivation{{msgIndex: 0, keyId: 0}}
	keysInitialized = true
	lock.Unlock()

	// Test: all messages should use key 0
	for queueIndex := uint64(0); queueIndex < 100; queueIndex++ {
		key, err := GetKeyForMessage(queueIndex)
		if err != nil {
			t.Fatalf("GetKeyForMessage(%d) error: %v", queueIndex, err)
		}
		if key != key0 {
			t.Errorf("GetKeyForMessage(%d) returned wrong key", queueIndex)
		}
	}
}

func TestGetKeyForMessage_MultipleKeys(t *testing.T) {
	defer resetKeyState()

	// Setup: key rotation at msgIndex 0, 100, 200
	key0, _ := ecies.GenerateKey()
	key1, _ := ecies.GenerateKey()
	key2, _ := ecies.GenerateKey()

	lock.Lock()
	sequencerKeys = map[uint64]*ecies.PrivateKey{
		0: key0,
		1: key1,
		2: key2,
	}
	keyActivations = []keyActivation{
		{msgIndex: 0, keyId: 0},
		{msgIndex: 100, keyId: 1},
		{msgIndex: 200, keyId: 2},
	}
	keysInitialized = true
	lock.Unlock()

	// Test cases
	testCases := []struct {
		queueIndex    uint64
		expectedKeyId uint64
	}{
		{0, 0},
		{50, 0},
		{99, 0},
		{100, 1}, // Boundary: exactly at key1 activation
		{150, 1},
		{199, 1},
		{200, 2}, // Boundary: exactly at key2 activation
		{500, 2},
	}

	for _, tc := range testCases {
		key, err := GetKeyForMessage(tc.queueIndex)
		if err != nil {
			t.Fatalf("GetKeyForMessage(%d) error: %v", tc.queueIndex, err)
		}
		expectedKey := sequencerKeys[tc.expectedKeyId]
		if key != expectedKey {
			t.Errorf("queueIndex=%d should use keyId=%d", tc.queueIndex, tc.expectedKeyId)
		}
	}
}

func TestGetKeyForMessage_LegacyMode(t *testing.T) {
	defer resetKeyState()

	// Setup: keys not initialized, but legacy key set
	legacyKey, _ := ecies.GenerateKey()
	lock.Lock()
	SequencerKey = legacyKey
	keysInitialized = false
	lock.Unlock()

	// Test: should fall back to legacy key
	key, err := GetKeyForMessage(100)
	if err != nil {
		t.Fatalf("GetKeyForMessage error: %v", err)
	}
	if key != legacyKey {
		t.Errorf("should return legacy key")
	}
}

func TestGetNextKeyRotationIndex_SingleKey(t *testing.T) {
	defer resetKeyState()

	// Setup: single key starting at msgIndex 0
	key0, _ := ecies.GenerateKey()
	lock.Lock()
	sequencerKeys = map[uint64]*ecies.PrivateKey{0: key0}
	keyActivations = []keyActivation{{msgIndex: 0, keyId: 0}}
	keysInitialized = true
	lock.Unlock()

	// Test: no rotation should occur
	nextRotation := GetNextKeyRotationIndex(0)
	if nextRotation != ^uint64(0) {
		t.Errorf("expected max uint64, got %d", nextRotation)
	}

	nextRotation = GetNextKeyRotationIndex(1000)
	if nextRotation != ^uint64(0) {
		t.Errorf("expected max uint64, got %d", nextRotation)
	}
}

func TestGetNextKeyRotationIndex_MultipleKeys(t *testing.T) {
	defer resetKeyState()

	// Setup: key rotation at msgIndex 0, 100, 200
	key0, _ := ecies.GenerateKey()
	key1, _ := ecies.GenerateKey()
	key2, _ := ecies.GenerateKey()

	lock.Lock()
	sequencerKeys = map[uint64]*ecies.PrivateKey{
		0: key0,
		1: key1,
		2: key2,
	}
	keyActivations = []keyActivation{
		{msgIndex: 0, keyId: 0},
		{msgIndex: 100, keyId: 1},
		{msgIndex: 200, keyId: 2},
	}
	keysInitialized = true
	lock.Unlock()

	// Test cases
	testCases := []struct {
		queueIndex           uint64
		expectedNextRotation uint64
	}{
		{0, 100},          // Starting at 0, next rotation at 100
		{50, 100},         // In middle of key0 range
		{99, 100},         // Just before rotation
		{100, 200},        // At rotation boundary, next rotation at 200
		{150, 200},        // In middle of key1 range
		{199, 200},        // Just before second rotation
		{200, ^uint64(0)}, // At last key, no more rotations
		{500, ^uint64(0)}, // Beyond last rotation
	}

	for _, tc := range testCases {
		nextRotation := GetNextKeyRotationIndex(tc.queueIndex)
		if nextRotation != tc.expectedNextRotation {
			t.Errorf("queueIndex=%d: expected nextRotation=%d, got %d",
				tc.queueIndex, tc.expectedNextRotation, nextRotation)
		}
	}
}

func TestGetNextKeyRotationIndex_LegacyMode(t *testing.T) {
	defer resetKeyState()

	// Setup: keys not initialized
	lock.Lock()
	keysInitialized = false
	lock.Unlock()

	// Test: should return max uint64 (no rotations in legacy mode)
	nextRotation := GetNextKeyRotationIndex(100)
	if nextRotation != ^uint64(0) {
		t.Errorf("expected max uint64 in legacy mode, got %d", nextRotation)
	}
}

func TestGetKeyIdForMessage(t *testing.T) {
	defer resetKeyState()

	// Setup: key rotation at msgIndex 0, 100, 200
	key0, _ := ecies.GenerateKey()
	key1, _ := ecies.GenerateKey()
	key2, _ := ecies.GenerateKey()

	lock.Lock()
	sequencerKeys = map[uint64]*ecies.PrivateKey{
		0: key0,
		1: key1,
		2: key2,
	}
	keyActivations = []keyActivation{
		{msgIndex: 0, keyId: 0},
		{msgIndex: 100, keyId: 1},
		{msgIndex: 200, keyId: 2},
	}
	keysInitialized = true
	lock.Unlock()

	// Test cases
	testCases := []struct {
		queueIndex    uint64
		expectedKeyId uint64
	}{
		{0, 0},
		{50, 0},
		{99, 0},
		{100, 1},
		{150, 1},
		{199, 1},
		{200, 2},
		{500, 2},
	}

	for _, tc := range testCases {
		keyId, err := GetKeyIdForMessage(tc.queueIndex)
		if err != nil {
			t.Fatalf("GetKeyIdForMessage(%d) error: %v", tc.queueIndex, err)
		}
		if keyId != tc.expectedKeyId {
			t.Errorf("queueIndex=%d: expected keyId=%d, got %d",
				tc.queueIndex, tc.expectedKeyId, keyId)
		}
	}
}
