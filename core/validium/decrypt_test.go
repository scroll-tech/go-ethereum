package validium

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"

	ecies "github.com/scroll-tech/ecies-go/v2"
)

func TestDepositERC20NotEncrypted(t *testing.T) {
	defer resetKeyState()

	// Setup: configure a key (even though this data isn't encrypted)
	testKey, _ := ecies.NewPrivateKeyFromHex("32d11e92cdb5ed666faa2ec639a03c63dfd730b6ae41f1306a59f1d1e9201b59")
	SetSequencerKeys([]*ecies.PrivateKey{testKey})

	data := common.Hex2Bytes("8ef1332e000000000000000000000000f1af3b23de0a5ca3cab7261cb0061c0d779a5c7b00000000000000000000000033b60d5dd260d453cac3782b0bdc01ce84672142000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e9cd600000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e48431f5c1000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb4800000000000000000000000006efdbff2a14a7c8e15944d1f4a48f9f95f663a4000000000000000000000000f4e147db314947fc1275a8cbb6cde48c510cd8cf0000000000000000000000003a6a724595184dda4be69db1ce726f2ac3d66b870000000000000000000000000000000000000000000000000000000783a8a06400000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")

	// Non-encrypted data should pass through unchanged regardless of pubKey
	compressedPubKey := testKey.PublicKey.Bytes(true)
	newData, err := DecryptTxDataWithPubKey(data, compressedPubKey)
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
	SetSequencerKeys([]*ecies.PrivateKey{testKey})

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
	compressedPubKey := testKey.PublicKey.Bytes(true)
	decryptedCall, err := DecryptTxDataWithPubKey(encryptedCall, compressedPubKey)
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
	SetSequencerKeys([]*ecies.PrivateKey{testKey})

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
	compressedPubKey := testKey.PublicKey.Bytes(true)
	decryptedOuterCall, err := DecryptTxDataWithPubKey(encryptedOuterCall, compressedPubKey)
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
	sequencerKeys = nil
}

// TestDecryptTxDataWithPubKey_SingleKey tests decryption with a single key
func TestDecryptTxDataWithPubKey_SingleKey(t *testing.T) {
	defer resetKeyState()

	// Setup: generate test key
	testKey, _ := ecies.NewPrivateKeyFromHex("32d11e92cdb5ed666faa2ec639a03c63dfd730b6ae41f1306a59f1d1e9201b59")
	SetSequencerKeys([]*ecies.PrivateKey{testKey})

	// Address encrypted using eciesjs
	target := common.HexToAddress("127b15f37acbeaa4188a3388689445ae892787bc")
	targetEncrypted := common.Hex2Bytes("047228ab6bed95b93ceef9a64da739375952d1c53cf3dd9a20b76821d74f393dd1a8cd854afd67d8ad5dc0672314c5c059d7b3ba1479fe1efc522b3049351969da362cabc57d7a99446e4341ad6fce79a38b6ed3cc1bb6b51858660b3f365cf45ceea21a1ea08aa65a4107d8cc27c5462153321e27")

	// Create encrypted relay message call
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

	// Decrypt using the public key
	compressedPubKey := testKey.PublicKey.Bytes(true)
	decryptedCall, err := DecryptTxDataWithPubKey(encryptedCall, compressedPubKey)
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

// TestDecryptTxDataWithPubKey_MultipleKeys tests that multiple keys can be configured
func TestDecryptTxDataWithPubKey_MultipleKeys(t *testing.T) {
	defer resetKeyState()

	// Setup: generate multiple test keys
	key0, _ := ecies.GenerateKey()
	key1, _ := ecies.GenerateKey()
	key2, _ := ecies.GenerateKey()

	SetSequencerKeys([]*ecies.PrivateKey{key0, key1, key2})

	// Verify all keys can be retrieved by their public keys
	pubKey0 := key0.PublicKey.Bytes(true)
	pubKey1 := key1.PublicKey.Bytes(true)
	pubKey2 := key2.PublicKey.Bytes(true)

	if _, err := GetKeyByPubKey(pubKey0); err != nil {
		t.Errorf("failed to get key0: %v", err)
	}
	if _, err := GetKeyByPubKey(pubKey1); err != nil {
		t.Errorf("failed to get key1: %v", err)
	}
	if _, err := GetKeyByPubKey(pubKey2); err != nil {
		t.Errorf("failed to get key2: %v", err)
	}

	// Test that unknown key returns error
	unknownKey, _ := ecies.GenerateKey()
	unknownPubKey := unknownKey.PublicKey.Bytes(true)
	if _, err := GetKeyByPubKey(unknownPubKey); err == nil {
		t.Error("expected error for unknown public key")
	}
}

// TestDecryptTxDataWithPubKey_NilPubKey tests that nil pubKey returns error
func TestDecryptTxDataWithPubKey_NilPubKey(t *testing.T) {
	defer resetKeyState()

	// Setup: configure a key
	testKey, _ := ecies.NewPrivateKeyFromHex("32d11e92cdb5ed666faa2ec639a03c63dfd730b6ae41f1306a59f1d1e9201b59")
	SetSequencerKeys([]*ecies.PrivateKey{testKey})

	// Address encrypted using eciesjs
	targetEncrypted := common.Hex2Bytes("047228ab6bed95b93ceef9a64da739375952d1c53cf3dd9a20b76821d74f393dd1a8cd854afd67d8ad5dc0672314c5c059d7b3ba1479fe1efc522b3049351969da362cabc57d7a99446e4341ad6fce79a38b6ed3cc1bb6b51858660b3f365cf45ceea21a1ea08aa65a4107d8cc27c5462153321e27")

	// Create encrypted relay message call
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

	// Decrypt with nil pubKey (should return error)
	_, err = DecryptTxDataWithPubKey(encryptedCall, nil)
	if err == nil {
		t.Error("expected error when calling DecryptTxDataWithPubKey with nil pubKey")
	}
}
