// Copyright (c) 2021 Wireleap

package relaylib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/status"
	"github.com/wireleap/common/api/texturl"

	"github.com/wireleap/common/api/relayentry"
	"github.com/wireleap/relay/relaycfg"
)

const (
	errNotStarted           = "controller not started"
	errAlreadyStarted       = "controller already started"
	errContractNotFount     = "relay contract not found: %s"
	errContractNotAvailable = "relay contract not available: %s"
	beatInterval            = 5 * time.Minute
)

// Controller is the serverlib relay handler
// Interaction with the relays is always done through the controller
type Controller struct {
	client   *client.Client
	hbt      *time.Ticker
	relays   map[string]*relayStatus
	callback chan *status.T
}

// Create new controller instance
func NewController(cl *client.Client, callback chan *status.T) *Controller {
	return &Controller{
		client:   cl,
		relays:   map[string]*relayStatus{},
		callback: callback,
	}
}

// Add relay to the controller
func (c *Controller) add(scurl texturl.URL, cfg *relayentry.T) (contractId string, err error) {
	if contractId, err = getPK(c.client, scurl); err != nil {
		return
	}

	var rs relayStatus
	if rs, err = NewRelayStatus(c.client, scurl, cfg); err == nil {
		c.relays[contractId] = &rs
	} else {
		contractId = ""
	}
	return
}

// Fresh relay in the controller
func (c *Controller) update(contractId string, cfg *relayentry.T) (err error) {
	if rs, ok := c.relays[contractId]; !ok {
		err = fmt.Errorf(errContractNotFount, contractId)
	} else if err = rs.Reload(cfg); err != nil {
		// pass
	} else if c.hbt != nil && rs.status.Enrolled {
		// update config in directory
		_, err = c.beat(rs)
	}
	return
}

// Remove relay from the controller
func (c *Controller) remove(contractId string) (err error) {
	if rs, ok := c.relays[contractId]; !ok {
		err = fmt.Errorf(errContractNotFount, contractId)
	} else if c.hbt != nil {
		if rs.status.Enrolled {
			// relay only needs to be disenrolled if the controller is started
			err = c.disenroll(rs)
		}

		if err == nil {
			c.disable(rs)
		}
	}

	if err != nil {
		return
	}

	delete(c.relays, contractId)
	return
}

// Enroll specific relay
func (c *Controller) enroll(rs *relayStatus) (err error) {
	_, err = rs.Enroll(c.client)

	if err != nil {
		log.Printf("Enrolled successfully as %s relay into %s", rs.Relay.Role, rs.scUrl)
	}

	return
}

// Heartbeat specific relay
func (c *Controller) beat(rs *relayStatus) (*status.T, error) {
	return rs.Beat(c.client)
}

// Disenroll specific relay
func (c *Controller) disenroll(rs *relayStatus) error {
	return rs.Disenroll(c.client)
}

// Disenroll specific relay, do not return an error
func (c *Controller) forcedisenroll(rs *relayStatus) bool {
	return rs.ForceDisenroll(c.client)
}

// Disable specific relay
func (c *Controller) disable(rs *relayStatus) {
	rs.Disable()
	return
}

// Load current relays configuration
func (c *Controller) Load(scfg *relaycfg.C) (err error) {
	for sc, cfg := range scfg.Contracts {
		if _, err = c.add(sc, cfg); err != nil {
			break
		}
	}
	return
}

// Reload current relays configuration
func (c *Controller) Reload(scfg *relaycfg.C) (err error) {
	// get current relays
	scs := c.SCS()
	// to_remove_list: copy + tune scs map index
	urls := make(map[string]string, len(scs))
	for id, url := range scs {
		urls[url] = id
	}

	// iterate over new list:
	// if also in previous cfg delete from the list
	// if not present add to controller
	for sc, cfg := range scfg.Contracts {
		url := sc.String()
		if id, ok := urls[url]; ok {
			delete(urls, url)

			if err = c.update(id, cfg); err != nil {
				break
			}
		} else if _, err = c.add(sc, cfg); err != nil {
			break
		}
	}

	// Delete remaining relays
	for _, id := range urls {
		if err = c.remove(id); err != nil {
			break
		}
	}
	return
}

// Enroll relay by contractId
func (c *Controller) Enroll(contractId string) error {
	if c.hbt == nil {
		return errors.New(errNotStarted)
	}

	if rs, ok := c.relays[contractId]; ok {
		return c.enroll(rs)
	} else {
		return fmt.Errorf(errContractNotFount, contractId)
	}
}

// Enroll all the relays
func (c *Controller) enrollAll() (err error) {
	for _, rs := range c.relays {
		if rs.Status().Flags.Enrolled {
			// skipping relay already enrolled
		} else if err = c.enroll(rs); err != nil {
			break
		}
	}
	return
}

