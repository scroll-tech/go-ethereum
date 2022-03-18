package types

import (
	"github.com/scroll-tech/go-ethereum/common"
)

// BlockResult contains block execution traces and results required for rollers.
type BlockResult struct {
	BlockTrace       *BlockTrace        `json:"blockTrace"`
	ExecutionResults []*ExecutionResult `json:"executionResults"`
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         uint64 `json:"gas"`
	Failed      bool   `json:"failed"`
	ReturnValue string `json:"returnValue,omitempty"`
	// It's exist only when tx is a contract call.
	CodeHash *common.Hash `json:"codeHash,omitempty"`
	// If it is a contract call, the contract code is returned.
	ByteCode string `json:"byteCode,omitempty"`

	// Deprecated: The account's proof.
	// Proof      []string       `json:"proof,omitempty"`

	Storage    *StorageRes    `json:"storage,omitempty"`
	StructLogs []StructLogRes `json:"structLogs"`
}

// SMTPathNode represent a node in the SMT Path
type SMTPathNode struct {
	Value    string `json:"value"`
	Silbling string `json:"silbiling"`
}

// SMTPath is the whole path of SMT
type SMTPath struct {
	Root string        `json:"root"`
	Path []SMTPathNode `json:"path"` //from top to leaf
}

// StateAccountL2 is the represent of StateAccount in L2 circuit
// Notice in L2 we have different hash scheme against StateAccount.MarshalByte
type StateAccountL2 struct {
	Address  string `json:"address"`
	Nonce    int    `json:"nonce"`
	Balance  string `json:"balance"` //just the common hex expression of integer (big-endian)
	CodeHash string `json:"codeHash,omitempty"`
}

// StateStorageL2 is the represent of a stored key-value pair for specified account
type StateStorageL2 struct {
	Key   string `json:"key"` //notice this is the preimage of storage key
	Value string `json:"value"`
}

// StateTrace record the updating on state trie and (if changed) account trie
// represent by the [before, after] updating of SMTPath amont tries and Account
type StateTrace struct {
	// which log the trace is responded for, -1 indicate not caused
	// by opcode (like gasRefund, coinbase, setNonce, etc)
	Index            int                `json:"index"`
	AccountKey       string             `json:"accountKey"`
	AccountPath      [2]*SMTPath        `json:"accountPath"`
	AccountUpdate    [2]*StateAccountL2 `json:"accountUpdate"`
	StateKey         string             `json:"stateKey,omitempty"`
	CommonStateRoot  string             `json:"commonStateRoot,omitempty"`
	StatePath        [2]*SMTPath        `json:"statePath,omitempty"`
	StateUpdate      [2]*StateStorageL2 `json:"stateUpdate,omitempty"`
	AccountKeyBefore string             `json:"accountKeyBefore,omitempty"`
	StateKeyBefore   string             `json:"stateKeyBefore,omitempty"`
}

// StorageRes stores data required in storage circuit
type StorageRes struct {

	// Root hash before execution:
	RootBefore *common.Hash `json:"rootBefore,omitempty"`
	// Root hash after execution, is nil if execution has failed
	RootAfter *common.Hash `json:"rootAfter,omitempty"`
	// AccountsAfter recode and encoded all accounts
	AccountsAfter map[string]string `json:"accountAfter"`

	// The from account's proof BEFORE execution
	ProofFrom []string `json:"proofFrom,omitempty"`
	// The to account's proof BEFORE execution, these proof,
	// along with account proof's inside structLogs, form the
	// dataset required by tracing the updates of account trie
	ProofTo []string `json:"proofTo,omitempty"`
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc        uint64             `json:"pc"`
	Op        string             `json:"op"`
	Gas       uint64             `json:"gas"`
	GasCost   uint64             `json:"gasCost"`
	Depth     int                `json:"depth"`
	Error     string             `json:"error,omitempty"`
	Stack     *[]string          `json:"stack,omitempty"`
	Memory    *[]string          `json:"memory,omitempty"`
	Storage   *map[string]string `json:"storage,omitempty"`
	ExtraData *ExtraData         `json:"extraData,omitempty"`
}

type ExtraData struct {
	// CREATE | CREATE2: sender address
	From *common.Address `json:"from,omitempty"`
	// CREATE: sender nonce
	Nonce *uint64 `json:"nonce,omitempty"`
	// CALL | CALLCODE | DELEGATECALL | STATICCALL: [tx.to address’s code_hash, stack.nth_last(1) address’s code_hash]
	CodeHashList []common.Hash `json:"codeHashList,omitempty"`
	// SSTORE | SLOAD: [storageProof]
	// SELFDESTRUCT: [contract address’s accountProof, stack.nth_last(0) address’s accountProof]
	// SELFBALANCE: [contract address’s accountProof]
	// BALANCE | EXTCODEHASH: [stack.nth_last(0) address’s accountProof]
	// CREATE | CREATE2: [created contract address’s accountProof]
	// CALL | CALLCODE: [caller contract address’s accountProof, stack.nth_last(1) address’s accountProof]
	ProofList [][]string `json:"proofList,omitempty"`
}

// NewExtraData create, init and return ExtraData
func NewExtraData() *ExtraData {
	return &ExtraData{
		CodeHashList: make([]common.Hash, 0),
		ProofList:    make([][]string, 0),
	}
}

// SealExtraData doesn't show empty fields.
func (e *ExtraData) SealExtraData() *ExtraData {
	if len(e.CodeHashList) == 0 {
		e.CodeHashList = nil
	}
	if len(e.ProofList) == 0 {
		e.ProofList = nil
	}
	if e.From == nil && e.Nonce == nil && e.CodeHashList == nil && e.ProofList == nil {
		return nil
	}
	return e
}
