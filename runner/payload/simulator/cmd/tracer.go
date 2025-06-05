package main

import (
	"math/big"

	"github.com/base/base-bench/runner/payload/simulator/simulatorstats"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
)

// opcodeTracer is a live tracer that tracks the opcode and precompile stats.
type opcodeTracer struct {
	opcodeStats      simulatorstats.OpcodeStats
	precompileStats  simulatorstats.OpcodeStats
	addressesChanged map[common.Address]struct{}
}

func newOpcodeTracer() *opcodeTracer {
	return &opcodeTracer{
		opcodeStats:      make(simulatorstats.OpcodeStats),
		precompileStats:  make(simulatorstats.OpcodeStats),
		addressesChanged: make(map[common.Address]struct{}),
	}
}

func (t *opcodeTracer) Tracer() *tracing.Hooks {
	return &tracing.Hooks{
		OnOpcode:        t.OnOpcode,
		OnBalanceChange: t.OnBalanceChange,
		OnNonceChangeV2: t.OnNonceChangeV2,
		OnCodeChange:    t.OnCodeChange,
	}
}

func (t *opcodeTracer) OnBalanceChange(addr common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
	t.addressesChanged[addr] = struct{}{}
}

func (t *opcodeTracer) OnNonceChangeV2(addr common.Address, old, new uint64, reason tracing.NonceChangeReason) {
	t.addressesChanged[addr] = struct{}{}
}

func (t *opcodeTracer) OnCodeChange(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
	t.addressesChanged[addr] = struct{}{}
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
