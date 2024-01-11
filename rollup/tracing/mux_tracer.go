package tracing

import (
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/vm"
	_ "github.com/scroll-tech/go-ethereum/eth/tracers/native"
)

type MuxTracer struct {
	tracers []vm.EVMLogger
}

func NewMuxTracer(tracers ...vm.EVMLogger) *MuxTracer {
	return &MuxTracer{tracers}
}

func (t *MuxTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	for _, tracer := range t.tracers {
		tracer.CaptureStart(env, from, to, create, input, gas, value)
	}
}

func (t *MuxTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	for _, tracer := range t.tracers {
		tracer.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
	}
}

func (t *MuxTracer) CaptureStateAfter(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	for _, tracer := range t.tracers {
		tracer.CaptureStateAfter(pc, op, gas, cost, scope, rData, depth, err)
	}
}

func (t *MuxTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	for _, tracer := range t.tracers {
		tracer.CaptureEnter(typ, from, to, input, gas, value)
	}
}

func (t *MuxTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	for _, tracer := range t.tracers {
		tracer.CaptureExit(output, gasUsed, err)
	}
}

func (t *MuxTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	for _, tracer := range t.tracers {
		tracer.CaptureFault(pc, op, gas, cost, scope, depth, err)
	}
}

func (t *MuxTracer) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	for _, tracer := range t.tracers {
		tracer.CaptureEnd(output, gasUsed, d, err)
	}
}
