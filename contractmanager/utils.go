// Copyright (c) 2022 Wireleap

package contractmanager

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/wireleap/common/cli/fsdir"

	"github.com/wireleap/relay/api/map_counter"
	"github.com/wireleap/relay/filenames"
	"github.com/wireleap/relay/relaystats"
	"github.com/wireleap/relay/relaystats/file"
)

type void struct{}

// Strip legacy statistics from storage file
func filterInactive(sfile *file.NetStats, contractIds []string) map[string]uint64 {
	contractIdSet := make(map[string]void, len(contractIds))
	for _, id := range contractIds {
		contractIdSet[id] = void{}
	}

	legacyns := make(map[string]uint64)
	for ct, cs := range sfile.ContractStats {
		if _, ok := contractIdSet[ct]; !ok {
			legacyns[ct] = cs.NetworkBytes
			delete(sfile.ContractStats, ct)
		}
	}

	// Skip initialisation is not needed
	if len(legacyns) == 0 {
		return nil
	}

	return legacyns
}

func loadStats(fm fsdir.T, contractIds []string) (netstats relaystats.NetStats, legacyns map[string]uint64, err error) {
	fns := relaystats.NewFileNetStats()

	if err = fm.Get(fns, filenames.Stats); err == nil {
		legacyns = filterInactive(fns, contractIds)
		netstats = relaystats.Load(fns, map_counter.NewList)
	} else if errors.Is(err, os.ErrNotExist) {
		// lazy file generation
		if err = fm.Set(fns, filenames.Stats); err != nil {
			err = fmt.Errorf("could not initialise statistics file: %s", err)
			return
		}

		//legacyns = map[string]uint64{}
		netstats = relaystats.NewNetStats(map_counter.NewList)
		log.Printf("initialising statistics file")
	}
	return
}

// Insert not inactive and legacy contracts
func mergeInactive(sfile *file.NetStats, contractIds []string, legacyns map[string]uint64) {
	for _, ct := range contractIds {
		if _, ok := sfile.ContractStats[ct]; !ok {
			sfile.Update(ct, 0)
		}
	}

	if legacyns != nil {
		for ct, netusage := range legacyns {
			if _, ok := sfile.ContractStats[ct]; !ok {
				sfile.Update(ct, netusage)
			}
		}
	}
}

func saveStats(netstats relaystats.NetStats, contractIds []string, legacyns map[string]uint64) (fns *file.NetStats, err error) {
	fns = relaystats.NewFileNetStats()

	if !relaystats.Save(netstats, fns) {
		err = errors.New("error generating network usage stats")
		return
	} else {
		mergeInactive(fns, contractIds, legacyns)
	}
	return
}

func NewDummyManager() *Manager {
	return &Manager{
		NetStats: netStats{
			Active: relaystats.NewNetStats(map_counter.NewList),
		},
	}
}
