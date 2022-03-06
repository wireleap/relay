// Copyright (c) 2022 Wireleap

package contractmanager

import (
	"github.com/blang/semver"

	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/status"
	"github.com/wireleap/common/cli/fsdir"
	"github.com/wireleap/common/cli/upgrade"
	"github.com/wireleap/relay/api/synccounters"
	"github.com/wireleap/relay/filenames"
	"github.com/wireleap/relay/relaycfg"
	"github.com/wireleap/relay/relaylib"
	"github.com/wireleap/relay/relaystats"
	"github.com/wireleap/relay/relaystats/nustore"
	"github.com/wireleap/relay/version"

	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"
)

const (
	netCapSoftLimit = 0.9
	netCapHardLimit = 0.93
)

const (
	okCap = iota
	softCap
	hardCap
)

var ErrMissingConf = errors.New("missing configuration: wireleap:// listening is not enabled")

// Network usage config holder
type netStatsCfg struct {
	archiveDir    *string
	writeInterval time.Duration
	timeframe     time.Duration
}

// Load clean network usage config from file
func loadNSCfg(c *relaycfg.C) netStatsCfg {
	writeInterval := time.Minute
	if c.NetUsage.WriteInterval != nil {
		writeInterval = time.Duration(*c.NetUsage.WriteInterval)
	}

	timeframe := time.Duration(c.NetUsage.Timeframe)

	if timeframe != time.Duration(0) {
		if writeInterval == time.Duration(0) {
			log.Println("network usage measurement is disabled, write_interval must be set")
			return netStatsCfg{}
		}

		return netStatsCfg{
			archiveDir:    c.NetUsage.ArchiveDir,
			writeInterval: writeInterval,
			timeframe:     timeframe,
		}
	}

	return netStatsCfg{}
}

// Returns if network usage telemetry is enabled
func (n netStatsCfg) Enabled() bool {
	return n.timeframe != time.Duration(0)
}

// Returns if network usage telemetry archive is enabled
func (n netStatsCfg) Archive() bool {
	return n.archiveDir != nil
}

// Network Caps
type cap struct {
	soft uint64
	hard uint64
}

// Network usage limiter config holder
type netCapsCfg struct {
	contractCaps func() map[string]uint64
	globalCap    uint64
}

// Returns network usage limtter clean config
func newNCCfg() netCapsCfg {
	return netCapsCfg{
		contractCaps: func() map[string]uint64 {
			return map[string]uint64{}
		},
	}
}

// Partially load clean network usage limtter config from file
func (cfg *netCapsCfg) loadNCCfg(c *relaycfg.C) {
	cfg.globalCap = uint64(c.NetUsage.GlobalLimit)
}

// Returns if network usage limiter is enabled
func (n netCapsCfg) Enabled() bool {
	return n.globalCap != 0 || len(n.contractCaps()) > 0
}

// Retuns new cap calculation
func (_ netCapsCfg) capCalc(orig, factor *big.Float) (res uint64) {
	ff := new(big.Float)
	ff.Mul(orig, factor)
	res, _ = ff.Uint64()
	return
}

// Retuns new map cap calculation
func (n netCapsCfg) capMap(caps map[string]*big.Float, factor *big.Float) (res map[string]uint64) {
	res = make(map[string]uint64, len(caps))
	for k, f := range caps {
		res[k] = n.capCalc(f, factor)
	}
	return
}

// Retuns new map and global cap calculations
func (n netCapsCfg) xCap(f float64) (resCaps map[string]uint64, resGlobal uint64) {
	contractCaps := n.contractCaps()

	factor := new(big.Float)
	factor.SetFloat64(f)

	globalCap := new(big.Float)
	globalCap.SetUint64(n.globalCap)

	globalXCap := new(big.Float)
	globalXCap.Mul(globalCap, factor)
	resGlobal, _ = globalXCap.Uint64()

	caps := make(map[string]*big.Float, len(contractCaps))
	for k, u := range contractCaps {
		f := new(big.Float)
		f.SetUint64(u)
		caps[k] = f
	}

	resCaps = n.capMap(caps, factor)
	return
}

