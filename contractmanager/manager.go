// Copyright (c) 2021 Wireleap

package contractmanager

import (
	"github.com/blang/semver"

	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/status"
	"github.com/wireleap/common/cli/fsdir"
	"github.com/wireleap/common/cli/upgrade"
	"github.com/wireleap/relay/relaycfg"
	"github.com/wireleap/relay/relaylib"
	"github.com/wireleap/relay/version"

	"errors"
	"fmt"
	"log"
	"sync"
)

var ErrMissingConf = errors.New("missing configuration: wireleap:// listening is not enabled")

// Contract Manager Status
type managerStatus struct {
	ControllerStarted bool
	RelayStatus       []relayStatus
}

// Relay status extended
type relayStatus struct {
	Id     string
	Role   string
	Status relaylib.RelayFlags
}

// Contract Manager
type Manager struct {
	Controller  *relaylib.Controller
	pubkey      string
	autoupgrade bool
	upgradecfg  *upgrade.Config
	upgradechan chan *status.T
	fm          fsdir.T
	stopOnce    sync.Once
}

func NewManager(fm fsdir.T, c *relaycfg.C, pubkey string, cl *client.Client) (m *Manager, err error) {
	if c.Address == nil {
		return nil, ErrMissingConf
	}

	callback := make(chan *status.T)
	controller := relaylib.NewController(cl, callback)

	if err = controller.Load(c); err != nil {
		return
	}

	m = &Manager{
		Controller:  controller,
		pubkey:      pubkey, // jsonb.PK(pk).String()
		autoupgrade: c.AutoUpgrade,
		upgradecfg:  upgrade.NewConfig(fm, "wireleap-relay", false),
		upgradechan: callback,
		fm:          fm,
	}
	return
}

func (m *Manager) upgradeRunloop() {
	for st := range m.upgradechan {
		m.upgradeHandler(st)
	}
}

func (m *Manager) Start() error {
	go m.upgradeRunloop()

	return m.Controller.Start()
}

func (m *Manager) Stop() {
	// catch potential panic
	var r interface{}

	if r = recover(); r != nil {
		log.Printf("captured panic(\"%v\") to disenroll relay", r)
	}

	// Disenroll, show errors
	if err := m.Controller.Stop(); err != nil {
		log.Println(err.Error())
	}

	// Close upgrade channel
	m.stopOnce.Do(
		func() {
			close(m.upgradechan)
		},
	)

	if r != nil {
		log.Println("back to panicking!")
		panic(r)
	}
	return
}

func (m *Manager) ReloadCfg(c *relaycfg.C) (err error) {
	if c.Address == nil {
		return ErrMissingConf
	}

	return m.Controller.Reload(c)
}

func (m *Manager) Status() (ms managerStatus) {
	crs := m.Controller.Status()

	mrs := make([]relayStatus, 0, len(crs))
	for cid, rs := range crs {
		mrs = append(mrs, relayStatus{
			Id:     cid,
			Role:   rs.Role,
			Status: rs.Flags,
		})
	}

	ms = managerStatus{
		ControllerStarted: m.Controller.Started(),
		RelayStatus:       mrs,
	}

	return
}

// Handle upgrade response
func (m *Manager) upgradeHandler(st *status.T) {
	v1s := st.Cause.Error() // FIXME this is slightly unobvious
	v1, err := semver.Parse(v1s)
	skip := m.upgradecfg.SkippedVersion()
	if err == nil && (skip == nil || skip.NE(v1)) {
		if m.autoupgrade && upgrade.Supported {
			// attempt upgrade
			log.Printf("Received new update notification for version %s.", v1s)
			if clog, err := m.upgradecfg.GetChangelog(v1); err == nil {
				log.Println("Changelog:")
				fmt.Println(clog)
			} else {
				log.Printf("-- error getting changelog: %s --", err)
			}
			log.Printf("Upgrading to version %s...", v1s)
			// upgrade func will attempt rollback in case of failure so no need to do it here
			if err = m.upgradecfg.Upgrade(upgrade.ExecutorSupervised, version.VERSION, v1); err != nil {
				log.Printf(
					"Could not upgrade to new wireleap-relay version %s: %s, skipping update.",
					v1s, err,
				)
				if err = m.upgradecfg.SkipVersion(v1); err != nil {
					log.Printf(
						"Could not persist skipped version %s: %s",
						v1s, err,
					)
				}
			}
		} else {
			// just log
			log.Printf("There is a new wireleap-relay version available: %s. Since %s, please upgrade manually.", v1, func() string {
				if upgrade.Supported {
					return "'auto_upgrade' is disabled"
				} else {
					return "this binary was not built with upgrade support"
				}
			}())
			if err = m.upgradecfg.SkipVersion(v1); err != nil {
				log.Printf(
					"Could not persist skipped version %s: %s",
					v1s, err,
				)
			}
		}
	}
}
