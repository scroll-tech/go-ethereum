// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abis

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
)

// L1BlocksMetaData contains all meta data concerning the L1Blocks contract.
var L1BlocksMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"ErrorBlockUnavailable\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockHeight\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockTimestamp\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"}],\"name\":\"ImportBlock\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getBaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getBlobBaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getParentBeaconRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getStateRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlobBaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestParentBeaconRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestStateRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"blockHeaderRlp\",\"type\":\"bytes\"}],\"name\":\"setL1BlockHeader\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// L1BlocksABI is the input ABI used to generate the binding from.
// Deprecated: Use L1BlocksMetaData.ABI instead.
var L1BlocksABI = L1BlocksMetaData.ABI

// L1Blocks is an auto generated Go binding around an Ethereum contract.
type L1Blocks struct {
	L1BlocksCaller     // Read-only binding to the contract
	L1BlocksTransactor // Write-only binding to the contract
	L1BlocksFilterer   // Log filterer for contract events
}

// L1BlocksCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1BlocksCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlocksTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1BlocksTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlocksFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1BlocksFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlocksSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1BlocksSession struct {
	Contract     *L1Blocks         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1BlocksCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1BlocksCallerSession struct {
	Contract *L1BlocksCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// L1BlocksTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1BlocksTransactorSession struct {
	Contract     *L1BlocksTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// L1BlocksRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1BlocksRaw struct {
	Contract *L1Blocks // Generic contract binding to access the raw methods on
}

// L1BlocksCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1BlocksCallerRaw struct {
	Contract *L1BlocksCaller // Generic read-only contract binding to access the raw methods on
}

// L1BlocksTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1BlocksTransactorRaw struct {
	Contract *L1BlocksTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1Blocks creates a new instance of L1Blocks, bound to a specific deployed contract.
