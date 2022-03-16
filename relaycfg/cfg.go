// Copyright (c) 2021 Wireleap

// Package relaycfg describes the configuration file format and data types
// used by wireleap-relay.
package relaycfg

import (
	"errors"
	"fmt"
	"time"

	"github.com/wireleap/common/api/duration"
	"github.com/wireleap/common/api/relayentry"
	"github.com/wireleap/common/api/texturl"
)

// C is the type of the config struct describing the config file format.
type C struct {
	// Address is the wireleap:// listening address.
	Address *string `json:"address,omitempty"`
	// AutoSubmitInterval is the retry interval for autosubmission.
	// Autosubmission is disabled if it is 0.
	AutoSubmitInterval duration.T `json:"auto_submit_interval,omitempty"`
	// ArchiveDir is the path of the archived sharetoken store directory.
	ArchiveDir *string `json:"archive_dir,omitempty"`
	// MaxTime is the maximum time for a single connection.
	MaxTime duration.T `json:"maxtime,omitempty"`
	// Timeout is the dial timeout.
	Timeout duration.T `json:"timeout,omitempty"`
	// BufSize is the size in bytes of transmit/receive buffers.
	BufSize int `json:"bufsize,omitempty"`
	// Contracts is the map of service contracts used by this wireleap-relay.
	Contracts map[texturl.URL]*relayentry.T `json:"contracts,omitempty"`
	// AutoUpgrade sets whether this relay should attempt auto-upgrades.
	AutoUpgrade bool `json:"auto_upgrade,omitempty"`
	// Those are expert settings. Take care.
	DangerZone DangerZone `json:"danger_zone,omitempty"`
}

type DangerZone struct {
	AllowLoopback bool `json:"allow_loopback,omitempty"`
}

// Defaults provides a config with sane defaults whenever possible.
func Defaults() C {
	return C{
		AutoSubmitInterval: duration.T(time.Minute * 5),
		Timeout:            duration.T(time.Second * 5),
		BufSize:            4096,
		Contracts:          map[texturl.URL]*relayentry.T{},
		AutoUpgrade:        true,
	}
}

// Validate validates the config. It can change between wireleap-relay releases.
func (c *C) Validate() error {
	if c.Address == nil {
		return errors.New("'address' has to be set")
	}

	if len(c.Contracts) == 0 {
		return errors.New("'contracts' have to be set")
	}

	seen := false
	if len(c.Contracts) > 1 {
		for _, c := range c.Contracts {
			if c.UpgradeChannel != "" {
				if seen {
					return errors.New("only 1 contract in 'contracts' can have an 'upgrade_channel' field set")
				} else {
					seen = true
				}
			}
		}
		if !seen {
			return errors.New("no contract in 'contracts' has an 'upgrade_channel' field set to watch for upgrades")
		}
	}

	for k, v := range c.Contracts {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("enrollment config for %s failed to validate: %w", k.String(), err)
		}
	}

	return nil
}
