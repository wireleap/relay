// Copyright (c) 2022 Wireleap

package synccounters

import (
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
)

func assertEquals(t *testing.T, toBe interface{}, asIs interface{}, err string) {
	// Untyped nil checks
	nilIs := (asIs == nil || (reflect.ValueOf(asIs).Kind() == reflect.Ptr && reflect.ValueOf(asIs).IsNil()))
	nilBe := (toBe == nil || (reflect.ValueOf(toBe).Kind() == reflect.Ptr && reflect.ValueOf(toBe).IsNil()))

	if nilIs && nilBe {
		// pass
	} else if !reflect.DeepEqual(asIs, toBe) { // reflect.DeepEqual compares Typed nils
		t.Errorf("Expected %v but found %v, errorMsg: %s", asIs, toBe, err)
	}
}

func assertEqualErrs(t *testing.T, toBe error, asIs error, err string) {
	if !errors.Is(asIs, toBe) {
		strIs, strBe := "<nil>", "<nil>"

		if asIs != nil {
			strIs = asIs.Error()
		}

		if toBe != nil {
			strBe = toBe.Error()
		}

		t.Errorf("Expected [error] %v but found [error] %v, errorMsg: %s", strIs, strBe, err)
	}
}

func TestCounter(t *testing.T) {
	counter := NewContractCounter()

	// Countainer length
	assertEquals(t, 0, len(counter.cnt.cts), "Container length must be 0")

	// Init & Sum
	assertEquals(t, uint64(0), counter.Sum(), "Sum() must be 0")

	// Add & Sum
	counter.Add(1) // Can't fail
	assertEquals(t, uint64(1), counter.Sum(), "Sum() must be 1")

	// Reset & Sum
	assertEquals(t, uint64(1), counter.Reset(), "Reset() must be 1")
	assertEquals(t, uint64(0), counter.Sum(), "Sum() must be 0")
}

func TestCounterNotInit(t *testing.T) {
	counter := ContractCounter{}

	// Countainer is nil
	assertEquals(t, nil, counter.cnt, "Container must be nil")

	// Child Init
	assertEquals(t, nil, counter.NewChild(), "NewChild must return nil")
}

func TestConnCounter(t *testing.T) {
	counter := NewContractCounter()

	// Child Init
	child := counter.NewChild()

	// Countainer length
	assertEquals(t, 1, len(counter.cnt.cts), "Container length must be 1")

	// Child Sum
	assertEquals(t, uint64(0), child.Sum(), "Sum() must be 0")

	// Child Add & Sum
	inPtr, outPtr := child.Inner()
	atomic.AddUint64(inPtr, uint64(1))
	assertEquals(t, uint64(1), child.Sum(), "Sum() must be 1")
	atomic.AddUint64(outPtr, uint64(2))
	assertEquals(t, uint64(3), child.Sum(), "Sum() must be 3")

	// Parent Sum
	assertEquals(t, uint64(3), counter.Sum(), "Sum() must be 3")

	// Parent Reset
	assertEquals(t, uint64(3), counter.Reset(), "Reset() must be 3")

	// Child Sum
	assertEquals(t, uint64(0), child.Sum(), "Sum() must be 0")

	// Child Add & Sum
	// Pointers should still be valid after parent or child reset
	atomic.AddUint64(inPtr, uint64(1))
	assertEquals(t, uint64(1), child.Sum(), "Sum() must be 1")
	atomic.AddUint64(outPtr, uint64(1))
	assertEquals(t, uint64(2), child.Sum(), "Sum() must be 2")

	// Child Add & Parent Add & Sum
	counter.Add(1) // Can't fail
	assertEquals(t, nil, child.Add(2), "Add() must be nil")
	assertEquals(t, uint64(5), counter.Sum(), "Sum() must be 5")

	// Child Close
	assertEqualErrs(t, nil, child.Close(), "Close shouldn't return an error")

	// Countainer length
	assertEquals(t, 0, len(counter.cnt.cts), "Container length must be 0")

	// Parent Sum
	assertEquals(t, uint64(5), counter.Sum(), "Sum() must be 5")
}

func TestClosedConnCounter(t *testing.T) {
	counter := NewContractCounter()

	// Child Init
	child := counter.NewChild()

	// Child Close
	assertEqualErrs(t, nil, child.Close(), "Close shouldn't return an error")

	// Child Close twice
	assertEqualErrs(t, ErrParent, child.Add(0), "Add should return an error")

	// Child Close twice
	assertEqualErrs(t, ErrParent, child.Close(), "Close should return an error")

	// Fore parent "reopen"
	child.parent = counter

	// Child Close twice
	assertEqualErrs(t, ErrContainer, child.Close(), "Close should return an error")
}

func TestContainerInterrupt(t *testing.T) {
	fn := func(_ int, cc *ConnCounter) bool {
		return true // interrupt
	}

	cnt := newContainer(5)
	cnt.create(nil)
	_, interrupt := cnt.readLoop(fn)

	assertEquals(t, true, interrupt, "Readloop must be interrupted")
}

func TestContainerOutOfBounds(t *testing.T) {
	cnt := newContainer(1)
	cnt.create(nil)

	assertEquals(t, false, cnt._delete(2), "_delete must break")
}

func TestContainerNotInit(t *testing.T) {
	cnt := container{}

	assertEquals(t, false, cnt._delete(0), "_delete must break")

	child := cnt.create(nil)
	assertEquals(t, nil, child, "create must break")
}