func NewL1Blocks(address common.Address, backend bind.ContractBackend) (*L1Blocks, error) {
	contract, err := bindL1Blocks(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1Blocks{L1BlocksCaller: L1BlocksCaller{contract: contract}, L1BlocksTransactor: L1BlocksTransactor{contract: contract}, L1BlocksFilterer: L1BlocksFilterer{contract: contract}}, nil
}

// NewL1BlocksCaller creates a new read-only instance of L1Blocks, bound to a specific deployed contract.
func NewL1BlocksCaller(address common.Address, caller bind.ContractCaller) (*L1BlocksCaller, error) {
	contract, err := bindL1Blocks(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlocksCaller{contract: contract}, nil
}

// NewL1BlocksTransactor creates a new write-only instance of L1Blocks, bound to a specific deployed contract.
func NewL1BlocksTransactor(address common.Address, transactor bind.ContractTransactor) (*L1BlocksTransactor, error) {
	contract, err := bindL1Blocks(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlocksTransactor{contract: contract}, nil
}

// NewL1BlocksFilterer creates a new log filterer instance of L1Blocks, bound to a specific deployed contract.
func NewL1BlocksFilterer(address common.Address, filterer bind.ContractFilterer) (*L1BlocksFilterer, error) {
	contract, err := bindL1Blocks(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1BlocksFilterer{contract: contract}, nil
}

// bindL1Blocks binds a generic wrapper to an already deployed contract.
func bindL1Blocks(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1BlocksABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1Blocks *L1BlocksRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1Blocks.Contract.L1BlocksCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1Blocks *L1BlocksRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1Blocks.Contract.L1BlocksTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1Blocks *L1BlocksRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1Blocks.Contract.L1BlocksTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1Blocks *L1BlocksCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1Blocks.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1Blocks *L1BlocksTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1Blocks.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1Blocks *L1BlocksTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1Blocks.Contract.contract.Transact(opts, method, params...)
}

// GetBaseFee is a free data retrieval call binding the contract method 0x6c8af435.
//
// Solidity: function getBaseFee(uint256 blockNumber) view returns(uint256)
func (_L1Blocks *L1BlocksCaller) GetBaseFee(opts *bind.CallOpts, blockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "getBaseFee", blockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBaseFee is a free data retrieval call binding the contract method 0x6c8af435.
//
// Solidity: function getBaseFee(uint256 blockNumber) view returns(uint256)
func (_L1Blocks *L1BlocksSession) GetBaseFee(blockNumber *big.Int) (*big.Int, error) {
	return _L1Blocks.Contract.GetBaseFee(&_L1Blocks.CallOpts, blockNumber)
}

// GetBaseFee is a free data retrieval call binding the contract method 0x6c8af435.
//
// Solidity: function getBaseFee(uint256 blockNumber) view returns(uint256)
func (_L1Blocks *L1BlocksCallerSession) GetBaseFee(blockNumber *big.Int) (*big.Int, error) {
	return _L1Blocks.Contract.GetBaseFee(&_L1Blocks.CallOpts, blockNumber)
}

// GetBlobBaseFee is a free data retrieval call binding the contract method 0x7e96ce1c.
//
// Solidity: function getBlobBaseFee(uint256 blockNumber) view returns(uint256)
func (_L1Blocks *L1BlocksCaller) GetBlobBaseFee(opts *bind.CallOpts, blockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "getBlobBaseFee", blockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBlobBaseFee is a free data retrieval call binding the contract method 0x7e96ce1c.
//
// Solidity: function getBlobBaseFee(uint256 blockNumber) view returns(uint256)
func (_L1Blocks *L1BlocksSession) GetBlobBaseFee(blockNumber *big.Int) (*big.Int, error) {
	return _L1Blocks.Contract.GetBlobBaseFee(&_L1Blocks.CallOpts, blockNumber)
}

// GetBlobBaseFee is a free data retrieval call binding the contract method 0x7e96ce1c.
//
// Solidity: function getBlobBaseFee(uint256 blockNumber) view returns(uint256)
func (_L1Blocks *L1BlocksCallerSession) GetBlobBaseFee(blockNumber *big.Int) (*big.Int, error) {
	return _L1Blocks.Contract.GetBlobBaseFee(&_L1Blocks.CallOpts, blockNumber)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32)
func (_L1Blocks *L1BlocksCaller) GetBlockHash(opts *bind.CallOpts, blockNumber *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "getBlockHash", blockNumber)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32)
func (_L1Blocks *L1BlocksSession) GetBlockHash(blockNumber *big.Int) ([32]byte, error) {
	return _L1Blocks.Contract.GetBlockHash(&_L1Blocks.CallOpts, blockNumber)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32)
func (_L1Blocks *L1BlocksCallerSession) GetBlockHash(blockNumber *big.Int) ([32]byte, error) {
	return _L1Blocks.Contract.GetBlockHash(&_L1Blocks.CallOpts, blockNumber)
}

// GetBlockTimestamp is a free data retrieval call binding the contract method 0x47e26f1a.
//
// Solidity: function getBlockTimestamp(uint256 blockNumber) view returns(uint256)
func (_L1Blocks *L1BlocksCaller) GetBlockTimestamp(opts *bind.CallOpts, blockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "getBlockTimestamp", blockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBlockTimestamp is a free data retrieval call binding the contract method 0x47e26f1a.
//
// Solidity: function getBlockTimestamp(uint256 blockNumber) view returns(uint256)
func (_L1Blocks *L1BlocksSession) GetBlockTimestamp(blockNumber *big.Int) (*big.Int, error) {
	return _L1Blocks.Contract.GetBlockTimestamp(&_L1Blocks.CallOpts, blockNumber)
}

// GetBlockTimestamp is a free data retrieval call binding the contract method 0x47e26f1a.
//
// Solidity: function getBlockTimestamp(uint256 blockNumber) view returns(uint256)
func (_L1Blocks *L1BlocksCallerSession) GetBlockTimestamp(blockNumber *big.Int) (*big.Int, error) {
	return _L1Blocks.Contract.GetBlockTimestamp(&_L1Blocks.CallOpts, blockNumber)
}

// GetParentBeaconRoot is a free data retrieval call binding the contract method 0x78d8f5fe.
//
// Solidity: function getParentBeaconRoot(uint256 blockNumber) view returns(bytes32)
func (_L1Blocks *L1BlocksCaller) GetParentBeaconRoot(opts *bind.CallOpts, blockNumber *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "getParentBeaconRoot", blockNumber)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetParentBeaconRoot is a free data retrieval call binding the contract method 0x78d8f5fe.
//
// Solidity: function getParentBeaconRoot(uint256 blockNumber) view returns(bytes32)
func (_L1Blocks *L1BlocksSession) GetParentBeaconRoot(blockNumber *big.Int) ([32]byte, error) {
	return _L1Blocks.Contract.GetParentBeaconRoot(&_L1Blocks.CallOpts, blockNumber)
}

// GetParentBeaconRoot is a free data retrieval call binding the contract method 0x78d8f5fe.
//
// Solidity: function getParentBeaconRoot(uint256 blockNumber) view returns(bytes32)
func (_L1Blocks *L1BlocksCallerSession) GetParentBeaconRoot(blockNumber *big.Int) ([32]byte, error) {
	return _L1Blocks.Contract.GetParentBeaconRoot(&_L1Blocks.CallOpts, blockNumber)
}

// GetStateRoot is a free data retrieval call binding the contract method 0xc3801938.
//
// Solidity: function getStateRoot(uint256 blockNumber) view returns(bytes32)
func (_L1Blocks *L1BlocksCaller) GetStateRoot(opts *bind.CallOpts, blockNumber *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "getStateRoot", blockNumber)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetStateRoot is a free data retrieval call binding the contract method 0xc3801938.
//
// Solidity: function getStateRoot(uint256 blockNumber) view returns(bytes32)
func (_L1Blocks *L1BlocksSession) GetStateRoot(blockNumber *big.Int) ([32]byte, error) {
	return _L1Blocks.Contract.GetStateRoot(&_L1Blocks.CallOpts, blockNumber)
}

// GetStateRoot is a free data retrieval call binding the contract method 0xc3801938.
//
// Solidity: function getStateRoot(uint256 blockNumber) view returns(bytes32)
func (_L1Blocks *L1BlocksCallerSession) GetStateRoot(blockNumber *big.Int) ([32]byte, error) {
	return _L1Blocks.Contract.GetStateRoot(&_L1Blocks.CallOpts, blockNumber)
}

// LatestBaseFee is a free data retrieval call binding the contract method 0x0385f4f1.
//
// Solidity: function latestBaseFee() view returns(uint256)
func (_L1Blocks *L1BlocksCaller) LatestBaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "latestBaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBaseFee is a free data retrieval call binding the contract method 0x0385f4f1.
//
// Solidity: function latestBaseFee() view returns(uint256)
func (_L1Blocks *L1BlocksSession) LatestBaseFee() (*big.Int, error) {
	return _L1Blocks.Contract.LatestBaseFee(&_L1Blocks.CallOpts)
}

// LatestBaseFee is a free data retrieval call binding the contract method 0x0385f4f1.
//
// Solidity: function latestBaseFee() view returns(uint256)
func (_L1Blocks *L1BlocksCallerSession) LatestBaseFee() (*big.Int, error) {
	return _L1Blocks.Contract.LatestBaseFee(&_L1Blocks.CallOpts)
}

// LatestBlobBaseFee is a free data retrieval call binding the contract method 0x6146da50.
//
// Solidity: function latestBlobBaseFee() view returns(uint256)
func (_L1Blocks *L1BlocksCaller) LatestBlobBaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "latestBlobBaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlobBaseFee is a free data retrieval call binding the contract method 0x6146da50.
//
// Solidity: function latestBlobBaseFee() view returns(uint256)
func (_L1Blocks *L1BlocksSession) LatestBlobBaseFee() (*big.Int, error) {
	return _L1Blocks.Contract.LatestBlobBaseFee(&_L1Blocks.CallOpts)
}

// LatestBlobBaseFee is a free data retrieval call binding the contract method 0x6146da50.
//
// Solidity: function latestBlobBaseFee() view returns(uint256)
func (_L1Blocks *L1BlocksCallerSession) LatestBlobBaseFee() (*big.Int, error) {
	return _L1Blocks.Contract.LatestBlobBaseFee(&_L1Blocks.CallOpts)
}

// LatestBlockHash is a free data retrieval call binding the contract method 0x6c4f6ba9.
//
// Solidity: function latestBlockHash() view returns(bytes32)
func (_L1Blocks *L1BlocksCaller) LatestBlockHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "latestBlockHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// LatestBlockHash is a free data retrieval call binding the contract method 0x6c4f6ba9.
//
// Solidity: function latestBlockHash() view returns(bytes32)
func (_L1Blocks *L1BlocksSession) LatestBlockHash() ([32]byte, error) {
	return _L1Blocks.Contract.LatestBlockHash(&_L1Blocks.CallOpts)
}

// LatestBlockHash is a free data retrieval call binding the contract method 0x6c4f6ba9.
//
// Solidity: function latestBlockHash() view returns(bytes32)
func (_L1Blocks *L1BlocksCallerSession) LatestBlockHash() ([32]byte, error) {
	return _L1Blocks.Contract.LatestBlockHash(&_L1Blocks.CallOpts)
}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L1Blocks *L1BlocksCaller) LatestBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "latestBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L1Blocks *L1BlocksSession) LatestBlockNumber() (*big.Int, error) {
	return _L1Blocks.Contract.LatestBlockNumber(&_L1Blocks.CallOpts)
}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L1Blocks *L1BlocksCallerSession) LatestBlockNumber() (*big.Int, error) {
	return _L1Blocks.Contract.LatestBlockNumber(&_L1Blocks.CallOpts)
}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_L1Blocks *L1BlocksCaller) LatestBlockTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "latestBlockTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_L1Blocks *L1BlocksSession) LatestBlockTimestamp() (*big.Int, error) {
	return _L1Blocks.Contract.LatestBlockTimestamp(&_L1Blocks.CallOpts)
}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_L1Blocks *L1BlocksCallerSession) LatestBlockTimestamp() (*big.Int, error) {
	return _L1Blocks.Contract.LatestBlockTimestamp(&_L1Blocks.CallOpts)
}

