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
	testKey, _ := ecies.NewPrivateKeyFromHex("108ccd658782903954a11f6061fe7f30302feeca84a95354ed37384841956efb")
	SetSequencerKey(testKey)

	// Address encrypted using eciesjs.
	target := common.HexToAddress("127b15f37acbeaa4188a3388689445ae892787bc")
	targetEncrypted := common.Hex2Bytes("047940cffef9d00a0064cfde73bb473cb42ca9b62d2703b0009af7e9fa903095be5d914753f16b3f9451fee66cad74c075a247d64210793ea4225022581c8ac7b7d303f767fd538073f9d4207ca9e670b46dd4703d0eae18fd3e3e26921e4cd2abf57419ced65f10156a7d40642bcce03ee37425a1aa9e4244b6cd6f39944f6c4cb39081a1843602a0")

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
	testKey, _ := ecies.NewPrivateKeyFromHex("04273c070e32f8d44d49e44fb45f02e726ef8ffa857af1751d5d24bdad26021a")
	SetSequencerKey(testKey)

	// Address encrypted using eciesjs.
	target := common.HexToAddress("127b15f37acbeaa4188a3388689445ae892787bc")
	targetEncrypted := common.Hex2Bytes("041d424ad8ae52c1b9f9a5ee23d2b8a0689545fdd889645976f808df87ef4471e37ea0b77a494fa3d0b7c6e2794180a31b1326c5af1f21c40cfde321cccdfd48b2234800ffb1a93c5583b7267ce9274a193d28fae1f2f34653cc286fb8f0eb4d4f9a1234743fdd4d84544b5db30696578e8e7d1af76bc875ff6f8d3a60500322a70fd83879a8fef0d7")

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
