package vm

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type traceFunc func(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error

var (
	// OpcodeExecs the map to load opcodes' trace funcs.
	OpcodeExecs = map[OpCode][]traceFunc{
		CALL:         {traceToAddressCode, traceLastNAddressCode(1), traceCallerProof, traceLastNAddressProof(1)},
		CALLCODE:     {traceToAddressCode, traceLastNAddressCode(1), traceCallerProof, traceLastNAddressProof(1)},
		DELEGATECALL: {traceToAddressCode, traceLastNAddressCode(1)},
		STATICCALL:   {traceToAddressCode, traceLastNAddressCode(1), traceLastNAddressProof(1)},
		CREATE:       {}, // sender's wrapped_proof is already recorded in BlockChain.writeBlockResult
		CREATE2:      {}, // sender's wrapped_proof is already recorded in BlockChain.writeBlockResult
		SLOAD:        {}, // record storage_proof in `captureState` instead of here, to handle `l.cfg.DisableStorage` flag
		SSTORE:       {}, // record storage_proof in `captureState` instead of here, to handle `l.cfg.DisableStorage` flag
		SELFDESTRUCT: {traceContractProof, traceLastNAddressProof(0)},
		SELFBALANCE:  {traceContractProof},
		BALANCE:      {traceLastNAddressProof(0)},
		EXTCODEHASH:  {traceLastNAddressProof(0)},
	}
)

// traceToAddressCode gets tx.to address’s code
func traceToAddressCode(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	if l.env.To == nil {
		return nil
	}
	code := l.env.StateDB.GetCode(*l.env.To)
	extraData.CodeList = append(extraData.CodeList, code)
	return nil
}

// traceLastNAddressCode
func traceLastNAddressCode(n int) traceFunc {
	return func(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
		stack := scope.Stack
		if stack.len() <= n {
			return nil
		}
		address := common.Address(stack.data[stack.len()-1-n].Bytes20())
		code := l.env.StateDB.GetCode(address)
		extraData.CodeList = append(extraData.CodeList, code)
		return nil
	}
}

// traceStorageProof get contract's storage proof at storage_address
func traceStorageProof(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	if scope.Stack.len() == 0 {
		return nil
	}
	key := common.Hash(scope.Stack.peek().Bytes32())
	proof, err := getWrappedForStorage(l, scope.Contract.Address(), key)
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, proof)
	}
	return err
}

// traceContractProof gets the contract's account proof
func traceContractProof(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	// Get account proof.
	proof, err := getWrappedForAddr(l, scope.Contract.Address())
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, proof)
		l.statesAffected[scope.Contract.Address()] = struct{}{}
	}
	return err
}

// traceLastNAddressProof returns func about the last N's address proof.
func traceLastNAddressProof(n int) traceFunc {
	return func(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
		stack := scope.Stack
		if stack.len() <= n {
			return nil
		}

		address := common.Address(stack.data[stack.len()-1-n].Bytes20())
		proof, err := getWrappedForAddr(l, address)
		if err == nil {
			extraData.ProofList = append(extraData.ProofList, proof)
			l.statesAffected[scope.Contract.Address()] = struct{}{}
		}
		return err
	}
}

// traceCallerProof gets caller address's proof.
func traceCallerProof(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	address := scope.Contract.CallerAddress
	proof, err := getWrappedForAddr(l, address)
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, proof)
		l.statesAffected[scope.Contract.Address()] = struct{}{}
	}
	return err
}

// StorageWrapper will be empty
func getWrappedForAddr(l *StructLogger, address common.Address) (*types.AccountWrapper, error) {
	return &types.AccountWrapper{
		Address:  address,
		Nonce:    l.env.StateDB.GetNonce(address),
		Balance:  (*hexutil.Big)(l.env.StateDB.GetBalance(address)),
		CodeHash: l.env.StateDB.GetCodeHash(address),
	}, nil
}

func getWrappedForStorage(l *StructLogger, address common.Address, key common.Hash) (*types.AccountWrapper, error) {
	return &types.AccountWrapper{
		Address:  address,
		Nonce:    l.env.StateDB.GetNonce(address),
		Balance:  (*hexutil.Big)(l.env.StateDB.GetBalance(address)),
		CodeHash: l.env.StateDB.GetCodeHash(address),
		Storage: &types.StorageWrapper{
			Key:   key.String(),
			Value: l.env.StateDB.GetState(address, key).String(),
		},
	}, nil
}
