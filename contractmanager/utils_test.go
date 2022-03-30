// Copyright (c) 2022 Wireleap

package contractmanager

import (
	"testing"

	"github.com/wireleap/relay/relaystats/file"
)

func TestFilterInactive(t *testing.T) {
	fns := &file.NetStats{
		ContractStats: file.NewContractStats(),
		CreatedAt:     1, // Fake init
		UpdatedAt:     1,
	}

	cts := []string{"ct1", "ct2", "ct3"}
	nocts := []string{"ct4", "ct5", "ct6"}

	for _, ct := range append(cts, nocts...) {
		if !fns.Update(ct, 100) {
			t.Fatal("File NetStat should update")
		}
	}

	inactive := filterInactive(fns, append(cts, "ct7"))

	for _, ct := range cts {
		if _, ok := fns.Get(ct); !ok {
			t.Fatal("ConstractStat should be present")
		}
	}

	for _, ct := range append(nocts, "ct7") {
		if _, ok := fns.Get(ct); ok {
			t.Fatal("ConstractStat shouldn't be present")
		}
	}

	if len(inactive) != len(nocts) {
		t.Fatal("Inactive contractStats should match")
	}
}

func TestMergeInactive(t *testing.T) {
	fns := &file.NetStats{
		ContractStats: file.NewContractStats(),
		CreatedAt:     1, // Fake init
		UpdatedAt:     1,
	}

	cts := []string{"ct1", "ct2", "ct3"}
	cts_ext := append(cts, "ct7")
	nocts := map[string]uint64{"ct4": 100, "ct5": 200, "ct6": 300}

	for _, ct := range append(cts) {
		if !fns.Update(ct, 100) {
			t.Fatal("File NetStat should update")
		}
	}

	mergeInactive(fns, cts_ext, nocts)

	for _, ct := range cts_ext {
		if val, ok := fns.Get(ct); ct == "ct7" && ok && val.NetworkBytes != uint64(0) {
			t.Fatal("ConstractStat should be present")
		} else if ct != "ct7" && ok && val.NetworkBytes != uint64(100) {
			t.Fatal("ConstractStat should be present")
		}
	}

	for ct, val_orig := range nocts {
		if val, ok := fns.Get(ct); !ok {
			t.Fatal("ConstractStat should be present")
		} else if val.NetworkBytes != val_orig {
			t.Fatal("ConstractStat should be the same value")
		}
	}
}
