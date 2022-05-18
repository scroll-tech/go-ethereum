package types

import (
	"encoding/json"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
)

// BlockResult contains block execution traces and results required for rollers.
type BlockResult struct {
	BlockTrace       *BlockTrace        `json:"blockTrace"`
	StorageTrace     *StorageTrace      `json:"storageTrace"`
	ExecutionResults []*ExecutionResult `json:"executionResults"`
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

	// post-handled output for ready-to-use trace
	SMTTrace []*StateTrace `json:"smtTrace,omitempty"`
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         uint64 `json:"gas"`
	Failed      bool   `json:"failed"`
	ReturnValue string `json:"returnValue,omitempty"`
	// Sender's account data (before Tx).
	Sender *AccountWrapper `json:"sender,omitempty"`

	// AccountCreated record the account in case tx is create
	// (for creating inside contracts we handle CREATE op)
	AccountCreated *AccountWrapper `json:"accountCreated,omitempty"`

	// Record all accounts' state which would be affected AFTER tx executed
	// currently they are just sender and to account
	AccountsAfter []*AccountWrapper `json:"accountAfter"`

	// It's exist only when tx is a contract call.
	CodeHash *common.Hash `json:"codeHash,omitempty"`
	// If it is a contract call, the contract code is returned.
	ByteCode string `json:"byteCode,omitempty"`

	// Deprecated: The account's proof.
	// Proof      []string       `json:"proof,omitempty"`
	StructLogs []StructLogRes `json:"structLogs"`
}

// HexInt wrap big.Int for hex encoding
type HexInt struct {
	*big.Int
}

// MarshalText implements encoding.TextMarshaler
func (hi HexInt) MarshalJSON() ([]byte, error) {
	if hi.Int == nil {
		return json.Marshal("0x")
	}
	return json.Marshal(hexutil.Encode(hi.Bytes()))
}

// UnmarshalJSON implements json.Unmarshaler.
func (hi *HexInt) UnmarshalJSON(input []byte) error {

	var s string
	if err := json.Unmarshal(input, &s); err != nil {
		return err
	}

	hi.Int, _ = big.NewInt(0).SetString(string(input), 0)
	return nil
}

// SMTPathNode represent a node in the SMT Path
type SMTPathNode struct {
	Value   hexutil.Bytes `json:"value"`
	Sibling hexutil.Bytes `json:"sibling"`
}

// SMTPath is the whole path of SMT
type SMTPath struct {
	KeyPathPart HexInt        `json:"pathPart"` //the path part in key
	Root        hexutil.Bytes `json:"root"`
	Path        []SMTPathNode `json:"path,omitempty"` //path start from top
	Leaf        *SMTPathNode  `json:"leaf,omitempty"` //would be omitted for empty leaf, the sibling indicate key
}

// StateAccountL2 is the represent of StateAccount in L2 circuit
// Notice in L2 we have different hash scheme against StateAccount.MarshalByte
type StateAccountL2 struct {
	Nonce    int           `json:"nonce"`
	Balance  HexInt        `json:"balance"` //just the common hex expression of integer (big-endian)
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

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc            uint64             `json:"pc"`
	Op            string             `json:"op"`
	Gas           uint64             `json:"gas"`
	GasCost       uint64             `json:"gasCost"`
	Depth         int                `json:"depth"`
	Error         string             `json:"error,omitempty"`
	Stack         *[]string          `json:"stack,omitempty"`
	Memory        *[]string          `json:"memory,omitempty"`
	Storage       *map[string]string `json:"storage,omitempty"`
	RefundCounter uint64             `json:"refund,omitempty"`
	ExtraData     *ExtraData         `json:"extraData,omitempty"`
}

type ExtraData struct {
	// Indicate the call success or not for CALL/CREATE op
	CallFailed bool `json:"callFailed,omitempty"`
	// CALL | CALLCODE | DELEGATECALL | STATICCALL: [tx.to address’s code, stack.nth_last(1) address’s code]
	CodeList [][]byte `json:"codeList,omitempty"`
	// SSTORE | SLOAD: [storageProof]
	// SELFDESTRUCT: [contract address’s accountProof, stack.nth_last(0) address’s accountProof]
	// SELFBALANCE: [contract address’s accountProof]
	// BALANCE | EXTCODEHASH: [stack.nth_last(0) address’s accountProof]
	// CREATE | CREATE2: [created contract address’s accountProof (before constructed),
	// 					  created contract address's data (after constructed)]
	// CALL | CALLCODE: [caller contract address’s accountProof, stack.nth_last(1) (i.e. called) address’s accountProof
	//					  called contract address's data (value updated, before called)]
	// STATICCALL: [stack.nth_last(1) (i.e. called) address’s accountProof
	//					  called contract address's data (before called)]
	ProofList []*AccountWrapper `json:"proofList,omitempty"`
}

type AccountWrapper struct {
	Address  common.Address `json:"address"`
	Nonce    uint64         `json:"nonce"`
	Balance  *hexutil.Big   `json:"balance"`
	CodeHash common.Hash    `json:"codeHash,omitempty"`
	//Proof    []string             `json:"proof,omitempty"`
	Storage *StorageWrapper `json:"storage,omitempty"` // StorageProofWrapper can be empty if irrelated to storage operation
}

// while key & value can also be retrieved from StructLogRes.Storage,
// we still stored in here for roller's processing convenience.
type StorageWrapper struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	//Proof []string `json:"proof,omitempty"`
}

// NewExtraData create, init and return ExtraData
func NewExtraData() *ExtraData {
	return &ExtraData{
		CodeList:  make([][]byte, 0),
		ProofList: make([]*AccountWrapper, 0),
	}
}

func (e *ExtraData) Clean() {
	e.CodeList = e.CodeList[:0]
	e.ProofList = e.ProofList[:0]
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