// LatestParentBeaconRoot is a free data retrieval call binding the contract method 0xa3483d56.
//
// Solidity: function latestParentBeaconRoot() view returns(bytes32)
func (_L1Blocks *L1BlocksCaller) LatestParentBeaconRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "latestParentBeaconRoot")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// LatestParentBeaconRoot is a free data retrieval call binding the contract method 0xa3483d56.
//
// Solidity: function latestParentBeaconRoot() view returns(bytes32)
func (_L1Blocks *L1BlocksSession) LatestParentBeaconRoot() ([32]byte, error) {
	return _L1Blocks.Contract.LatestParentBeaconRoot(&_L1Blocks.CallOpts)
}

// LatestParentBeaconRoot is a free data retrieval call binding the contract method 0xa3483d56.
//
// Solidity: function latestParentBeaconRoot() view returns(bytes32)
func (_L1Blocks *L1BlocksCallerSession) LatestParentBeaconRoot() ([32]byte, error) {
	return _L1Blocks.Contract.LatestParentBeaconRoot(&_L1Blocks.CallOpts)
}

// LatestStateRoot is a free data retrieval call binding the contract method 0x991beafd.
//
// Solidity: function latestStateRoot() view returns(bytes32)
func (_L1Blocks *L1BlocksCaller) LatestStateRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1Blocks.contract.Call(opts, &out, "latestStateRoot")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// LatestStateRoot is a free data retrieval call binding the contract method 0x991beafd.
//
// Solidity: function latestStateRoot() view returns(bytes32)
func (_L1Blocks *L1BlocksSession) LatestStateRoot() ([32]byte, error) {
	return _L1Blocks.Contract.LatestStateRoot(&_L1Blocks.CallOpts)
}

