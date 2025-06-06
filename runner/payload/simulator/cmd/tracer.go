// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
)

// noop is a no-op live tracer. It's there to
// catch changes in the tracing interface, as well as
// for testing live tracing performance. Can be removed
// as soon as we have a real live tracer.
type opcodeTracer struct {
	opcodeStats     opcodeStats
	precompileStats opcodeStats
}

func newOpcodeTracer() *opcodeTracer {
	return &opcodeTracer{
		opcodeStats:     make(opcodeStats),
		precompileStats: make(opcodeStats),
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
			t.precompileStats[allPrecompiles[addr]]++
		}
	}
}
