// Copyright (c) 2022 Wireleap

package map_counter

import (
	"github.com/wireleap/relay/api/synccounters"
)

type Map interface {
	// Finds the exisiting contract counter or initialises a new one,
	// and always retuns a new child counter
	GetOrInit(key string) (value *synccounters.ConnCounter)
	// Iterate over the entire itemlist
	// f should return false to abort iteration
	// Range(f) returns if iteration was completed
	Range(f func(key string, value *synccounters.ContractCounter) bool) bool
	// Reset values to 0
	// Returns a map[string]uint64 with the previous values, and a completion flag
	Reset() (map[string]uint64, bool)
}
