// Copyright (c) 2021 Wireleap

// The release version is defined here.
package version

import (
	"fmt"
	"log"

	"github.com/blang/semver"
	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/consume"
	"github.com/wireleap/common/api/interfaces/relaycontract"
	"github.com/wireleap/common/api/interfaces/relaydir"
	"github.com/wireleap/common/api/relayentry"
	"github.com/wireleap/common/api/signer"
	"github.com/wireleap/common/api/texturl"
	"github.com/wireleap/common/cli"
	"github.com/wireleap/common/cli/fsdir"
	"github.com/wireleap/common/cli/upgrade"
	"github.com/wireleap/relay/filenames"
	"github.com/wireleap/relay/relaycfg"
)

// old name compat
var GITREV string = "<unset>"

// VERSION_STRING is the current version string, set by the linker via go build
// -X flag.
var VERSION_STRING = GITREV

// VERSION is the semver version struct of VERSION_STRING.
var VERSION = semver.MustParse(VERSION_STRING)

// Post-rollback hook for rollbackcmd.
func PostRollbackHook(f fsdir.T) (err error) {
	// get old binary back up asap, try 3 times
	log.Printf("starting old wireleap-relay...")
	for i := 0; i < 3; i++ {
		if err = cli.RunChild(f.Path("wireleap-relay"), "start"); err == nil {
			// ok
			return nil
		} else {
			err = fmt.Errorf("FAILED to start old wireleap-relay, try %d: %s", i, err)
		}
	}
	// hard fail
	return fmt.Errorf("failed to bring old binary up -- there is no wireleap-relay running! %s", err)
}

// MIGRATIONS is the slice of versioned migrations.
var MIGRATIONS = []*upgrade.Migration{{
	Name:    "upgrade_channel",
	Version: semver.MustParse("0.5.0"),
	Apply: func(f fsdir.T) error {
		c := relaycfg.Defaults()
		if err := f.Get(&c, "config.json.next"); err != nil {
			return fmt.Errorf("could not load config.json.next: %s", err)
		}
		for _, re := range c.Contracts {
			re.UpgradeChannel = re.Channel
			re.Channel = ""
		}
		if err := f.Set(&c, "config.json.next"); err != nil {
			return fmt.Errorf("could not save config.json.next: %s", err)
		}
		return nil
	},
}}

// LatestChannelVersion is a special function for wireleap-relay which will
// obtain the latest version supported by the currently configured update
// channel from the directory.
func LatestChannelVersion(f fsdir.T) (semver.Version, error) {
	c := relaycfg.Defaults()
	// err := fm.Get(&c, filenames.Config)
	if err := f.Get(&c, filenames.Config); err != nil {
		return semver.Version{}, err
	}
	if err := c.Validate(); err != nil {
		return semver.Version{}, err
	}
	privkey, err := cli.LoadKey(f, filenames.Seed)
	if err != nil {
		return semver.Version{}, err
	}
	cl := client.New(signer.New(privkey), relaydir.T, relaycontract.T)
	// NOTE: this depends on there being only 1 contract
	var (
		scurl texturl.URL
		sccfg *relayentry.T
	)
	for k, v := range c.Contracts {
		scurl, sccfg = k, v
		break
	}
	dinfo, err := consume.DirectoryInfo(cl, &scurl)
	if err != nil {
		return semver.Version{}, err
	}
	v, ok := dinfo.UpgradeChannels.Relay[sccfg.UpgradeChannel]
	if !ok {
		return v, fmt.Errorf("no version for channel '%s' is provided by directory", sccfg.UpgradeChannel)
	}
	return v, nil
}
