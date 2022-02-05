// Copyright (c) 2022 Wireleap

package map_counter

import (
	"fmt"
	"testing"

	"github.com/wireleap/relay/api/synccounters"
)

func testMap(t *testing.T, newMap func() Map) {
	m := newMap()
	x := m.GetOrInit("key1")
	x_ := m.GetOrInit("key1")

	if x == x_ {
		t.Error("Children should have different address")
	}

	if x.Close() != nil {
		t.Error("Error should be nil")
	}

	if x_.Close() != nil {
		t.Error("Error should be nil")
	}

	if m.GetOrInit("key2").Close() != nil {
		t.Error("Error should be nil")
	}

	var counter int

	f := func(key string, value *synccounters.ContractCounter) bool {
		counter += 1
		if counter == 2 {
			return false
		}
		return true
	}

	if m.Range(f) {
		t.Error("Iteration should have been aborted")
	}

	f = func(key string, value *synccounters.ContractCounter) bool {
		return true
	}

	if !m.Range(f) {
		t.Error("Iteration shouldn't have been aborted")
	}
}

func testMapReset(t *testing.T, newMap func() Map) {
	test_map := map[string]uint64{
		"key1": uint64(5),
		"key2": uint64(5),
	}

	m := newMap()

	// Case1: Initialised synccounters.Map.
	for k, v := range test_map {
		x := m.GetOrInit(k)
		x.Add(v)

		if x.Close() != nil {
			t.Error("Error should be nil")
		}

		if x.Close() == nil {
			t.Error("Error shouldn't be nil")
		}
	}

	// Case2: This parent counter has been pushed in a RWC pipe, but child is new.
	x := m.GetOrInit("key1")

	if x.Sum() != 0 {
		t.Error("Item has wrong value")
	}

	// Add some value in child
	x.Add(uint64(5))
	test_map["key1"] += uint64(5)

	ms, ok := m.Reset() // Reset parent and childs

	// Case3: Poped reset result matches original synccounters.Map.
	if !ok {
		t.Error("Iteration shouldn't have been aborted")
	} else if len(ms) != len(test_map) {
		t.Error("Map length should match")
	}

	// Case3: Poped reset result matches original synccounters.Map.
	for k, v := range ms {
		if test_map[k] != v {
			fmt.Println(test_map[k], v)
			t.Error("Values should match")
		}
	}

	// Case1: Reset synccounters.Map.
	m.Range(func(_ string, value *synccounters.ContractCounter) bool {
		if value.Sum() != 0 {
			t.Error("Item in map wasn't reset")
		}
		return true
	})

	// Case2: Connection counter has also been reset.
	if x.Sum() != 0 {
		t.Error("Item has worng value")
	}

	if x.Close() != nil {
		t.Error("Error should be nil")
	}
}

func TestCMap(t *testing.T) {
	testMap(t, NewCMap)
	testMapReset(t, NewCMap)
}

func TestAtomicList(t *testing.T) {
	testMap(t, NewList)
	testMapReset(t, NewList)
}
