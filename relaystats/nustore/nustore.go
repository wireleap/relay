// Copyright (c) 2022 Wireleap

package nustore

import (
	"sync"

	"github.com/wireleap/common/cli/fsdir"
)

// T is the type of a network usage store.
type T struct {
	m  fsdir.T
	mu sync.RWMutex
}

// New initializes a network usage store in the directory under the path given by
// the dir argument.
func New(dir string) (t *T, err error) {
	t = &T{}
	t.m, err = fsdir.New(dir)

	return
}

// Add adds a archive document to the store, returns error in case of failure
func (t *T) Add(ns NetStats) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, k2, k3 := ns.Keys()

	return t.m.Set(ns, k2+"-"+k3+".json")
}