// Retuns new map and global soft cap calculations
func (n netCapsCfg) SoftCap() (map[string]uint64, uint64) {
	return n.xCap(netCapSoftLimit)
}

// Retuns new map and global hard cap calculations
func (n netCapsCfg) HardCap() (map[string]uint64, uint64) {
	return n.xCap(netCapHardLimit)
}

// Retuns new map and global soft+hard cap calculations
func (n netCapsCfg) Caps() (resCaps map[string]cap, resGlobal cap) {
	contractCaps := n.contractCaps()

	factorSoft := new(big.Float)
	factorSoft.SetFloat64(netCapSoftLimit)

	factorHard := new(big.Float)
	factorHard.SetFloat64(netCapHardLimit)

	globalCap := new(big.Float)
	globalCap.SetUint64(n.globalCap)

	globalSoftCap := new(big.Float)
	globalSoftCap.Mul(globalCap, factorSoft)
	resGlobal.soft, _ = globalSoftCap.Uint64()

	globalHardCap := new(big.Float)
	globalHardCap.Mul(globalCap, factorHard)
	resGlobal.hard, _ = globalHardCap.Uint64()

	caps := make(map[string]*big.Float, len(contractCaps))
	for k, u := range contractCaps {
		f := new(big.Float)
		f.SetUint64(u)
		caps[k] = f
	}

	resSoftCaps := n.capMap(caps, factorSoft)
	resHardCaps := n.capMap(caps, factorHard)

	resCaps = make(map[string]cap, len(contractCaps))
	for k, _ := range contractCaps {
		soft := resSoftCaps[k]
		hard := resHardCaps[k]
		resCaps[k] = cap{soft, hard}
	}
	return
}

// Network usage
type netStats struct {
	cfg    netStatsCfg
	Active relaystats.NetStats
	legacy map[string]uint64
}

// Returns if network usage telemetry is enabled
func (n netStats) Enabled() bool {
	return n.cfg.Enabled() && n.Active.Enabled()
}

func (n netStats) GetNextReset() (time.Time, bool) {
	cap_duration := time.Duration(n.cfg.timeframe)
	return n.Active.GetNextReset(cap_duration)
}

// Contract Manager functions
type netFns struct {
	lock           *sync.Mutex
	storeStats     func()
	getReachedCaps func() (bool, map[string]int)
	checkStats     func()
	resetStats     func(time.Time)
	nextReset      time.Time
}

// Contract Manager function init
func newFns() netFns {
	return netFns{
		lock: &sync.Mutex{},
	}
}

// Contract Manager Status
type managerStatus struct {
	ControllerStarted bool
	Since             *int64
	Until             *int64
	GlobalCap         *uint64
	GlobalUsage       *uint64
	RelayStatus       []relayStatus
}

// Relay status extended
type relayStatus struct {
	Id       string
	Role     string
	Status   relaylib.RelayFlags
	NetCap   *uint64
	NetUsage uint64
}

// Contract Manager
type Manager struct {
	Controller  *relaylib.Controller
	pubkey      string
	autoupgrade bool
	upgradecfg  *upgrade.Config
	upgradechan chan *status.T
	NetStats    netStats
	netCaps     netCapsCfg
	netFns      netFns
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

	// Load Network usage configuration
	ns := netStats{
		cfg: loadNSCfg(c),
	}

	// Init netCap
	nc := newNCCfg()

	if ns.cfg.Enabled() { // If Network usage is enabled
		ns.Active, ns.legacy, err = loadStats(fm, controller.Contracts())

		// Load NetworkCap Cfg
		nc.loadNCCfg(c)
		nc.contractCaps = controller.NetCap
	}

	m = &Manager{
		Controller:  controller,
		pubkey:      pubkey, // jsonb.PK(pk).String()
		autoupgrade: c.AutoUpgrade,
		upgradecfg:  upgrade.NewConfig(fm, "wireleap-relay", false),
		upgradechan: callback,
		NetStats:    ns,
		netCaps:     nc,
		netFns:      newFns(),
		fm:          fm,
	}
	return
}