// LatestStateRoot is a free data retrieval call binding the contract method 0x991beafd.
//
// Solidity: function latestStateRoot() view returns(bytes32)
func (_L1Blocks *L1BlocksCallerSession) LatestStateRoot() ([32]byte, error) {
	return _L1Blocks.Contract.LatestStateRoot(&_L1Blocks.CallOpts)
}

// SetL1BlockHeader is a paid mutator transaction binding the contract method 0x6a9100cc.
//
// Solidity: function setL1BlockHeader(bytes blockHeaderRlp) returns(bytes32 blockHash)
func (_L1Blocks *L1BlocksTransactor) SetL1BlockHeader(opts *bind.TransactOpts, blockHeaderRlp []byte) (*types.Transaction, error) {
	return _L1Blocks.contract.Transact(opts, "setL1BlockHeader", blockHeaderRlp)
}

// SetL1BlockHeader is a paid mutator transaction binding the contract method 0x6a9100cc.
//
// Solidity: function setL1BlockHeader(bytes blockHeaderRlp) returns(bytes32 blockHash)
func (_L1Blocks *L1BlocksSession) SetL1BlockHeader(blockHeaderRlp []byte) (*types.Transaction, error) {
	return _L1Blocks.Contract.SetL1BlockHeader(&_L1Blocks.TransactOpts, blockHeaderRlp)
}

