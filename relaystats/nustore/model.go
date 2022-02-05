// Copyright (c) 2022 Wireleap

package nustore

import (
	"strconv"

	"github.com/wireleap/relay/relaystats/epoch"
)

type ContractMetric struct {
	Contract string `json:"contract"`
	Active   bool   `json:"active"`
	NetUsage uint64 `json:"network_usage_bytes"`
}

type NetStats struct {
	RelayId   string           `json:"relay_id"`
	Metrics   []ContractMetric `json:"metrics"`
	StartAt   int64            `json:"start_at"`
	EndAt     int64            `json:"end_at"`
	UpdatedAt int64            `json:"updated_at"`
}

func mergeCts(ctActive map[string]bool, netusageMetrics map[string]uint64) (cts map[string]bool) {
	cts = make(map[string]bool)

	for ct, b := range ctActive {
		cts[ct] = b
	}

	for ct, _ := range netusageMetrics {
		if _, ok := cts[ct]; !ok {
			cts[ct] = false
		}
	}
	return
}

func buildMetrics(ctActive map[string]bool, netusageMetrics map[string]uint64) (cms []ContractMetric) {
	cts := mergeCts(ctActive, netusageMetrics)

	cms = make([]ContractMetric, 0, len(cts))
	for ct, b := range cts {
		nu, _ := netusageMetrics[ct]
		cms = append(cms, ContractMetric{Contract: ct, Active: b, NetUsage: nu})
	}
	return
}

func NewArchiveFile(relayId string, ctActive map[string]bool, netusageMetrics map[string]uint64, startAt, endAt int64) NetStats {
	metrics := buildMetrics(ctActive, netusageMetrics)
	return NetStats{
		RelayId:   relayId,
		Metrics:   metrics,
		StartAt:   startAt,
		EndAt:     endAt,
		UpdatedAt: epoch.EpochMillis(),
	}
}

func (ns NetStats) Keys() (string, string, string) {
	return "", ns.RelayId, strconv.FormatInt(ns.EndAt, 10)
}
