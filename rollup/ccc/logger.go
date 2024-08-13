package ccc

import (
	"maps"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
)

var _ vm.EVMLogger = (*Logger)(nil)

type Logger struct {
	limitPerCircuit uint64

	currentEnv    *vm.EVM
	codesAccessed map[common.Hash]bool

	evmUsage      uint64
	stateUsage    uint64
	bytecodeUsage uint64
}

func NewLogger(limitPerCircuit uint64) *Logger {
	return &Logger{
		limitPerCircuit: limitPerCircuit,
		codesAccessed:   make(map[common.Hash]bool),
	}
}

// Snapshot creates an independent copy of the logger
func (l *Logger) Snapshot() *Logger {
	newL := *l
	newL.codesAccessed = maps.Clone(newL.codesAccessed)
	return &newL
}

func (l *Logger) logBytecodeAccessAt(addr common.Address) {
	codeHash := l.currentEnv.StateDB.GetKeccakCodeHash(addr)
	if codeHash != (common.Hash{}) && !l.codesAccessed[codeHash] {
		l.bytecodeUsage += l.currentEnv.StateDB.GetCodeSize(addr)
		l.codesAccessed[codeHash] = true
	}
}

func (l *Logger) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	l.currentEnv = env
	if !create {
		l.logBytecodeAccessAt(to)
	}
}

func (l *Logger) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	l.evmUsage += evmUsagePerOpCode[op]
	l.stateUsage += stateUsagePerOpCode[op](scope, depth)

	switch op {
	case vm.EXTCODECOPY:
		l.logBytecodeAccessAt(common.Address(scope.Stack.Back(0).Bytes20()))
	case vm.DELEGATECALL, vm.CALL, vm.STATICCALL, vm.CALLCODE:
		l.logBytecodeAccessAt(common.Address(scope.Stack.Back(1).Bytes20()))
	}
}

