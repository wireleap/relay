// Copyright (c) 2022 Wireleap

package synccounters

import (
	"sync/atomic"
)

// CNTLENGTH is the standard container capacity
const CNTLENGTH = 500

// ContractCounter contains the accumulated for an entire contract
type ContractCounter struct {
	cnt   *container
	value uint64
}

// NewContractCounter returns new traffic counter per contract
func NewContractCounter() *ContractCounter {
	return &ContractCounter{
		cnt: newContainer(CNTLENGTH),
	}
}

// Add to inner value
func (ctc *ContractCounter) Add(i uint64) {
	atomic.AddUint64(&ctc.value, i)
}

// Sum to inner value and child values
func (ctc *ContractCounter) Sum() uint64 {
	fn, sum := containerSum()
	if _, interrupt := ctc.cnt.readLoop(fn); interrupt {
		return 0
	}

	return *sum + ctc.value
}

// Reset inner and child values
func (ctc *ContractCounter) Reset() uint64 {
	fn, sum := containerReset()
	if _, interrupt := ctc.cnt.readLoop(fn); interrupt {
		return 0
	}

	return *sum + atomic.SwapUint64(&ctc.value, 0)
}

// NewChild linked to the ContractCounter
func (ctc *ContractCounter) NewChild() *ConnCounter {
	if ctc.cnt == nil {
		return nil
	}

	return ctc.cnt.create(ctc)
}
