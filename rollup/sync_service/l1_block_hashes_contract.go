// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package sync_service

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// L1BlockHashesMetaData contains all meta data concerning the L1BlockHashes contract.
var L1BlockHashesMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"_blocks\",\"type\":\"bytes32[]\"}],\"name\":\"appendBlockhashes\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_number\",\"type\":\"uint256\"}],\"name\":\"l1Blockhash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"hash_\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockhash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"hash_\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// L1BlockHashesABI is the input ABI used to generate the binding from.
// Deprecated: Use L1BlockHashesMetaData.ABI instead.
var L1BlockHashesABI = L1BlockHashesMetaData.ABI

// L1BlockHashes is an auto generated Go binding around an Ethereum contract.
type L1BlockHashes struct {
	L1BlockHashesCaller     // Read-only binding to the contract
	L1BlockHashesTransactor // Write-only binding to the contract
	L1BlockHashesFilterer   // Log filterer for contract events
}

// L1BlockHashesCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1BlockHashesCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockHashesTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1BlockHashesTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockHashesFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1BlockHashesFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockHashesSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1BlockHashesSession struct {
	Contract     *L1BlockHashes    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1BlockHashesCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1BlockHashesCallerSession struct {
	Contract *L1BlockHashesCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// L1BlockHashesTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1BlockHashesTransactorSession struct {
	Contract     *L1BlockHashesTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// L1BlockHashesRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1BlockHashesRaw struct {
	Contract *L1BlockHashes // Generic contract binding to access the raw methods on
}

// L1BlockHashesCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1BlockHashesCallerRaw struct {
	Contract *L1BlockHashesCaller // Generic read-only contract binding to access the raw methods on
}

// L1BlockHashesTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1BlockHashesTransactorRaw struct {
	Contract *L1BlockHashesTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1BlockHashes creates a new instance of L1BlockHashes, bound to a specific deployed contract.
