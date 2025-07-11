package validium

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sync"

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

	SequencerKey *ecies.PrivateKey
	lock         sync.RWMutex
)

func SetSequencerKey(key *ecies.PrivateKey) {
	lock.Lock()
	defer lock.Unlock()
	SequencerKey = key
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
func decryptAddress(data []byte) (common.Address, error) {
	raw, err := ecies.Decrypt(SequencerKey, data)
	if err != nil {
		log.Warn("failed to decrypt address", "data", common.Bytes2Hex(data), "error", err)
		return common.Address{}, errors.New("decryption failed")
	}

	plaintext := string(raw)
	if !common.IsHexAddress(plaintext) {
		log.Warn("decrypted data is not address", "data", common.Bytes2Hex(data), "plaintext", plaintext)
		return common.Address{}, errors.New("decryption failed")
	}

	return common.HexToAddress(plaintext), nil
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
