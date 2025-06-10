package main

import (
	"github.com/base/base-bench/runner/payload/simulator/simulatorstats"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
)

// opcodeTracer is a live tracer that tracks the opcode and precompile stats.
type opcodeTracer struct {
	opcodeStats     simulatorstats.OpcodeStats
	precompileStats simulatorstats.OpcodeStats
}

func newOpcodeTracer() *opcodeTracer {
	return &opcodeTracer{
		opcodeStats:     make(simulatorstats.OpcodeStats),
		precompileStats: make(simulatorstats.OpcodeStats),
	}
}

func (t *opcodeTracer) Tracer() *tracing.Hooks {
	return &tracing.Hooks{
		OnOpcode: t.OnOpcode,
	}
}

func (t *opcodeTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	opcode := vm.OpCode(op)
	t.opcodeStats[opcode.String()]++
	if opcode == vm.CALL || opcode == vm.CALLCODE || opcode == vm.DELEGATECALL || opcode == vm.STATICCALL || opcode == vm.EXTSTATICCALL {
		addressBig := scope.StackData()[0]
		addr := common.BigToAddress(addressBig.ToBig())
		precompiles := vm.PrecompiledContractsIsthmus
		if precompiles[addr] != nil {
			t.precompileStats[simulatorstats.PrecompileAddressToName[addr]]++
		}
	}
}
