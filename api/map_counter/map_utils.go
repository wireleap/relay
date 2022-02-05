// Copyright (c) 2022 Wireleap

package map_counter

import (
	"github.com/wireleap/relay/api/synccounters"
)

func resetMapFunction() (func(string, *synccounters.ContractCounter) bool, map[string]uint64) {
	m := make(map[string]uint64, 0)
	return func(key string, value *synccounters.ContractCounter) bool {
		if _, ok := m[key]; ok {
			// In theory we're only iterating over each element once
		} else if value == nil {
			// Safety check, no address to override ==> ABORT!
			return false
		} else {
			// Save netstat status if not zero
			if count := value.Reset(); count != uint64(0) {
				m[key] = count
			}
		}
		return true
	}, m
}

func resetMap(m Map) (map[string]uint64, bool) {
	f, ms := resetMapFunction()
	return ms, m.Range(f)
}
