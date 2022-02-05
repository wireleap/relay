// Copyright (c) 2022 Wireleap

package relaystats

import (
	"time"

	"github.com/wireleap/relay/api/map_counter"
	"github.com/wireleap/relay/api/synccounters"
	"github.com/wireleap/relay/relaystats/epoch"
	"github.com/wireleap/relay/relaystats/file"
)

type NetStats struct {
	ContractStats map_counter.Map
	CreatedAt     int64
}

// Initialise Netstats
func NewNetStats(initMap func() map_counter.Map) NetStats {
	return NetStats{
		ContractStats: initMap(),
		CreatedAt:     epoch.EpochMillis(),
	}
}

// Return if NetStats have been initialised
func (ns NetStats) Enabled() bool {
	return ns.CreatedAt != 0
}

// Reset NetStats
func (ns *NetStats) Reset() (map[string]uint64, bool) {
	return ns.ResetWithDate(time.Now())
}

// Reset NetStats with date
func (ns *NetStats) ResetWithDate(t time.Time) (m map[string]uint64, ok bool) {
	if m, ok = ns.ContractStats.Reset(); ok {
		ns.CreatedAt = epoch.ToEpochMillis(t)
	}
	return
}

// Returns time next Reset in the future, and if NS should have been already reset
func (ns NetStats) GetNextReset(d time.Duration) (time.Time, bool) {
	c_at := epoch.FromEpochMillis(ns.CreatedAt)
	now := time.Now()

	if n2 := c_at.Add(2 * d); n2.Before(now) {
		// Now + Duration
		return now.Add(d), true
	} else if n1 := c_at.Add(d); n1.Before(now) {
		// Created_At + 2 * Duration
		return n2, true
	} else {
		// Created_At + Duration
		return n1, false
	}
}

// Load from file format
func Load(sfile *file.NetStats, initMap func() map_counter.Map) (ns NetStats) {
	ns = NetStats{initMap(), sfile.CreatedAt}
	for ct, cs := range sfile.ContractStats {

		if cs != nil {
			x := ns.ContractStats.GetOrInit(ct)
			x.Add(cs.NetworkBytes) // Set

			if err := x.Close(); err != nil {
				panic(err)
			}
		}
	}
	return
}

// Save to file format
func status(ns NetStats) (m map[string]uint64, res bool) {
	m = make(map[string]uint64)

	fLoad := func(contract string, contractBytes *synccounters.ContractCounter) bool {
		// 1) Check if nB has been initialised, 2) Check is not null, 3) Copy
		if contractBytes != nil {
			m[contract] = contractBytes.Sum()
			return false
		}
		return true
	}

	res = ns.ContractStats.Range(fLoad)
	return
}

// Save to file format
func Save(ns NetStats, sfile *file.NetStats) bool {
	fLoad := func(contract string, contractBytes *synccounters.ContractCounter) bool {
		// 1) Check if nB has been initialised, 2) Check is not null, 3) Copy
		if contractBytes == nil {
			// pass
		} else if cB := contractBytes.Sum(); cB == uint64(0) {
			// pass
		} else if ok := sfile.Update(contract, cB); !ok {
			return false
		}
		return true
	}

	res := ns.ContractStats.Range(fLoad)
	if res {
		sfile.CreatedAt = ns.CreatedAt
	}
	return res
}

// Initialise FileNetstats
func NewFileNetStats() *file.NetStats {
	now := epoch.EpochMillis()
	return &file.NetStats{
		ContractStats: file.NewContractStats(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
