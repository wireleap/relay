// Copyright (c) 2022 Wireleap

package map_counter

import (
	"github.com/wireleap/relay/api/synccounters"

	"sync"
	"sync/atomic"
)

type atomicList struct {
	atomic.Value
	sync.Mutex
}

func NewList() Map {
	r := &atomicList{}
	r.Value.Store([]Tuple{})
	return r
}

func (m *atomicList) load(key string) (value *synccounters.ContractCounter, ok bool) {
	for _, t := range m.Value.Load().([]Tuple) {
		if t.Key == key {
			value, ok = t.Val, true
			break
		}
	}
	return
}

func (m *atomicList) append(key string, value *synccounters.ContractCounter) {
	m.Lock()
	l := m.Value.Load().([]Tuple)
	m.Value.Store(append(l, Tuple{Key: key, Val: value}))
	m.Unlock()
}

func (m *atomicList) GetOrInit(key string) *synccounters.ConnCounter {
	v, ok := m.load(key)
	if ok {
		return v.NewChild()
	}

	ctc := synccounters.NewContractCounter()
	m.append(key, ctc)
	return ctc.NewChild()
}

func (m *atomicList) Range(f func(key string, value *synccounters.ContractCounter) bool) bool {
	for _, t := range m.Value.Load().([]Tuple) {
		if !f(t.Key, t.Val) {
			return false
		}
	}
	return true
}

func (m *atomicList) Reset() (map[string]uint64, bool) {
	return resetMap(m)
}
