// Copyright (c) 2022 Wireleap

package nustore

import (
	"strconv"
	"testing"

	"github.com/wireleap/relay/api/map_counter"
	"github.com/wireleap/relay/relaystats"
)

func handle_err(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Unexpected error [%s] happened", err.Error())
	}
}

func TestArchive(t *testing.T) {
	ns := relaystats.NewNetStats(map_counter.NewList)

	if !ns.Enabled() {
		t.Error("Netstats not initialised")
	}

	ns.CreatedAt = ns.CreatedAt - int64(1) // fake sleep

	// add some values
	cs1 := ns.ContractStats.GetOrInit("ct1")
	cs1.Add(uint64(100))
	handle_err(t, cs1.Close())

	cs2 := ns.ContractStats.GetOrInit("ct2")
	cs2.Add(uint64(200))
	handle_err(t, cs2.Close())

	cs_zero := ns.ContractStats.GetOrInit("ct_zero")
	cs_zero.Add(uint64(0))
	handle_err(t, cs_zero.Close())
	// This value won't be returned because the counter is 0

	// start recording date
	since := ns.CreatedAt

	m, b := ns.Reset()

	if !b {
		t.Error("Couldn't reset contract stats")
	}

	// ending recording date
	until := ns.CreatedAt

	archive := NewArchiveFile("someRelayId", map[string]bool{}, m, since, until)

	// match metric lengths
	if len(archive.Metrics) != len(m) {
		t.Error("Metrics should match in length")
	}

	for _, metric := range archive.Metrics {
		if nu, ok := m[metric.Contract]; !ok {
			t.Error("Metric should be present")
		} else if metric.Active {
			t.Error("Contract should be inactive")
		} else if metric.NetUsage != nu {
			t.Error("Metric should match")
		}
	}

	if archive.RelayId != "someRelayId" {
		t.Error("Archive relayId should match")
	}

	if archive.StartAt != since {
		t.Error("Archive startAt should match")
	}

	if archive.EndAt != until {
		t.Error("Archive endAt should match")
	}

	if archive.EndAt > archive.UpdatedAt {
		t.Error("Archive endAt should be lower than updatedAt")
	}

	_, k2, k3 := archive.Keys()

	if archive.RelayId != k2 {
		t.Error("Archive relayId should match")
	}

	if strconv.FormatInt(archive.EndAt, 10) != k3 {
		t.Error("Archive endAt should match")
	}
}

func TestArchiveActiveCT(t *testing.T) {
	ns := relaystats.NewNetStats(map_counter.NewList)

	if !ns.Enabled() {
		t.Error("Netstats not initialised")
	}

	ns.CreatedAt = ns.CreatedAt - int64(1) // fake sleep

	// add some values
	cs_active := ns.ContractStats.GetOrInit("ct1")
	cs_active.Add(uint64(100))
	handle_err(t, cs_active.Close())

	cs_inactive := ns.ContractStats.GetOrInit("ct2")
	cs_inactive.Add(uint64(200))
	handle_err(t, cs_inactive.Close())

	cs_active_zero := ns.ContractStats.GetOrInit("ct_zero")
	cs_active_zero.Add(uint64(0))
	handle_err(t, cs_active_zero.Close())

	// This value won't be returned because the counter is 0

	// start recording date
	since := ns.CreatedAt

	m, b := ns.Reset()

	if !b {
		t.Error("Couldn't reset contract stats")
	}

	// ending recording date
	until := ns.CreatedAt

	ctActive := map[string]bool{
		"ct1":      true,
		"ct2":      false,
		"ct_zero":  true,
		"ct_zero2": false,
	}

	archive := NewArchiveFile("someRelayId", ctActive, m, since, until)

	// match metric lengths
	if len(archive.Metrics) != len(m)+2 {
		t.Error("Metrics should match in length")
	}

	for _, metric := range archive.Metrics {
		b, ok_ct := ctActive[metric.Contract]
		nu, ok_nu := m[metric.Contract]
		if !ok_ct && !ok_nu {
			t.Error("Contract shouldn't exist")
		} else if metric.Active != b {
			t.Error("Contract should match state")
		} else if metric.NetUsage != nu {
			t.Error("Metric should match")
		}
	}

	if archive.RelayId != "someRelayId" {
		t.Error("Archive relayId should match")
	}

	if archive.StartAt != since {
		t.Error("Archive startAt should match")
	}

	if archive.EndAt != until {
		t.Error("Archive endAt should match")
	}

	if archive.EndAt > archive.UpdatedAt {
		t.Error("Archive endAt should be lower than updatedAt")
	}

	_, k2, k3 := archive.Keys()

	if archive.RelayId != k2 {
		t.Error("Archive relayId should match")
	}

	if strconv.FormatInt(archive.EndAt, 10) != k3 {
		t.Error("Archive endAt should match")
	}
}
