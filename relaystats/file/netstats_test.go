// Copyright (c) 2022 Wireleap

package file

import (
	"testing"
)

func TestEmptyFileNetStats(t *testing.T) {
	cs := NewContractStats()

	if len(cs) > 0 {
		t.Error("Length must be 0")
	}

	cs1 := &contractStat{uint64(0)}
	cs["ct1"] = cs1

	ns := NetStats{
		ContractStats: cs,
	}

	// Test get and Get

	if csi, ok := ns.get("ct1"); !ok {
		t.Error("Couldn't recover contract stats")
	} else if csi != cs1 {
		t.Error("Recovered wrong contract stats")
	}

	if _, ok := ns.Get("ct1"); ok {
		t.Error("Recovered wrong contract stats")
	}

	if _, ok := ns.get("ct2"); ok {
		t.Error("Recovered wrong contract stats")
	}

	// Test append, Update and Set

	if ns.Set("ct2", uint64(0)) {
		t.Error("Shouldn't be allowed to set contract stats")
	}

	if ns.Update("ct2", uint64(0)) {
		t.Error("Shouldn't be allowed to set contract stats")
	}

	ns.append("ct2", uint64(100))

	// Check set

	if csi, ok := ns.get("ct2"); !ok {
		t.Error("Couldn't recover contract stats")
	} else if csi.NetworkBytes != uint64(100) {
		t.Error("Recovered wrong contract stats")
	}
}

func TestFileNetStats(t *testing.T) {
	ns := NetStats{
		ContractStats: NewContractStats(),
		CreatedAt:     1,
	}

	// Test Get

	if _, ok := ns.Get("ct1"); ok {
		t.Error("Recovered wrong contract stats")
	}

	if ns.UpdatedAt != 0 {
		t.Error("NetStat UpdatedAt shouldn't be initialised")
	}

	// Test Set

	if !ns.Set("ct1", uint64(0)) {
		t.Error("Should be allowed to set contract stats")
	}

	if ns.UpdatedAt == 0 {
		t.Error("NetStat UpdatedAt should be initialised")
	}

	var cs1 *contractStat
	var ok bool

	if cs1, ok = ns.Get("ct1"); !ok {
		t.Error("Couldn't recover contract stats")
	}

	// Test Update vs Set

	if !ns.Update("ct1", uint64(100)) {
		t.Error("Should be allowed to set contract stats")
	}

	if cs1.NetworkBytes != uint64(100) {
		t.Error("Recovered wrong contract stats")
	}

	if cs_, ok_ := ns.Get("ct1"); !ok_ {
		t.Error("Couldn't recover contract stats")
	} else if cs_ != cs1 {
		t.Error("Recovered wrong contract stats")
	}

	// Test Update with init

	if !ns.Update("ct2", uint64(100)) {
		t.Error("Should be allowed to set contract stats")
	}
}
