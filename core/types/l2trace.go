package types

import (
	"encoding/json"
	"runtime"
	"strconv"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
)

var (
	loggerResPool = sync.Pool{
		New: func() interface{} {
			// init arrays here; other types are inited with default values
			return &StructLogRes{
				Stack:  []string{},
				Memory: []string{},
			}
		},
	}
)

// BlockResult contains block execution traces and results required for rollers.
type BlockResult struct {
	BlockTrace       *BlockTrace        `json:"blockTrace"`
	StorageTrace     *StorageTrace      `json:"storageTrace"`
	ExecutionResults []*ExecutionResult `json:"executionResults"`
	MPTWitness       *json.RawMessage   `json:"mptwitness,omitempty"`
}

type AliasBlockResult BlockResult

// Store content in cache and return the "'s'+index".
func getIndex(cache map[string]int, content string) string {
	if len(content) < 10 {
		return content
	}
	idx, ok := cache[content]
	if !ok {
		idx = len(cache)
		cache[content] = idx
	}
	return "s" + strconv.Itoa(idx)
}

// get origin content from freCache by index
func getOrigin(freqCache []string, sIndex string) string {
	if len(sIndex) > 1 && sIndex[0] == 's' {
		idx, _ := strconv.Atoi(sIndex[1:])
		return freqCache[idx]
	}
	return sIndex
}

// MarshalJSON marshal block trace.
func (b *BlockResult) MarshalJSON() ([]byte, error) {
	js := struct {
		FreqCache []string `json:"freqCache,omitempty"`
		AliasBlockResult
		cache map[string]int
		err   error
	}{nil, AliasBlockResult(*b), make(map[string]int), nil}

	for _, results := range js.ExecutionResults {
		for _, logs := range results.StructLogs {
			// replace stack
			for i, stack := range logs.Stack {
				logs.Stack[i] = getIndex(js.cache, stack)
			}
			// replace memory
			for i, mem := range logs.Memory {
				logs.Memory[i] = getIndex(js.cache, mem)
			}
			// replace storage
			storage := make(map[string]string, len(logs.Storage))
			for key, val := range logs.Storage {
				storage[getIndex(js.cache, key)] = getIndex(js.cache, val)
			}
			logs.Storage = storage
			// replace extra data
			extra := logs.ExtraData
			if extra != nil {
				for i, code := range extra.CodeList {
					extra.CodeList[i] = getIndex(js.cache, code)
				}
				for _, lst := range extra.StateList {
					lst.Storage.Key = getIndex(js.cache, lst.Storage.Key)
					lst.Storage.Value = getIndex(js.cache, lst.Storage.Value)
				}
			}
		}
	}
	js.FreqCache = make([]string, len(js.cache))
	for key, idx := range js.cache {
		js.FreqCache[idx] = key
	}

	return json.Marshal(&js)
}

// UnmarshalJSON unmarshal bytes to blockResult struct.
func (b *BlockResult) UnmarshalJSON(input []byte) error {
	var js struct {
		AliasBlockResult
		FreqCache []string `json:"freqCache"`
	}
	if err := json.Unmarshal(input, &js); err != nil {
		return err
	}

	for _, results := range js.ExecutionResults {
		for _, logs := range results.StructLogs {
			// getOrigin stack
			for i, stack := range logs.Stack {
				logs.Stack[i] = getOrigin(js.FreqCache, stack)
			}
			// replace memory
			for i, mem := range logs.Memory {
				logs.Memory[i] = getOrigin(js.FreqCache, mem)
			}
			storage := make(map[string]string, len(logs.Storage))
			for key, val := range logs.Storage {
				storage[getOrigin(js.FreqCache, key)] = getOrigin(js.FreqCache, val)
			}
			logs.Storage = storage
			// replace extra data
			extra := logs.ExtraData
			if extra != nil {
				for i, code := range extra.CodeList {
					extra.CodeList[i] = getOrigin(js.FreqCache, code)
				}
				for _, lst := range extra.StateList {
					lst.Storage.Key = getOrigin(js.FreqCache, lst.Storage.Key)
					lst.Storage.Value = getOrigin(js.FreqCache, lst.Storage.Value)
				}
			}
		}
	}

	*b = BlockResult(js.AliasBlockResult)
	return nil
}

