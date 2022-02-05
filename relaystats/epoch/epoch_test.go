// Copyright (c) 2022 Wireleap

package epoch

import (
	"testing"
	"time"
)

func TestEpochMillis(t *testing.T) {
	e := ToEpochMillis(time.Now())

	// Check function duality
	if ti := FromEpochMillis(e); ToEpochMillis(ti) != e {
		t.Error("Time value should match")
	}

	e2 := EpochMillis()

	// Check Epoch.Now() returns a date older than previous one
	if e2 < e {
		t.Error("Epoch.Now() should be higher")
	}
}