// Enroll all the relays
func (c *Controller) EnrollAll() error {
	if c.hbt == nil {
		return errors.New(errNotStarted)
	}

	return c.enrollAll()
}

// Disenroll relay by contractId
func (c *Controller) Disenroll(contractid string) error {
	if c.hbt == nil {
		return errors.New(errNotStarted)
	}

	if rs, ok := c.relays[contractid]; ok {
		return c.disenroll(rs)
	} else {
		return fmt.Errorf(errContractNotFount, contractid)
	}
}

// Disenroll all the relays
func (c *Controller) DisenrollAll() (err error) {
	if c.hbt == nil {
		return errors.New(errNotStarted)
	}

	for _, rs := range c.relays {
		if !rs.Status().Flags.Enrolled {
			// skipping relay not enrolled
		} else if err = c.disenroll(rs); err != nil {
			break
		}
	}
	return
}

// Disenroll relay by contractId
func (c *Controller) Disable(contractid string) error {
	if c.hbt == nil {
		return errors.New(errNotStarted)
	}

	if rs, ok := c.relays[contractid]; ok {
		c.disable(rs)
		return nil
	} else {
		return fmt.Errorf(errContractNotFount, contractid)
	}
}

// Controller starter
// Enrolls relays and starts heartbeat goroutine
func (c *Controller) Start() error {
	contractIds := make([]string, 0, len(c.relays))
	for contractId, _ := range c.relays {
		contractIds = append(contractIds, contractId)
	}

	return c.StartWithList(contractIds...)
}

// Controller starter, with custom list
// Enrolls relays and starts heartbeat goroutine
func (c *Controller) StartWithList(contractIds ...string) error {
	if c.hbt != nil {
		return fmt.Errorf(errAlreadyStarted)
	}

	// enroll relays
	for _, contractId := range contractIds {
		if rs, ok := c.relays[contractId]; !ok {
			return fmt.Errorf(errContractNotFount, contractId)
		} else if rs.Status().Flags.Enrolled {
			// skipping relay already enrolled
		} else if err := c.enroll(rs); err != nil {
			return err
		}
	}

	// set heartbeat interval
	hbt := time.NewTicker(beatInterval)
	c.hbt = hbt

	// heartbeat thread
	go func() {
		for _ = range hbt.C {
			for _, rs := range c.relays {
				if !rs.Status().Flags.Enrolled {
					// skip heartbeat if relay is not already enrolled
					continue
				}

				st, err := c.beat(rs)
				if err != nil {
					log.Printf("could not send heartbeat to directory %s: %s", rs.scUrl, err)
					continue
					// if too much errors force disenrollment?
				}
				if st.Is(status.ErrUpgrade) {
					select {
					case c.callback <- st:
						// pass
					default:
						log.Printf("could not send send upgrade callback to the contract manager")
					}
				}
			}
		}
	}()
	return nil
}

// Controller finisher
func (c *Controller) Stop() error {
	// stop sending heartbeat
	if c.hbt != nil {
		c.hbt.Stop()
		c.hbt = nil
	} else {
		return errors.New("controller not started")
	}

	erroredRelays := []string{}

	// disenroll relays
	for contractId, rs := range c.relays {
		if !c.forcedisenroll(rs) {
			erroredRelays = append(erroredRelays, contractId)
		}
	}

	if len(erroredRelays) != 0 {
		return fmt.Errorf("disenrollment patially failed, couldn't disenroll the following contracts: [%s]", strings.Join(erroredRelays, ", "))
	}
	return nil
}

// Returns current Controller status
func (c *Controller) Started() bool {
	return c.hbt != nil
}

// Returns if a new connection should be accepted
// Relays accept new connections meanwhile they're enrolled
func (c *Controller) NewConn(contractId string) (ctx context.Context, err error) {
	if c.hbt == nil {
		err = fmt.Errorf(errNotStarted)
	} else if rs, ok := c.relays[contractId]; !ok {
		err = fmt.Errorf(errContractNotFount, contractId)
	} else if !rs.Status().Flags.Enrolled {
		err = fmt.Errorf(errContractNotAvailable, contractId)
	} else {
		ctx = rs.Context()
	}
	return
}

// Returns current contractIds
func (c *Controller) Contracts() (l []string) {
	l = make([]string, 0, len(c.relays))

	for contractId, _ := range c.relays {
		l = append(l, contractId)
	}
	return
}

// Returns current relays status, by contractId
func (c *Controller) Status() (m map[string]RelayStatus) {
	m = make(map[string]RelayStatus, len(c.relays))

	for contractId, rs := range c.relays {
		m[contractId] = rs.Status()
	}
	return
}

// Returns current relays SC, by contractId
func (c *Controller) SCS() (m map[string]string) {
	m = make(map[string]string, len(c.relays))

	for contractId, rs := range c.relays {
		m[contractId] = rs.scUrl
	}
	return
}
