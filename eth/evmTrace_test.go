package eth

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/core/vm/runtime"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/internal/ethapi"
	"github.com/scroll-tech/go-ethereum/rlp"
)

func traceOps(db ethdb.Database, code []byte) (*ethapi.ExecutionResult, error) {
	newState, err := state.New(common.Hash{}, state.NewDatabase(db), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize new state")
	}
	toAddress := common.Address{0xff}
	newState.SetCode(toAddress, code)
	config := &runtime.Config{
		GasLimit: 1000000,
		State:    newState,
	}
	tracer := vm.NewStructLogger(nil)
	// Overwrite config with tracer
	config.EVMConfig.Debug = true
	config.EVMConfig.Tracer = tracer

	res, _, err := runtime.Call(toAddress, nil, config)
	if err != nil {
		return nil, errors.Wrap(err, "transaction fails")
	}
	return &ethapi.ExecutionResult{
		Gas:         1,
		Failed:      false,
		ReturnValue: fmt.Sprintf("%x", res),
		StructLogs:  ethapi.FormatLogs(tracer.StructLogs()),
	}, nil
}

func TestBlockEvmTracesStorage(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	evmTrace1, err := traceOps(db, []byte{
		// Add slot `0x1` to access list
		byte(vm.PUSH1), 0x01, byte(vm.SLOAD), byte(vm.POP), // SLOAD( 0x1) (add to access list)
		// Write to `0x1` which is already in access list
		byte(vm.PUSH1), 0x11, byte(vm.PUSH1), 0x01, byte(vm.SSTORE), // SSTORE( loc: 0x01, val: 0x11)
		// Write to `0x2` which is not in access list
		byte(vm.PUSH1), 0x11, byte(vm.PUSH1), 0x02, byte(vm.SSTORE), // SSTORE( loc: 0x02, val: 0x11)
		// Write again to `0x2`
		byte(vm.PUSH1), 0x11, byte(vm.PUSH1), 0x02, byte(vm.SSTORE), // SSTORE( loc: 0x02, val: 0x11)
		// Read slot in access list (0x2)
		byte(vm.PUSH1), 0x02, byte(vm.SLOAD), // SLOAD( 0x2)
		// Read slot in access list (0x1)
		byte(vm.PUSH1), 0x01, byte(vm.SLOAD), // SLOAD( 0x1)
	})
	if err != nil {
		t.Fatal("Failed to traceOps", "err", err)
	}
	evmTrace2, err := traceOps(db, []byte{
		byte(vm.PUSH1), 10,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	})
	if err != nil {
		t.Fatal("Failed to traceOps", "err", err)
	}
	evmTraces := []*ethapi.ExecutionResult{evmTrace1, evmTrace2}
	hash := common.BytesToHash([]byte{0x03, 0x04})
	eth := Ethereum{chainDb: db}
	// Insert the evmTrace list into the database and check presence.
	if err := eth.WriteEvmTraces(hash, evmTraces); err != nil {
		t.Fatalf(err.Error())
	}
	// Read evmTrace list from db.
	if traces := eth.ReadEvmTraces(hash); len(traces) == 0 {
		t.Fatalf("No evmTraces returned")
	} else {
		if err := checkEvmTracesRLP(traces, evmTraces); err != nil {
			t.Fatalf(err.Error())
		}
	}
	// Delete evmTrace list by blockHash.
	if err := eth.DeleteEvmTraces(hash); err != nil {
		t.Fatalf(err.Error())
	}
	if traces := eth.ReadEvmTraces(hash); len(traces) != 0 {
		t.Fatalf("The evmTrace list should be empty.")
	}
}

func checkEvmTracesRLP(have, want []*ethapi.ExecutionResult) error {
	if len(have) != len(want) {
		return fmt.Errorf("evmTraces sizes mismatch: have: %d, want: %d", len(have), len(want))
	}
	for i := 0; i < len(want); i++ {
		rlpHave, err := rlp.EncodeToBytes(have[i])
		if err != nil {
			return err
		}
		rlpWant, err := rlp.EncodeToBytes(want[i])
		if err != nil {
			return err
		}
		if !bytes.Equal(rlpHave, rlpWant) {
			return fmt.Errorf("evmTrace #%d: evmTrace mismatch: have %s, want %s", i, hex.EncodeToString(rlpHave), hex.EncodeToString(rlpWant))
		}
	}
	return nil
}
