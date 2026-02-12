package validium

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
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
)

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

// decryptAddressWithKey decrypts the given encrypted address using the specified private key.
func decryptAddressWithKey(data []byte, privateKey *ecies.PrivateKey) (common.Address, error) {
	if privateKey == nil {
		return common.Address{}, errors.New("no decryption key provided")
	}

	raw, err := ecies.Decrypt(privateKey, data)
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

// decryptAddressWithPubKey decrypts the given encrypted address using the private key
// corresponding to the given compressed public key.
func decryptAddressWithPubKey(data []byte, compressedPubKey []byte) (common.Address, error) {
	privateKey, err := GetKeyByPubKey(compressedPubKey)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get private key: %w", err)
	}

	raw, err := ecies.Decrypt(privateKey, data)
	if err != nil {
		pubKeyHex := common.Bytes2Hex(compressedPubKey)
		log.Warn("failed to decrypt address", "pubKeyHex", pubKeyHex[:16]+"...", "data", common.Bytes2Hex(data), "error", err)
		return common.Address{}, errors.New("decryption failed")
	}

	if len(raw) != common.AddressLength {
		pubKeyHex := common.Bytes2Hex(compressedPubKey)
		log.Warn("decrypted data is not address", "pubKeyHex", pubKeyHex[:16]+"...", "data", common.Bytes2Hex(data), "plaintext", common.Bytes2Hex(raw))
		return common.Address{}, errors.New("decryption failed")
	}

	address := common.BytesToAddress(raw)
	pubKeyHex := common.Bytes2Hex(compressedPubKey)
	log.Info("Successfully decrypted address", "pubKeyHex", pubKeyHex[:16]+"...", "address", address)
	return address, nil
}

// decryptWithPubKey converts a finalizeDepositERC20EncryptedArgs to finalizeDepositERC20Args by decrypting the To field using the specified public key.
func (v *finalizeDepositERC20EncryptedArgs) decryptWithPubKey(compressedPubKey []byte) (*finalizeDepositERC20Args, error) {
	decrypted, err := decryptAddressWithPubKey(v.To, compressedPubKey)
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

// decryptWithPubKey converts a relayMessageEncryptedArgs to relayMessageArgs by decrypting the Target field using the specified public key.
func (v relayMessageEncryptedArgs) decryptWithPubKey(compressedPubKey []byte) (relayMessageArgs, error) {
	decrypted, err := decryptAddressWithPubKey(v.Target, compressedPubKey)
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

// decryptInnerCallWithPubKey decrypts the inner call of a relayMessageArgs if it is encrypted, using the specified public key.
func (v *relayMessageArgs) decryptInnerCallWithPubKey(compressedPubKey []byte) error {
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
		} else if decrypted, err := encrypted.decryptWithPubKey(compressedPubKey); err != nil {
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

// DecryptTxDataWithPubKey decrypts the transaction data using the specified compressed public key.
// This function supports key rotation by allowing the caller to specify which public key to use.
// compressedPubKey must be provided (33 bytes) - it identifies which private key to use for decryption.
func DecryptTxDataWithPubKey(data []byte, compressedPubKey []byte) ([]byte, error) {
	if len(data) < 4 {
		return data, nil
	}

	if compressedPubKey == nil {
		return nil, fmt.Errorf("compressedPubKey must be provided")
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
		} else if args, err = encrypted.decryptWithPubKey(compressedPubKey); err != nil {
			return nil, err
		}

	// payload is unknown, nothing to do
	default:
		return data, nil
	}

	// decrypt inner call
	if err := args.decryptInnerCallWithPubKey(compressedPubKey); err != nil {
		return nil, err
	} else if data, err = args.packFunctionCall(); err != nil { // overwrite the message with the decrypted call
		return nil, err
	} else {
		return data, nil
	}
}