func (m *Manager) setNetStats() {
	m.netFns.storeStats = func() {
		m.netFns.lock.Lock()
		defer m.netFns.lock.Unlock()

		if fns, err := saveStats(m.NetStats.Active, m.Controller.Contracts(), m.NetStats.legacy); err != nil {
			log.Print(err)
		} else if errS := m.fm.Set(fns, filenames.Stats); errS != nil {
			log.Fatalf("could not store network usage file: %s", errS)
		}
	}
}

func (m *Manager) setReachedCaps() {
	m.netFns.getReachedCaps = func() (globalCap bool, reachedCaps map[string]int) {

		// gather external data sources
		contracts := m.Controller.Contracts()
		caps, globalXCap := m.netCaps.Caps()

		// initialise global counter
		sum := uint64(0)

		// Init result map
		reachedCaps = make(map[string]int, len(contracts))
		for _, ct := range contracts {
			reachedCaps[ct] = okCap
		}

		m.netFns.lock.Lock()
		defer m.netFns.lock.Unlock()

		f := func(contract string, contractBytes *synccounters.ContractCounter) bool {
			if contractBytes == nil {
				log.Printf("Contract metric %s returned nil value", contract)
				// To check, maybe it's better to abort returning false
				return true
			}

			i := contractBytes.Sum()

			if m.netCaps.globalCap != 0 {
				sum = sum + i
			}

			if ct_cap, ok := caps[contract]; !ok {
				// pass
			} else if i > ct_cap.hard {
				reachedCaps[contract] = hardCap
			} else if i > ct_cap.soft {
				reachedCaps[contract] = softCap
			}
			return true
		}

		m.NetStats.Active.ContractStats.Range(f)

		if m.netCaps.globalCap != 0 {
			globalCap = sum >= globalXCap.hard
		}

		return
	}
}

func (m *Manager) setReachedCapsMock() {
	m.netFns.getReachedCaps = func() (globalCap bool, reachedCaps map[string]int) {
		globalCap = false

		// gather external data sources
		contracts := m.Controller.Contracts()

		// Init result map
		reachedCaps = make(map[string]int, len(contracts))
		for _, ct := range contracts {
			reachedCaps[ct] = okCap
		}

		return
	}
}

func (m *Manager) setCheckStats() {
	m.netFns.checkStats = func() {
		// gather external data sources
		caps, globalXCap := m.netCaps.Caps()

		// initialise global counter
		sum := uint64(0)

		// Init result flags
		globalCap := false
		reachedCaps := map[string]int{}

		m.netFns.lock.Lock()
		f := func(contract string, contractBytes *synccounters.ContractCounter) bool {
			if contractBytes == nil {
				log.Printf("Contract metric %s returned nil value", contract)
				// To check, maybe it's better to abort returning false
				return true
			}

			i := contractBytes.Sum()

			if m.netCaps.globalCap != 0 {
				sum = sum + i

				if sum >= globalXCap.hard {
					globalCap = true
					return false
				}
			}

			if ct_cap, ok := caps[contract]; !ok {
				// pass
			} else if i > ct_cap.hard {
				reachedCaps[contract] = hardCap
			} else if i > ct_cap.soft {
				reachedCaps[contract] = softCap
			}
			return true
		}

		/** result := **/
		m.NetStats.Active.ContractStats.Range(f)
		/** If stopped, (returns false) the global netcap was "certainly" reached**/
		m.netFns.lock.Unlock()

		relaystatus := m.Controller.Status()

		if globalCap {
			// Disenrolling all
			for cid, rs := range relaystatus {
				if rs.Flags.Enrolled {
					if err := m.Controller.Disenroll(cid); err != nil {
						log.Printf("Error while disenrolling, %s", err.Error())
					} else {
						log.Printf("Network Cap: Disenrolling from contract %s", cid)
					}
				}

				if !rs.Flags.NetCapReached {
					if err := m.Controller.Disable(cid); err != nil {
						log.Printf("Error while disabling, %s", err.Error())
					} else {
						log.Printf("Network Cap: Disabling from contract %s", cid)
					}
				}
			}
			return
		} else {
			// Disenrolling relays
			for cid, capType := range reachedCaps {
				rs := relaystatus[cid]
				if rs.Flags.Enrolled {
					if err := m.Controller.Disenroll(cid); err != nil {
						log.Printf("Error while disenrolling, %s", err.Error())
					} else {
						log.Printf("Network Cap: Disenrolling from contract %s", cid)
					}
				}

				if !rs.Flags.NetCapReached && capType == hardCap {
					if err := m.Controller.Disable(cid); err != nil {
						log.Printf("Error while disabling, %s", err.Error())
					} else {
						log.Printf("Network Cap: Disabling from contract %s", cid)
					}
				}
				delete(relaystatus, cid)
			}
		}

		// Enrolling relays
		for cid, rs := range relaystatus {
			//log.Printf("Contract cap on %s:%v was not reached", cid, status)
			if !rs.Flags.Enrolled {
				if err := m.Controller.Enroll(cid); err != nil {
					log.Printf("Error while reenrolling, %s", err.Error())
				} else {
					log.Printf("Network Cap: Reenrolling on contract %s", cid)
				}
			}
		}
	}
}