// SetL1BlockHeader is a paid mutator transaction binding the contract method 0x6a9100cc.
//
// Solidity: function setL1BlockHeader(bytes blockHeaderRlp) returns(bytes32 blockHash)
func (_L1Blocks *L1BlocksTransactorSession) SetL1BlockHeader(blockHeaderRlp []byte) (*types.Transaction, error) {
	return _L1Blocks.Contract.SetL1BlockHeader(&_L1Blocks.TransactOpts, blockHeaderRlp)
}

// L1BlocksImportBlockIterator is returned from FilterImportBlock and is used to iterate over the raw logs and unpacked data for ImportBlock events raised by the L1Blocks contract.
type L1BlocksImportBlockIterator struct {
	Event *L1BlocksImportBlock // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *L1BlocksImportBlockIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BlocksImportBlock)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(L1BlocksImportBlock)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *L1BlocksImportBlockIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BlocksImportBlockIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BlocksImportBlock represents a ImportBlock event raised by the L1Blocks contract.
type L1BlocksImportBlock struct {
	BlockHash      [32]byte
	BlockHeight    *big.Int
	BlockTimestamp *big.Int
	BaseFee        *big.Int
	StateRoot      [32]byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterImportBlock is a free log retrieval operation binding the contract event 0xa7823f45e1ee21f9530b77959b57507ad515a14fa9fa24d262ee80e79b2b5745.
//
// Solidity: event ImportBlock(bytes32 indexed blockHash, uint256 blockHeight, uint256 blockTimestamp, uint256 baseFee, bytes32 stateRoot)
func (_L1Blocks *L1BlocksFilterer) FilterImportBlock(opts *bind.FilterOpts, blockHash [][32]byte) (*L1BlocksImportBlockIterator, error) {

	var blockHashRule []interface{}
	for _, blockHashItem := range blockHash {
		blockHashRule = append(blockHashRule, blockHashItem)
	}

	logs, sub, err := _L1Blocks.contract.FilterLogs(opts, "ImportBlock", blockHashRule)
	if err != nil {
		return nil, err
	}
	return &L1BlocksImportBlockIterator{contract: _L1Blocks.contract, event: "ImportBlock", logs: logs, sub: sub}, nil
}

// WatchImportBlock is a free log subscription operation binding the contract event 0xa7823f45e1ee21f9530b77959b57507ad515a14fa9fa24d262ee80e79b2b5745.
//
// Solidity: event ImportBlock(bytes32 indexed blockHash, uint256 blockHeight, uint256 blockTimestamp, uint256 baseFee, bytes32 stateRoot)
func (_L1Blocks *L1BlocksFilterer) WatchImportBlock(opts *bind.WatchOpts, sink chan<- *L1BlocksImportBlock, blockHash [][32]byte) (event.Subscription, error) {

	var blockHashRule []interface{}
	for _, blockHashItem := range blockHash {
		blockHashRule = append(blockHashRule, blockHashItem)
	}

	logs, sub, err := _L1Blocks.contract.WatchLogs(opts, "ImportBlock", blockHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BlocksImportBlock)
				if err := _L1Blocks.contract.UnpackLog(event, "ImportBlock", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseImportBlock is a log parse operation binding the contract event 0xa7823f45e1ee21f9530b77959b57507ad515a14fa9fa24d262ee80e79b2b5745.
//
// Solidity: event ImportBlock(bytes32 indexed blockHash, uint256 blockHeight, uint256 blockTimestamp, uint256 baseFee, bytes32 stateRoot)
func (_L1Blocks *L1BlocksFilterer) ParseImportBlock(log types.Log) (*L1BlocksImportBlock, error) {
	event := new(L1BlocksImportBlock)
	if err := _L1Blocks.contract.UnpackLog(event, "ImportBlock", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
