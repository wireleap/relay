// Copyright (c) 2022 Wireleap

package synccounters

import "sync/atomic"

func containerSum() (func(int, *ConnCounter) bool, *uint64) {
	sum := new(uint64)

	return func(_ int, cc *ConnCounter) bool {
		atomic.AddUint64(sum, cc.Sum())
		return false // interrupt
	}, sum
}

func containerReset() (func(int, *ConnCounter) bool, *uint64) {
	sum := new(uint64)

	return func(_ int, cc *ConnCounter) bool {
		atomic.AddUint64(sum, cc.Reset())
		return false // interrupt
	}, sum
}