func NewL1BlockHashes(address common.Address, backend bind.ContractBackend) (*L1BlockHashes, error) {
	contract, err := bindL1BlockHashes(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1BlockHashes{L1BlockHashesCaller: L1BlockHashesCaller{contract: contract}, L1BlockHashesTransactor: L1BlockHashesTransactor{contract: contract}, L1BlockHashesFilterer: L1BlockHashesFilterer{contract: contract}}, nil
}

// NewL1BlockHashesCaller creates a new read-only instance of L1BlockHashes, bound to a specific deployed contract.
func NewL1BlockHashesCaller(address common.Address, caller bind.ContractCaller) (*L1BlockHashesCaller, error) {
	contract, err := bindL1BlockHashes(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlockHashesCaller{contract: contract}, nil
}

// NewL1BlockHashesTransactor creates a new write-only instance of L1BlockHashes, bound to a specific deployed contract.
func NewL1BlockHashesTransactor(address common.Address, transactor bind.ContractTransactor) (*L1BlockHashesTransactor, error) {
	contract, err := bindL1BlockHashes(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlockHashesTransactor{contract: contract}, nil
}

// NewL1BlockHashesFilterer creates a new log filterer instance of L1BlockHashes, bound to a specific deployed contract.
func NewL1BlockHashesFilterer(address common.Address, filterer bind.ContractFilterer) (*L1BlockHashesFilterer, error) {
	contract, err := bindL1BlockHashes(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1BlockHashesFilterer{contract: contract}, nil
}

// bindL1BlockHashes binds a generic wrapper to an already deployed contract.
func bindL1BlockHashes(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := L1BlockHashesMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1BlockHashes *L1BlockHashesRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1BlockHashes.Contract.L1BlockHashesCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1BlockHashes *L1BlockHashesRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BlockHashes.Contract.L1BlockHashesTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1BlockHashes *L1BlockHashesRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1BlockHashes.Contract.L1BlockHashesTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1BlockHashes *L1BlockHashesCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1BlockHashes.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1BlockHashes *L1BlockHashesTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BlockHashes.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1BlockHashes *L1BlockHashesTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1BlockHashes.Contract.contract.Transact(opts, method, params...)
}

// L1Blockhash is a free data retrieval call binding the contract method 0x35c5aba3.
//
// Solidity: function l1Blockhash(uint256 _number) view returns(bytes32 hash_)
func (_L1BlockHashes *L1BlockHashesCaller) L1Blockhash(opts *bind.CallOpts, _number *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _L1BlockHashes.contract.Call(opts, &out, "l1Blockhash", _number)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L1Blockhash is a free data retrieval call binding the contract method 0x35c5aba3.
//
// Solidity: function l1Blockhash(uint256 _number) view returns(bytes32 hash_)
func (_L1BlockHashes *L1BlockHashesSession) L1Blockhash(_number *big.Int) ([32]byte, error) {
	return _L1BlockHashes.Contract.L1Blockhash(&_L1BlockHashes.CallOpts, _number)
}

// L1Blockhash is a free data retrieval call binding the contract method 0x35c5aba3.
//
// Solidity: function l1Blockhash(uint256 _number) view returns(bytes32 hash_)
func (_L1BlockHashes *L1BlockHashesCallerSession) L1Blockhash(_number *big.Int) ([32]byte, error) {
	return _L1BlockHashes.Contract.L1Blockhash(&_L1BlockHashes.CallOpts, _number)
}

// LatestBlockhash is a free data retrieval call binding the contract method 0xeda743d3.
//
// Solidity: function latestBlockhash() view returns(bytes32 hash_)
func (_L1BlockHashes *L1BlockHashesCaller) LatestBlockhash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1BlockHashes.contract.Call(opts, &out, "latestBlockhash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// LatestBlockhash is a free data retrieval call binding the contract method 0xeda743d3.
//
// Solidity: function latestBlockhash() view returns(bytes32 hash_)
func (_L1BlockHashes *L1BlockHashesSession) LatestBlockhash() ([32]byte, error) {
	return _L1BlockHashes.Contract.LatestBlockhash(&_L1BlockHashes.CallOpts)
}

// LatestBlockhash is a free data retrieval call binding the contract method 0xeda743d3.
//
// Solidity: function latestBlockhash() view returns(bytes32 hash_)
func (_L1BlockHashes *L1BlockHashesCallerSession) LatestBlockhash() ([32]byte, error) {
	return _L1BlockHashes.Contract.LatestBlockhash(&_L1BlockHashes.CallOpts)
}

// AppendBlockhashes is a paid mutator transaction binding the contract method 0x9295b80c.
//
// Solidity: function appendBlockhashes(bytes32[] _blocks) returns()
func (_L1BlockHashes *L1BlockHashesTransactor) AppendBlockhashes(opts *bind.TransactOpts, _blocks [][32]byte) (*types.Transaction, error) {
	return _L1BlockHashes.contract.Transact(opts, "appendBlockhashes", _blocks)
}

// AppendBlockhashes is a paid mutator transaction binding the contract method 0x9295b80c.
//
// Solidity: function appendBlockhashes(bytes32[] _blocks) returns()
func (_L1BlockHashes *L1BlockHashesSession) AppendBlockhashes(_blocks [][32]byte) (*types.Transaction, error) {
	return _L1BlockHashes.Contract.AppendBlockhashes(&_L1BlockHashes.TransactOpts, _blocks)
}

// AppendBlockhashes is a paid mutator transaction binding the contract method 0x9295b80c.
//
// Solidity: function appendBlockhashes(bytes32[] _blocks) returns()
func (_L1BlockHashes *L1BlockHashesTransactorSession) AppendBlockhashes(_blocks [][32]byte) (*types.Transaction, error) {
	return _L1BlockHashes.Contract.AppendBlockhashes(&_L1BlockHashes.TransactOpts, _blocks)
}
