// Copyright (c) 2022 Wireleap

package synccounters

import (
	"errors"
	"sync/atomic"
)

var (
	ErrParent    = errors.New("parent is nil, child is closed")
	ErrContainer = errors.New("couldn't delete counter_child in container")
)

// ConnCounter is a linked counter for a connection
type ConnCounter struct {
	in     uint64
	out    uint64
	parent *ContractCounter
}

// NewConnCounter returns a new counter for a connection
func NewConnCounter(parent *ContractCounter) *ConnCounter {
	return &ConnCounter{
		parent: parent,
	}
}

// Add to parent value
func (cc *ConnCounter) Add(i uint64) error {
	if cc.parent == nil {
		return ErrParent
	}

	atomic.AddUint64(&cc.parent.value, i)
	return nil
}

// Inner Counters
func (cc *ConnCounter) Inner() (*uint64, *uint64) {
	return &cc.in, &cc.out
}

// Sum to inner value and child values
func (cc *ConnCounter) Sum() uint64 {
	return cc.in + cc.out
}

// Close counter, sum inner value to parent and self-destruct
func (cc *ConnCounter) Close() (err error) {
	if cc.parent == nil {
		err = ErrParent
		return
	}

	cc.parent.Add(cc.Reset())

	if _, ok := cc.parent.cnt.delete(cc); !ok {
		err = ErrContainer
	}

	// self-destruct
	cc.parent = nil
	return
}

// Reset inner value
func (cc *ConnCounter) Reset() uint64 {
	in := atomic.SwapUint64(&cc.in, 0)
	out := atomic.SwapUint64(&cc.out, 0)
	return in + out
}