func (l *Logger) CaptureStateAfter(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {

}

func (l *Logger) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {

}

func (l *Logger) CaptureExit(output []byte, gasUsed uint64, err error) {

}

func (l *Logger) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {

}

func (l *Logger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {

}

// Error returns an error if executed txns triggered an overflow
// Caller should revert some transactions and close the block
func (l *Logger) Error() error {
	if l.evmUsage > l.limitPerCircuit || l.stateUsage > l.limitPerCircuit || l.bytecodeUsage > l.limitPerCircuit {
		return ErrBlockRowConsumptionOverflow
	}
	return nil
}

// RowConsumption returns the accumulated resource utilization for each subcircuit so far
func (l *Logger) RowConsumption() types.RowConsumption {
	return types.RowConsumption{
		{
			Name:      "evm",
			RowNumber: l.evmUsage,
		}, {
			Name:      "state",
			RowNumber: l.stateUsage,
		}, {
			Name:      "bytecode",
			RowNumber: l.bytecodeUsage,
		},
	}
}

// evm circuit resource usage per OpCode
var evmUsagePerOpCode = [256]uint64{
	2,  // STOP (0)
	3,  // ADD (1)
	4,  // MUL (2)
	3,  // SUB (3)
	4,  // DIV (4)
	10, // SDIV (5)
	4,  // MOD (6)
	10, // SMOD (7)
	9,  // ADDMOD (8)
	10, // MULMOD (9)
	3,  // EXP (10)
	2,  // SIGNEXTEND (11)
	0,  // UNDEFINED (12)
	0,  // UNDEFINED (13)
	0,  // UNDEFINED (14)
	0,  // UNDEFINED (15)
	3,  // LT (16)
	3,  // GT (17)
	3,  // SLT (18)
	3,  // SGT (19)
	3,  // EQ (20)
	1,  // ISZERO (21)
	4,  // AND (22)
	4,  // OR (23)
	4,  // XOR (24)
	4,  // NOT (25)
	2,  // BYTE (26)
	5,  // SHL (27)
	5,  // SHR (28)
	5,  // SAR (29)
	0,  // UNDEFINED (30)
	0,  // UNDEFINED (31)
	2,  // SHA3 (32)
	0,  // UNDEFINED (33)
	0,  // UNDEFINED (34)
	0,  // UNDEFINED (35)
	0,  // UNDEFINED (36)
	0,  // UNDEFINED (37)
	0,  // UNDEFINED (38)
	0,  // UNDEFINED (39)
	0,  // UNDEFINED (40)
	0,  // UNDEFINED (41)
	0,  // UNDEFINED (42)
	0,  // UNDEFINED (43)
	0,  // UNDEFINED (44)
	0,  // UNDEFINED (45)
	0,  // UNDEFINED (46)
	0,  // UNDEFINED (47)
	1,  // ADDRESS (48)
	2,  // BALANCE (49)
	1,  // ORIGIN (50)
	1,  // CALLER (51)
	1,  // CALLVALUE (52)
	8,  // CALLDATALOAD (53)
	1,  // CALLDATASIZE (54)
	2,  // CALLDATACOPY (55)
	2,  // CODESIZE (56)
	2,  // CODECOPY (57)
	1,  // GASPRICE (58)
	2,  // EXTCODESIZE (59)
	3,  // EXTCODECOPY (60)
	1,  // RETURNDATASIZE (61)
	4,  // RETURNDATACOPY (62)
	1,  // EXTCODEHASH (63)
	3,  // BLOCKHASH (64)
	1,  // COINBASE (65)
	1,  // TIMESTAMP (66)
	1,  // NUMBER (67)
	1,  // DIFFICULTY (68)
	1,  // GASLIMIT (69)
	1,  // CHAINID (70)
	1,  // SELFBALANCE (71)
	1,  // BASEFEE (72)
	0,  // UNDEFINED (73)
	0,  // UNDEFINED (74)
	0,  // UNDEFINED (75)
	0,  // UNDEFINED (76)
	0,  // UNDEFINED (77)
	0,  // UNDEFINED (78)
	0,  // UNDEFINED (79)
	1,  // POP (80)
	5,  // MLOAD (81)
	5,  // MSTORE (82)
	5,  // MSTORE8 (83)
	2,  // SLOAD (84)
	3,  // SSTORE (85)
	2,  // JUMP (86)
	2,  // JUMPI (87)
	1,  // PC (88)
	1,  // MSIZE (89)
	1,  // GAS (90)
	1,  // JUMPDEST (91)
	2,  // TLOAD (92)
	3,  // TSTORE (93)
	2,  // MCOPY (94)
	1,  // PUSH0 (95)
	1,  // PUSH1 (96)
	1,  // PUSH2 (97)
	1,  // PUSH3 (98)
	1,  // PUSH4 (99)
	1,  // PUSH5 (100)
	1,  // PUSH6 (101)
	1,  // PUSH7 (102)
	1,  // PUSH8 (103)
	1,  // PUSH9 (104)
	1,  // PUSH10 (105)
	1,  // PUSH11 (106)
	1,  // PUSH12 (107)
	1,  // PUSH13 (108)
	1,  // PUSH14 (109)
	1,  // PUSH15 (110)
	1,  // PUSH16 (111)
	1,  // PUSH17 (112)
	1,  // PUSH18 (113)
	1,  // PUSH19 (114)
	1,  // PUSH20 (115)
	1,  // PUSH21 (116)
	1,  // PUSH22 (117)
	1,  // PUSH23 (118)
	1,  // PUSH24 (119)
	1,  // PUSH25 (120)
	1,  // PUSH26 (121)
	1,  // PUSH27 (122)
	1,  // PUSH28 (123)
	1,  // PUSH29 (124)
	1,  // PUSH30 (125)
	1,  // PUSH31 (126)
	1,  // PUSH32 (127)
	1,  // DUP1 (128)
	1,  // DUP2 (129)
	1,  // DUP3 (130)
	1,  // DUP4 (131)
	1,  // DUP5 (132)
	1,  // DUP6 (133)
	1,  // DUP7 (134)
	1,  // DUP8 (135)
	1,  // DUP9 (136)
	1,  // DUP10 (137)
	1,  // DUP11 (138)
	1,  // DUP12 (139)
	1,  // DUP13 (140)
	1,  // DUP14 (141)
	1,  // DUP15 (142)
	1,  // DUP16 (143)
	1,  // SWAP1 (144)
	1,  // SWAP2 (145)
	1,  // SWAP3 (146)
	1,  // SWAP4 (147)
	1,  // SWAP5 (148)
	1,  // SWAP6 (149)
	1,  // SWAP7 (150)
	1,  // SWAP8 (151)
	1,  // SWAP9 (152)
	1,  // SWAP10 (153)
	1,  // SWAP11 (154)
	1,  // SWAP12 (155)
	1,  // SWAP13 (156)
	1,  // SWAP14 (157)
	1,  // SWAP15 (158)
	1,  // SWAP16 (159)
	2,  // LOG0 (160)
	2,  // LOG1 (161)
	2,  // LOG2 (162)
	2,  // LOG3 (163)
	2,  // LOG4 (164)
	0,  // UNDEFINED (165)
	0,  // UNDEFINED (166)
	0,  // UNDEFINED (167)
	0,  // UNDEFINED (168)
	0,  // UNDEFINED (169)
	0,  // UNDEFINED (170)
	0,  // UNDEFINED (171)
	0,  // UNDEFINED (172)
	0,  // UNDEFINED (173)
	0,  // UNDEFINED (174)
	0,  // UNDEFINED (175)
	0,  // UNDEFINED (176)
	0,  // UNDEFINED (177)
	0,  // UNDEFINED (178)
	0,  // UNDEFINED (179)
	0,  // UNDEFINED (180)
	0,  // UNDEFINED (181)
	0,  // UNDEFINED (182)
	0,  // UNDEFINED (183)
	0,  // UNDEFINED (184)
	0,  // UNDEFINED (185)
	0,  // UNDEFINED (186)
	0,  // UNDEFINED (187)
	0,  // UNDEFINED (188)
	0,  // UNDEFINED (189)
	0,  // UNDEFINED (190)
	0,  // UNDEFINED (191)
	0,  // UNDEFINED (192)
	0,  // UNDEFINED (193)
	0,  // UNDEFINED (194)
	0,  // UNDEFINED (195)
	0,  // UNDEFINED (196)
	0,  // UNDEFINED (197)
	0,  // UNDEFINED (198)
	0,  // UNDEFINED (199)
	0,  // UNDEFINED (200)
	0,  // UNDEFINED (201)
	0,  // UNDEFINED (202)
	0,  // UNDEFINED (203)
	0,  // UNDEFINED (204)
	0,  // UNDEFINED (205)
	0,  // UNDEFINED (206)
	0,  // UNDEFINED (207)
	0,  // UNDEFINED (208)
	0,  // UNDEFINED (209)
	0,  // UNDEFINED (210)
	0,  // UNDEFINED (211)
	0,  // UNDEFINED (212)
	0,  // UNDEFINED (213)
	0,  // UNDEFINED (214)
	0,  // UNDEFINED (215)
	0,  // UNDEFINED (216)
	0,  // UNDEFINED (217)
	0,  // UNDEFINED (218)
	0,  // UNDEFINED (219)
	0,  // UNDEFINED (220)
	0,  // UNDEFINED (221)
	0,  // UNDEFINED (222)
	0,  // UNDEFINED (223)
	0,  // UNDEFINED (224)
	0,  // UNDEFINED (225)
	0,  // UNDEFINED (226)
	0,  // UNDEFINED (227)
	0,  // UNDEFINED (228)
	0,  // UNDEFINED (229)
	0,  // UNDEFINED (230)
	0,  // UNDEFINED (231)
	0,  // UNDEFINED (232)
	0,  // UNDEFINED (233)
	0,  // UNDEFINED (234)
	0,  // UNDEFINED (235)
	0,  // UNDEFINED (236)
	0,  // UNDEFINED (237)
	0,  // UNDEFINED (238)
	0,  // UNDEFINED (239)
	9,  // CREATE (240)
	12, // CALL (241)
	12, // CALLCODE (242)
	4,  // RETURN (243)
	12, // DELEGATECALL (244)
	9,  // CREATE2 (245)
	0,  // UNDEFINED (246)
	0,  // UNDEFINED (247)
	0,  // UNDEFINED (248)
	0,  // UNDEFINED (249)
	12, // STATICCALL (250)
	0,  // UNDEFINED (251)
	0,  // UNDEFINED (252)
	4,  // REVERT (253)
	0,  // INVALID (254)
	0,  // SELFDESTRUCT (255)
}

func constantStateUsage(usage uint64) func(*vm.ScopeContext, int) uint64 {
	return func(_ *vm.ScopeContext, _ int) uint64 {
		return usage
	}
}

func logStateUsage(size uint64) func(*vm.ScopeContext, int) uint64 {
	return func(scope *vm.ScopeContext, _ int) uint64 {
		return 2*(scope.Stack.Back(1).Uint64()/32) + 7 + 2*size
	}
}

// state circuit resource usage per OpCode
var stateUsagePerOpCode = [256]func(*vm.ScopeContext, int) uint64{
	constantStateUsage(13), // STOP (0)
	constantStateUsage(3),  // ADD (1)
	constantStateUsage(3),  // MUL (2)
	constantStateUsage(3),  // SUB (3)
	constantStateUsage(3),  // DIV (4)
	constantStateUsage(3),  // SDIV (5)
	constantStateUsage(3),  // MOD (6)
	constantStateUsage(3),  // SMOD (7)
	constantStateUsage(4),  // ADDMOD (8)
	constantStateUsage(4),  // MULMOD (9)
	constantStateUsage(3),  // EXP (10)
	constantStateUsage(3),  // SIGNEXTEND (11)
	constantStateUsage(0),  // UNDEFINED (12)
	constantStateUsage(0),  // UNDEFINED (13)
	constantStateUsage(0),  // UNDEFINED (14)
	constantStateUsage(0),  // UNDEFINED (15)
	constantStateUsage(3),  // LT (16)
	constantStateUsage(3),  // GT (17)
	constantStateUsage(3),  // SLT (18)
	constantStateUsage(3),  // SGT (19)
	constantStateUsage(3),  // EQ (20)
	constantStateUsage(2),  // ISZERO (21)
	constantStateUsage(3),  // AND (22)
	constantStateUsage(3),  // OR (23)
	constantStateUsage(3),  // XOR (24)
	constantStateUsage(2),  // NOT (25)
	constantStateUsage(3),  // BYTE (26)
	constantStateUsage(3),  // SHL (27)
	constantStateUsage(3),  // SHR (28)
	constantStateUsage(3),  // SAR (29)
	constantStateUsage(0),  // UNDEFINED (30)
	constantStateUsage(0),  // UNDEFINED (31)
	func(scope *vm.ScopeContext, _ int) uint64 {
		// let n = # bytes, then row_consumption = (n/32) + 3
		return scope.Stack.Back(1).Uint64()/32 + 3
	}, // SHA3 (32)
	constantStateUsage(0), // UNDEFINED (33)
	constantStateUsage(0), // UNDEFINED (34)
	constantStateUsage(0), // UNDEFINED (35)
	constantStateUsage(0), // UNDEFINED (36)
	constantStateUsage(0), // UNDEFINED (37)
	constantStateUsage(0), // UNDEFINED (38)
	constantStateUsage(0), // UNDEFINED (39)
	constantStateUsage(0), // UNDEFINED (40)
	constantStateUsage(0), // UNDEFINED (41)
	constantStateUsage(0), // UNDEFINED (42)
	constantStateUsage(0), // UNDEFINED (43)
	constantStateUsage(0), // UNDEFINED (44)
	constantStateUsage(0), // UNDEFINED (45)
	constantStateUsage(0), // UNDEFINED (46)
	constantStateUsage(0), // UNDEFINED (47)
	constantStateUsage(2), // ADDRESS (48)
	constantStateUsage(7), // BALANCE (49)
	constantStateUsage(2), // ORIGIN (50)
	constantStateUsage(2), // CALLER (51)
	constantStateUsage(2), // CALLVALUE (52)
	constantStateUsage(7), // CALLDATALOAD (53)
	constantStateUsage(2), // CALLDATASIZE (54)
	func(scope *vm.ScopeContext, depth int) uint64 {
		// let n = # bytes in calldata, then row_consumption = (n/32)*2 + (is_root? 5 : 6)
		constant := uint64(5)
		if depth != 0 {
			constant = 6
		}
		return 2*(scope.Stack.Back(2).Uint64()/32) + constant
	}, // CALLDATACOPY (55)
	constantStateUsage(1), // CODESIZE (56)
	func(scope *vm.ScopeContext, _ int) uint64 {
		// let n = # bytes in code, then row_consumption = (n/32) + 3
		return scope.Stack.Back(2).Uint64()/32 + 3
	}, // CODECOPY (57)
	constantStateUsage(2), // GASPRICE (58)
	constantStateUsage(7), // EXTCODESIZE (59)
	func(scope *vm.ScopeContext, _ int) uint64 {
		// let n = # bytes in code, then row_consumption = (n/32) + 9
		return scope.Stack.Back(3).Uint64()/32 + 3
	}, // EXTCODECOPY (60)
	constantStateUsage(2), // RETURNDATASIZE (61)
	func(scope *vm.ScopeContext, _ int) uint64 {
		// let n = # of bytes to return, then row_consumption = (n/32)*2 + 6
		return 2*(scope.Stack.Back(2).Uint64()/32) + 6
	}, // RETURNDATACOPY (62)
	constantStateUsage(7),   // EXTCODEHASH (63)
	constantStateUsage(2),   // BLOCKHASH (64)
	constantStateUsage(1),   // COINBASE (65)
	constantStateUsage(1),   // TIMESTAMP (66)
	constantStateUsage(1),   // NUMBER (67)
	constantStateUsage(1),   // DIFFICULTY (68)
	constantStateUsage(1),   // GASLIMIT (69)
	constantStateUsage(1),   // CHAINID (70)
	constantStateUsage(3),   // SELFBALANCE (71)
	constantStateUsage(1),   // BASEFEE (72)
	constantStateUsage(0),   // UNDEFINED (73)
	constantStateUsage(0),   // UNDEFINED (74)
	constantStateUsage(0),   // UNDEFINED (75)
	constantStateUsage(0),   // UNDEFINED (76)
	constantStateUsage(0),   // UNDEFINED (77)
	constantStateUsage(0),   // UNDEFINED (78)
	constantStateUsage(0),   // UNDEFINED (79)
	constantStateUsage(1),   // POP (80)
	constantStateUsage(4),   // MLOAD (81)
	constantStateUsage(4),   // MSTORE (82)
	constantStateUsage(3),   // MSTORE8 (83)
	constantStateUsage(9),   // SLOAD (84)
	constantStateUsage(11),  // SSTORE (85)
	constantStateUsage(1),   // JUMP (86)
	constantStateUsage(2),   // JUMPI (87)
	constantStateUsage(1),   // PC (88)
	constantStateUsage(1),   // MSIZE (89)
	constantStateUsage(1),   // GAS (90)
	constantStateUsage(0),   // JUMPDEST (91)
	constantStateUsage(5),   // TLOAD (92)
	constantStateUsage(8),   // TSTORE (93)
	constantStateUsage(7),   // MCOPY (94)
	constantStateUsage(1),   // PUSH0 (95)
	constantStateUsage(1),   // PUSH1 (96)
	constantStateUsage(1),   // PUSH2 (97)
	constantStateUsage(1),   // PUSH3 (98)
	constantStateUsage(1),   // PUSH4 (99)
	constantStateUsage(1),   // PUSH5 (100)
	constantStateUsage(1),   // PUSH6 (101)
	constantStateUsage(1),   // PUSH7 (102)
	constantStateUsage(1),   // PUSH8 (103)
	constantStateUsage(1),   // PUSH9 (104)
	constantStateUsage(1),   // PUSH10 (105)
	constantStateUsage(1),   // PUSH11 (106)
	constantStateUsage(1),   // PUSH12 (107)
	constantStateUsage(1),   // PUSH13 (108)
	constantStateUsage(1),   // PUSH14 (109)
	constantStateUsage(1),   // PUSH15 (110)
	constantStateUsage(1),   // PUSH16 (111)
	constantStateUsage(1),   // PUSH17 (112)
	constantStateUsage(1),   // PUSH18 (113)
	constantStateUsage(1),   // PUSH19 (114)
	constantStateUsage(1),   // PUSH20 (115)
	constantStateUsage(1),   // PUSH21 (116)
	constantStateUsage(1),   // PUSH22 (117)
	constantStateUsage(1),   // PUSH23 (118)
	constantStateUsage(1),   // PUSH24 (119)
	constantStateUsage(1),   // PUSH25 (120)
	constantStateUsage(1),   // PUSH26 (121)
	constantStateUsage(1),   // PUSH27 (122)
	constantStateUsage(1),   // PUSH28 (123)
	constantStateUsage(1),   // PUSH29 (124)
	constantStateUsage(1),   // PUSH30 (125)
	constantStateUsage(1),   // PUSH31 (126)
	constantStateUsage(1),   // PUSH32 (127)
	constantStateUsage(2),   // DUP1 (128)
	constantStateUsage(2),   // DUP2 (129)
	constantStateUsage(2),   // DUP3 (130)
	constantStateUsage(2),   // DUP4 (131)
	constantStateUsage(2),   // DUP5 (132)
	constantStateUsage(2),   // DUP6 (133)
	constantStateUsage(2),   // DUP7 (134)
	constantStateUsage(2),   // DUP8 (135)
	constantStateUsage(2),   // DUP9 (136)
	constantStateUsage(2),   // DUP10 (137)
	constantStateUsage(2),   // DUP11 (138)
	constantStateUsage(2),   // DUP12 (139)
	constantStateUsage(2),   // DUP13 (140)
	constantStateUsage(2),   // DUP14 (141)
	constantStateUsage(2),   // DUP15 (142)
	constantStateUsage(2),   // DUP16 (143)
	constantStateUsage(4),   // SWAP1 (144)
	constantStateUsage(4),   // SWAP2 (145)
	constantStateUsage(4),   // SWAP3 (146)
	constantStateUsage(4),   // SWAP4 (147)
	constantStateUsage(4),   // SWAP5 (148)
	constantStateUsage(4),   // SWAP6 (149)
	constantStateUsage(4),   // SWAP7 (150)
	constantStateUsage(4),   // SWAP8 (151)
	constantStateUsage(4),   // SWAP9 (152)
	constantStateUsage(4),   // SWAP10 (153)
	constantStateUsage(4),   // SWAP11 (154)
	constantStateUsage(4),   // SWAP12 (155)
	constantStateUsage(4),   // SWAP13 (156)
	constantStateUsage(4),   // SWAP14 (157)
	constantStateUsage(4),   // SWAP15 (158)
	constantStateUsage(4),   // SWAP16 (159)
	logStateUsage(0),        // LOG0 (160)
	logStateUsage(1),        // LOG1 (161)
	logStateUsage(2),        // LOG2 (162)
	logStateUsage(3),        // LOG3 (163)
	logStateUsage(4),        // LOG4 (164)
	constantStateUsage(0),   // UNDEFINED (165)
	constantStateUsage(0),   // UNDEFINED (166)
	constantStateUsage(0),   // UNDEFINED (167)
	constantStateUsage(0),   // UNDEFINED (168)
	constantStateUsage(0),   // UNDEFINED (169)
	constantStateUsage(0),   // UNDEFINED (170)
	constantStateUsage(0),   // UNDEFINED (171)
	constantStateUsage(0),   // UNDEFINED (172)
	constantStateUsage(0),   // UNDEFINED (173)
	constantStateUsage(0),   // UNDEFINED (174)
	constantStateUsage(0),   // UNDEFINED (175)
	constantStateUsage(0),   // UNDEFINED (176)
	constantStateUsage(0),   // UNDEFINED (177)
	constantStateUsage(0),   // UNDEFINED (178)
	constantStateUsage(0),   // UNDEFINED (179)
	constantStateUsage(0),   // UNDEFINED (180)
	constantStateUsage(0),   // UNDEFINED (181)
	constantStateUsage(0),   // UNDEFINED (182)
	constantStateUsage(0),   // UNDEFINED (183)
	constantStateUsage(0),   // UNDEFINED (184)
	constantStateUsage(0),   // UNDEFINED (185)
	constantStateUsage(0),   // UNDEFINED (186)
	constantStateUsage(0),   // UNDEFINED (187)
	constantStateUsage(0),   // UNDEFINED (188)
	constantStateUsage(0),   // UNDEFINED (189)
	constantStateUsage(0),   // UNDEFINED (190)
	constantStateUsage(0),   // UNDEFINED (191)
	constantStateUsage(0),   // UNDEFINED (192)
	constantStateUsage(0),   // UNDEFINED (193)
	constantStateUsage(0),   // UNDEFINED (194)
	constantStateUsage(0),   // UNDEFINED (195)
	constantStateUsage(0),   // UNDEFINED (196)
	constantStateUsage(0),   // UNDEFINED (197)
	constantStateUsage(0),   // UNDEFINED (198)
	constantStateUsage(0),   // UNDEFINED (199)
	constantStateUsage(0),   // UNDEFINED (200)
	constantStateUsage(0),   // UNDEFINED (201)
	constantStateUsage(0),   // UNDEFINED (202)
	constantStateUsage(0),   // UNDEFINED (203)
	constantStateUsage(0),   // UNDEFINED (204)
	constantStateUsage(0),   // UNDEFINED (205)
	constantStateUsage(0),   // UNDEFINED (206)
	constantStateUsage(0),   // UNDEFINED (207)
	constantStateUsage(0),   // UNDEFINED (208)
	constantStateUsage(0),   // UNDEFINED (209)
	constantStateUsage(0),   // UNDEFINED (210)
	constantStateUsage(0),   // UNDEFINED (211)
	constantStateUsage(0),   // UNDEFINED (212)
	constantStateUsage(0),   // UNDEFINED (213)
	constantStateUsage(0),   // UNDEFINED (214)
	constantStateUsage(0),   // UNDEFINED (215)
	constantStateUsage(0),   // UNDEFINED (216)
	constantStateUsage(0),   // UNDEFINED (217)
	constantStateUsage(0),   // UNDEFINED (218)
	constantStateUsage(0),   // UNDEFINED (219)
	constantStateUsage(0),   // UNDEFINED (220)
	constantStateUsage(0),   // UNDEFINED (221)
	constantStateUsage(0),   // UNDEFINED (222)
	constantStateUsage(0),   // UNDEFINED (223)
	constantStateUsage(0),   // UNDEFINED (224)
	constantStateUsage(0),   // UNDEFINED (225)
	constantStateUsage(0),   // UNDEFINED (226)
	constantStateUsage(0),   // UNDEFINED (227)
	constantStateUsage(0),   // UNDEFINED (228)
	constantStateUsage(0),   // UNDEFINED (229)
	constantStateUsage(0),   // UNDEFINED (230)
	constantStateUsage(0),   // UNDEFINED (231)
	constantStateUsage(0),   // UNDEFINED (232)
	constantStateUsage(0),   // UNDEFINED (233)
	constantStateUsage(0),   // UNDEFINED (234)
	constantStateUsage(0),   // UNDEFINED (235)
	constantStateUsage(0),   // UNDEFINED (236)
	constantStateUsage(0),   // UNDEFINED (237)
	constantStateUsage(0),   // UNDEFINED (238)
	constantStateUsage(0),   // UNDEFINED (239)
	constantStateUsage(42),  // CREATE (240)
	constantStateUsage(26),  // CALL (241)
	constantStateUsage(22),  // CALLCODE (242)
	constantStateUsage(273), // RETURN (243)
	constantStateUsage(23),  // DELEGATECALL (244)
	constantStateUsage(43),  // CREATE2 (245)
	constantStateUsage(0),   // UNDEFINED (246)
	constantStateUsage(0),   // UNDEFINED (247)
	constantStateUsage(0),   // UNDEFINED (248)
	constantStateUsage(0),   // UNDEFINED (249)
	constantStateUsage(21),  // STATICCALL (250)
	constantStateUsage(0),   // UNDEFINED (251)
	constantStateUsage(0),   // UNDEFINED (252)
	constantStateUsage(274), // REVERT (253)
	constantStateUsage(0),   // INVALID (254)
	constantStateUsage(0),   // SELFDESTRUCT (255)
}