func (m *Manager) setResetStats() {
	var (
		archive *nustore.T
		err     error
	)

	if !m.NetStats.cfg.Archive() {
		// pass
	} else if archive, err = nustore.New(m.fm.Path(*m.NetStats.cfg.archiveDir)); err != nil {
		log.Fatalf("could not initialize network usage archive: %s", err)
	}

	m.netFns.resetStats = func(t time.Time) {
		m.netFns.lock.Lock()
		defer m.netFns.lock.Unlock()

		// Get relay status
		status := m.Controller.Status()
		cts := make(map[string]bool, len(status))
		for ct, rs := range status {
			cts[ct] = rs.Flags.Enrolled
		}

		since := m.NetStats.Active.CreatedAt
		if r, ok := m.NetStats.Active.ResetWithDate(t); !ok {
			log.Fatalf("could not reset network usage stats")
		} else if archive != nil {
			// Append legacy Stats if not nil, then reset
			if m.NetStats.legacy != nil {
				for ct, b := range m.NetStats.legacy {
					r[ct] = b
				}
				m.NetStats.legacy = nil
			}

			// Create record
			f := nustore.NewArchiveFile(m.pubkey, cts, r, since, m.NetStats.Active.CreatedAt)

			// Store record
			if err := archive.Add(f); err != nil {
				log.Fatalf("could not store network usage archive: %s", err)
			}
		}
	}
}

func (m *Manager) setNetUsageFns() {
	if m.NetStats.Enabled() {
		m.setNetStats()
		m.setResetStats()

		if m.netCaps.Enabled() {
			m.setCheckStats()
			m.setReachedCaps()
		} else {
			m.setReachedCapsMock()
		}
	} else {
		m.setReachedCapsMock()
	}
}

func (m *Manager) unsetNetUsageFns() {
	m.netFns.checkStats = nil
	m.netFns.storeStats = nil
	m.netFns.resetStats = nil
	m.setReachedCapsMock()
}

