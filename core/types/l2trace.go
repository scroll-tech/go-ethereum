package types

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
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
	// Sender's account proof.
	Sender *AccountProofWrapper `json:"sender,omitempty"`

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
	Value    hexutil.Bytes `json:"value"`
	Silbling hexutil.Bytes `json:"silbiling"`
}

// SMTPath is the whole path of SMT
type SMTPath struct {
	Root hexutil.Bytes `json:"root"`
	Path []SMTPathNode `json:"path"`           //path start from top
	Leaf *SMTPathNode  `json:"leaf,omitempty"` //would be omitted for empty leaf
}

// StateAccountL2 is the represent of StateAccount in L2 circuit
// Notice in L2 we have different hash scheme against StateAccount.MarshalByte
type StateAccountL2 struct {
	Nonce    int           `json:"nonce"`
	Balance  hexutil.Bytes `json:"balance"` //just the common hex expression of integer (big-endian)
	CodeHash hexutil.Bytes `json:"codeHash,omitempty"`
}

// StateStorageL2 is the represent of a stored key-value pair for specified account
type StateStorageL2 struct {
	Key   hexutil.Bytes `json:"key"` //notice this is the preimage of storage key
	Value hexutil.Bytes `json:"value"`
}

// StateTrace record the updating on state trie and (if changed) account trie
// represent by the [before, after] updating of SMTPath amont tries and Account
type StateTrace struct {
	// which log the trace is responded for, -1 indicate not caused
	// by opcode (like gasRefund, coinbase, setNonce, etc)
	Index           int                `json:"index"`
	Address         hexutil.Bytes      `json:"address"`
	AccountKey      hexutil.Bytes      `json:"accountKey"`
	AccountPath     [2]*SMTPath        `json:"accountPath"`
	AccountUpdate   [2]*StateAccountL2 `json:"accountUpdate"`
	StateKey        hexutil.Bytes      `json:"stateKey,omitempty"`
	CommonStateRoot hexutil.Bytes      `json:"commonStateRoot,omitempty"` //CommonStateRoot is used if there is no state update
	StatePath       [2]*SMTPath        `json:"statePath,omitempty"`
	StateUpdate     [2]*StateStorageL2 `json:"stateUpdate,omitempty"`
}

// StorageRes stores data required in storage circuit
type StorageRes struct {

	// Root hash before execution:
	RootBefore *common.Hash `json:"rootBefore,omitempty"`
	// Root hash after execution, is nil if execution has failed
	RootAfter *common.Hash `json:"rootAfter,omitempty"`
	// AccountsAfter recode and encoded all accounts
	AccountsAfter map[string]hexutil.Bytes `json:"accountAfter"`

	// The from account's proof BEFORE execution
	ProofFrom []hexutil.Bytes `json:"proofFrom,omitempty"`
	// The to account's proof BEFORE execution, these proof,
	// along with account proof's inside structLogs, form the
	// dataset required by tracing the updates of account trie
	ProofTo []hexutil.Bytes `json:"proofTo,omitempty"`

	// The To Address, would be valid even when tx is creation
	ToAddress common.Address `json:"to"`
	// AccountCreated record the account in case tx is create
	// (for creating inside contracts we handle CREATE op)
	AccountCreated hexutil.Bytes `json:"accountCreated,omitempty"`

	SMTTrace []*StateTrace `json:"smtTrace,omitempty"`
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
	// CALL | CALLCODE | DELEGATECALL | STATICCALL: [tx.to address’s code, stack.nth_last(1) address’s code]
	CodeList [][]byte `json:"codeList,omitempty"`
	// SSTORE | SLOAD: [storageProof]
	// SELFDESTRUCT: [contract address’s accountProof, stack.nth_last(0) address’s accountProof]
	// SELFBALANCE: [contract address’s accountProof]
	// BALANCE | EXTCODEHASH: [stack.nth_last(0) address’s accountProof]
	// CREATE | CREATE2: [sender's accountProof, created contract address’s accountProof]
	// CALL | CALLCODE: [caller contract address’s accountProof, stack.nth_last(1) address’s accountProof]
	ProofList []*AccountProofWrapper `json:"proofList,omitempty"`
}

type AccountProofWrapper struct {
	Address common.Address       `json:"address,omitempty"`
	Nonce   uint64               `json:"nonce,omitempty"`
	Balance *big.Int             `json:"balance,omitempty"`
	Proof   []string             `json:"proof,omitempty"`
	Storage *StorageProofWrapper `json:"storage,omitempty"` // StorageProofWrapper can be empty if irrelated to storage operation
}

// while key & value can also be retrieved from StructLogRes.Storage,
// we still stored in here for roller's processing convenience.
type StorageProofWrapper struct {
	Key   string   `json:"key,omitempty"`
	Value string   `json:"value,omitempty"`
	Proof []string `json:"proof,omitempty"`
}

// NewExtraData create, init and return ExtraData
func NewExtraData() *ExtraData {
	return &ExtraData{
		CodeList:  make([][]byte, 0),
		ProofList: make([]*AccountProofWrapper, 0),
	}
}

// SealExtraData doesn't show empty fields.
func (e *ExtraData) SealExtraData() *ExtraData {
	if len(e.CodeList) == 0 {
		e.CodeList = nil
	}
	if len(e.ProofList) == 0 {
		e.ProofList = nil
	}
	if e.CodeList == nil && e.ProofList == nil {
		return nil
	}
	return e
}