// StorageTrace stores proofs of storage needed by storage circuit
type StorageTrace struct {
	// Root hash before block execution:
	RootBefore common.Hash `json:"rootBefore,omitempty"`
	// Root hash after block execution, is nil if execution has failed
	RootAfter common.Hash `json:"rootAfter,omitempty"`

	// All proofs BEFORE execution, for accounts which would be used in tracing
	Proofs map[string][]hexutil.Bytes `json:"proofs"`

	// All storage proofs BEFORE execution
	StorageProofs map[string]map[string][]hexutil.Bytes `json:"storageProofs,omitempty"`
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         uint64 `json:"gas"`
	Failed      bool   `json:"failed"`
	ReturnValue string `json:"returnValue,omitempty"`
	// Sender's account state (before Tx)
	From *AccountWrapper `json:"from,omitempty"`
	// Receiver's account state (before Tx)
	To *AccountWrapper `json:"to,omitempty"`
	// AccountCreated record the account if the tx is "create"
	// (for creating inside a contract, we just handle CREATE op)
	AccountCreated *AccountWrapper `json:"accountCreated,omitempty"`

	// Record all accounts' state which would be affected AFTER tx executed
	// currently they are just `from` and `to` account
	AccountsAfter []*AccountWrapper `json:"accountAfter"`

	// `CodeHash` only exists when tx is a contract call.
	CodeHash *common.Hash `json:"codeHash,omitempty"`
	// If it is a contract call, the contract code is returned.
	ByteCode   string          `json:"byteCode,omitempty"`
	StructLogs []*StructLogRes `json:"structLogs"`
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc            uint64            `json:"pc"`
	Op            string            `json:"op"`
	Gas           uint64            `json:"gas"`
	GasCost       uint64            `json:"gasCost"`
	Depth         int               `json:"depth"`
	Error         string            `json:"error,omitempty"`
	Stack         []string          `json:"stack,omitempty"`
	Memory        []string          `json:"memory,omitempty"`
	Storage       map[string]string `json:"storage,omitempty"`
	RefundCounter uint64            `json:"refund,omitempty"`
	ExtraData     *ExtraData        `json:"extraData,omitempty"`
}

// Basic StructLogRes skeleton, Stack&Memory&Storage&ExtraData are separated from it for GC optimization;
// still need to fill in with Stack&Memory&Storage&ExtraData
func NewStructLogResBasic(pc uint64, op string, gas, gasCost uint64, depth int, refundCounter uint64, err error) *StructLogRes {
	logRes := loggerResPool.Get().(*StructLogRes)
	logRes.Pc, logRes.Op, logRes.Gas, logRes.GasCost, logRes.Depth, logRes.RefundCounter = pc, op, gas, gasCost, depth, refundCounter
	if err != nil {
		logRes.Error = err.Error()
	}
	runtime.SetFinalizer(logRes, func(logRes *StructLogRes) {
		logRes.Stack = logRes.Stack[:0]
		logRes.Memory = logRes.Memory[:0]
		logRes.Storage = nil
		logRes.ExtraData = nil
		loggerResPool.Put(logRes)
	})
	return logRes
}

type ExtraData struct {
	// Indicate the call succeeds or not for CALL/CREATE op
	CallFailed bool `json:"callFailed,omitempty"`
	// CALL | CALLCODE | DELEGATECALL | STATICCALL: [tx.to address’s code, stack.nth_last(1) address’s code]
	// CREATE | CREATE2: [created contract’s code]
	// CODESIZE | CODECOPY: [contract’s code]
	// EXTCODESIZE | EXTCODECOPY: [stack.nth_last(0) address’s code]
	CodeList []string `json:"codeList,omitempty"`
	// SSTORE | SLOAD: [storageProof]
	// SELFDESTRUCT: [contract address’s account, stack.nth_last(0) address’s account]
	// SELFBALANCE: [contract address’s account]
	// BALANCE | EXTCODEHASH: [stack.nth_last(0) address’s account]
	// CREATE | CREATE2: [created contract address’s account (before constructed),
	// 					  created contract address's account (after constructed)]
	// CALL | CALLCODE: [caller contract address’s account,
	// 					stack.nth_last(1) (i.e. callee) address’s account,
	//					callee contract address's account (value updated, before called)]
	// STATICCALL: [stack.nth_last(1) (i.e. callee) address’s account,
	//					  callee contract address's account (before called)]
	StateList []*AccountWrapper `json:"proofList,omitempty"`
}

type AccountWrapper struct {
	Address  common.Address  `json:"address"`
	Nonce    uint64          `json:"nonce"`
	Balance  *hexutil.Big    `json:"balance"`
	CodeHash common.Hash     `json:"codeHash,omitempty"`
	Storage  *StorageWrapper `json:"storage,omitempty"` // StorageWrapper can be empty if irrelated to storage operation
}

// while key & value can also be retrieved from StructLogRes.Storage,
// we still stored in here for roller's processing convenience.
type StorageWrapper struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}
