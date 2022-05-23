// Copyright (c) 2022 Wireleap

package relaystats

import (
	"testing"
	"time"

	"github.com/wireleap/relay/api/epoch"
	"github.com/wireleap/relay/api/map_counter"
)

func TestNetStatsLoadSave(t *testing.T) {
	fns := NewFileNetStats()

	if !fns.Set("ct1", uint64(100)) {
		t.Error("Should be allowed to set contract stats")
	}

	fns.CreatedAt = fns.CreatedAt - int64(1) // fake sleep
	fns.UpdatedAt = fns.UpdatedAt - int64(1) // fake sleep

	ns := Load(fns, map_counter.NewList)

	if !ns.Enabled() {
		t.Error("Netstats not initialised")
	}

	if ns.CreatedAt != fns.CreatedAt {
		t.Error("Netstats not properly initialised")
	}

	m, _ := status(ns) //dump counters

	if m["ct1"] != uint64(100) {
		t.Error("Couldn't recover contract stats")
	}

	old_u_at := fns.UpdatedAt

	// Override old fns
	if !Save(ns, fns) {
		t.Error("Couldn't save contract stats")
	}

	// Reset FileNetStats
	fns = NewFileNetStats()

	// Override empty fns
	if !Save(ns, fns) {
		t.Error("Couldn't save contract stats")
	}

	if fns.CreatedAt != ns.CreatedAt {
		t.Error("Couldn't recover original created_at")
	}

	if fns.UpdatedAt == old_u_at {
		t.Error("Couldn't update updated_at")
	}
}

func TestNetStatsInit(t *testing.T) {
	ns := NewNetStats(map_counter.NewList)

	if !ns.Enabled() {
		t.Error("Netstats not initialised")
	}

	m, _ := status(ns) //dump counters

	if m["ct1"] != uint64(0) {
		t.Error("Couldn't recover contract stats")
	}
}

func TestNetStatsReset(t *testing.T) {
	ns := NewNetStats(map_counter.NewList)

	if !ns.Enabled() {
		t.Error("Netstats not initialised")
	}

	ns.CreatedAt = ns.CreatedAt - int64(1) // fake sleep

	cs1 := ns.ContractStats.GetOrInit("ct1")

	cs1.Add(uint64(100))

	mx, _ := status(ns) //dump counters
	cs1.Close()

	if mx["ct1"] == uint64(0) {
		t.Error("Couldn't update contract stats")
	}

	c_at := ns.CreatedAt

	if m, b := ns.Reset(); !b {
		t.Error("Couldn't reset contract stats")
	} else if cs, ok := m["ct1"]; len(m) != 1 || !ok {
		t.Error("Couldn't dump contract stats on Reset")
	} else if cs != uint64(100) {
		t.Error("Couldn't dump contract stats on Reset, ct1 not matching")
	}

	mx, _ = status(ns) //dump counters

	if mx["ct1"] != uint64(0) {
		t.Error("Couldn't update contract stats on Reset")
	}

	if c_at == ns.CreatedAt {
		t.Error("Couldn't update created_at stats on Reset")
	}
}

func TestNetStatsGetNextReset(t *testing.T) {
	ns := NewNetStats(map_counter.NewList)

	if !ns.Enabled() {
		t.Error("Netstats not initialised")
	}

	// Common case
	next_reset, reset_now := ns.GetNextReset(time.Minute)
	if reset_now {
		t.Error("Shouldn't reset now")
	}

	if next_reset != epoch.FromEpochMillis(ns.CreatedAt).Add(time.Minute) {
		t.Error("Incorrect next_reset date")
	}

	// Need to reset, but still in next time frame
	ns.CreatedAt = ns.CreatedAt - int64(90000) // fake sleep

	next_reset, reset_now = ns.GetNextReset(time.Minute)
	if !reset_now {
		t.Error("Should reset now")
	}

	if next_reset != epoch.FromEpochMillis(ns.CreatedAt).Add(2*time.Minute) {
		t.Error("Incorrect next_reset date")
	}

	// Need to reset, but ahead of next time frame, reseting
	ns.CreatedAt = ns.CreatedAt - int64(60000) // fake sleep

	next_reset, reset_now = ns.GetNextReset(time.Minute)
	if !reset_now {
		t.Error("Should reset now")
	}

	if !next_reset.After(epoch.FromEpochMillis(ns.CreatedAt).Add(2 * time.Minute)) {
		t.Error("Incorrect next_reset date")
	}

}
