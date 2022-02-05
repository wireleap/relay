// Copyright (c) 2022 Wireleap

package synccounters

import (
	"sync"
)

type container struct {
	cts []*ConnCounter
	mu  sync.RWMutex
}

func newContainer(size int) *container {
	return &container{
		cts: make([]*ConnCounter, 0, size),
	}
}

func (cnt *container) _delete(i int) (res bool) {
	if res = cnt.cts != nil; !res {
		// pass
	} else if l := len(cnt.cts); i < l {
		cnt.cts[i] = cnt.cts[l-1]
		cnt.cts = cnt.cts[:l-1]
	} else {
		res = false
	}
	return
}

func (cnt *container) delete(cc *ConnCounter) (value *ConnCounter, ok bool) {
	cnt.mu.Lock()
	defer cnt.mu.Unlock()

	for i, t := range cnt.cts {
		if t == cc {
			value, ok = t, cnt._delete(i)
			break
		}
	}

	return
}

func (cnt *container) readLoop(fn func(int, *ConnCounter) bool) (value *ConnCounter, interrupt bool) {
	var i int

	cnt.mu.RLock()
	defer cnt.mu.RUnlock()

	for i, value = range cnt.cts {
		if interrupt = fn(i, value); interrupt {
			break
		}
	}
	return
}

func (cnt *container) create(parent *ContractCounter) (child *ConnCounter) {
	if cnt.cts == nil {
		return
	}

	child = NewConnCounter(parent)

	cnt.mu.Lock()
	defer cnt.mu.Unlock()

	cnt.cts = append(cnt.cts, child)
	return
}
