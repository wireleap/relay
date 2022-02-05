// Copyright (c) 2022 Wireleap

package file

import (
	"github.com/wireleap/relay/relaystats/epoch"
)

// Statistics per contract
type contractStat struct {
	NetworkBytes uint64 `json:"network_bytes,omitempty"`
}

type NetStats struct {
	ContractStats map[string]*contractStat `json:"contract_stats,omitempty"`
	CreatedAt     int64                    `json:"created_at,omitempty"`
	UpdatedAt     int64                    `json:"updated_at,omitempty"`
}

func NewContractStats() map[string]*contractStat {
	return make(map[string]*contractStat)
}

func (ns *NetStats) Update(contract string, bytes uint64) bool {
	if ns.CreatedAt == 0 {
		return false
	}

	if cs, ok := ns.get(contract); ok {
		cs.NetworkBytes = bytes
		ns.UpdatedAt = epoch.EpochMillis()
	} else {
		ns.append(contract, bytes)
	}
	return true
}

func (ns *NetStats) get(contract string) (cs *contractStat, res bool) {
	for ct, csi := range ns.ContractStats {
		if ct == contract {
			cs = csi
			res = true
		}
	}
	return
}

func (ns *NetStats) Get(contract string) (*contractStat, bool) {
	if ns.CreatedAt == 0 {
		return &contractStat{}, false
	}

	return ns.get(contract)
}

func (ns *NetStats) Set(contract string, bytes uint64) bool {
	if ns.CreatedAt == 0 {
		return false
	}

	ns.append(contract, bytes)
	return true
}

func (ns *NetStats) append(contract string, bytes uint64) {
	cs := contractStat{bytes}
	ns.ContractStats[contract] = &cs
	ns.UpdatedAt = epoch.EpochMillis()
	return
}