func (m *Manager) runNetUsageFns() {
	if m.NetStats.Enabled() {
		// Write stats periodically
		go func() {
			for range time.Tick(m.NetStats.cfg.writeInterval) {
				if f := m.netFns.storeStats; f != nil {
					f()
				} else {
					return
				}
			}
		}()

		// Reset stats periodically
		go func() {
			var reset_now bool

			// Reset now if record is too old
			if m.netFns.nextReset, reset_now = m.NetStats.GetNextReset(); reset_now {
				cap_duration := time.Duration(m.NetStats.cfg.timeframe)
				m.netFns.resetStats(m.netFns.nextReset.Add(-cap_duration))
			}

			for {
				<-time.After(m.netFns.nextReset.Sub(time.Now()))
				if f := m.netFns.resetStats; f != nil {
					f(m.netFns.nextReset) // updates netstats.CreatedAt
					m.netFns.nextReset, _ = m.NetStats.GetNextReset()

					// Unleash the netcap
					if f_c := m.netFns.checkStats; f_c != nil {
						f_c()
					}
				} else {
					return
				}
			}
		}()

		if m.netCaps.Enabled() {
			go func() {
				for range time.Tick(10 * time.Second) {
					if f := m.netFns.checkStats; f != nil {
						f()
					} else {
						return
					}
				}
			}()
		}
	}

}

func (m *Manager) upgradeRunloop() {
	for st := range m.upgradechan {
		m.upgradeHandler(st)
	}
}

func (m *Manager) Start() error {
	m.setNetUsageFns()

	m.runNetUsageFns()

	go m.upgradeRunloop()

	// Prepare controller start
	contracts := []string{}
	globalCap, reachedCaps := m.netFns.getReachedCaps()

	if !globalCap {
		for contract, capType := range reachedCaps {
			if capType == okCap {
				contracts = append(contracts, contract)
			}
		}
	}

	return m.Controller.StartWithList(contracts...)
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

	if m.NetStats.Enabled() {
		// Store starts before exiting
		if f := m.netFns.storeStats; f != nil {
			f()
		}
		m.unsetNetUsageFns()
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

	err = m.Controller.Reload(c) // WIP

	// Reload Network usage configuration
	if nsCfg := loadNSCfg(c); m.NetStats.cfg.Enabled() != nsCfg.Enabled() {
		log.Println("please, restart the relay to enable or disable netStats")
	} else {
		m.NetStats.cfg = nsCfg
	}

	if m.NetStats.cfg.Enabled() { // If Network usage is enabled
		// Reload netCap
		nc := newNCCfg()
		nc.loadNCCfg(c)
		nc.contractCaps = m.Controller.NetCap

		if m.netCaps.Enabled() != nc.Enabled() {
			log.Println("please, restart the relay to enable or disable netCap")
		} else if m.netCaps.Enabled() {
			m.netCaps.loadNCCfg(c)
		}
	}

	return nil
}

func (m *Manager) Status() (ms managerStatus) {
	contractCaps := m.netCaps.contractCaps()

	crs := m.Controller.Status()

	netUsage := map[string]uint64{}

	// initialise global counter
	sum := uint64(0)

	if m.NetStats.Enabled() {
		f := func(contract string, contractBytes *synccounters.ContractCounter) bool {
			// 1) Check if nB has been initialised, 2) Check is not null, 3) Copy
			if contractBytes == nil {
				// pass
			} else if nB := contractBytes.Sum(); nB == uint64(0) {
				// pass
			} else {
				sum += nB
				netUsage[contract] = nB
			}
			return true
		}
		m.NetStats.Active.ContractStats.Range(f)
	}

	mrs := make([]relayStatus, 0, len(crs))
	for cid, rs := range crs {
		nu := netUsage[cid]

		var nc *uint64
		if i, ok := contractCaps[cid]; ok {
			nc = &i
		}

		mrs = append(mrs, relayStatus{
			Id:       cid,
			Role:     rs.Role,
			Status:   rs.Flags,
			NetCap:   nc,
			NetUsage: nu,
		})
	}

	ms = managerStatus{
		ControllerStarted: m.Controller.Started(),
		RelayStatus:       mrs,
	}

	if m.NetStats.Enabled() {
		until := int64(m.netFns.nextReset.UnixNano() / 1000000)

		ms.Since = &m.NetStats.Active.CreatedAt
		ms.Until = &until
		ms.GlobalUsage = &sum
		if m.netCaps.Enabled() {
			ms.GlobalCap = &m.netCaps.globalCap
		}
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
