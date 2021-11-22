// Copyright (c) 2021 Wireleap

package relaylib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"sync"

	"github.com/wireleap/common/api/auth"
	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/consume"
	"github.com/wireleap/common/api/contractinfo"
	"github.com/wireleap/common/api/interfaces/clientrelay"
	"github.com/wireleap/common/api/interfaces/relaycontract"
	"github.com/wireleap/common/api/interfaces/relaydir"
	"github.com/wireleap/common/api/interfaces/relayrelay"
	"github.com/wireleap/common/api/jsonb"
	"github.com/wireleap/common/api/status"
	"github.com/wireleap/common/api/texturl"

	"github.com/wireleap/common/api/relayentry"
	"github.com/wireleap/relay/version"
)

var (
	ErrFetchDirUrl = errors.New("could not get directory URL")
	ErrRequest     = errors.New("could not create enrollment request")
	ErrReloadCfg   = errors.New("forbidden change, could not apply new relay config")
)

// relayStatus handles the relay status for a cetain contract
type relayStatus struct {
	Relay  *relayentry.T
	rdUrl  string
	scUrl  string
	lock   *sync.RWMutex
	status RelayFlags
	ctx    ctx_
}

// RelayStatus minified version of relayStatus
type RelayStatus struct {
	Addr  *texturl.URL
	Role  string
	Flags RelayFlags
}

type RelayFlags struct {
	Enrolled bool
	//NetCapReached bool
	/**
		ToDo: More flags to define the current status
		- Failed heartbeats
		- Disabled
	**/
}

type ctx_ struct {
	Context context.Context
	Cancel  context.CancelFunc
}

func (c ctx_) isNil() bool {
	return c.Context == nil
}

func NewRelayStatus(cl *client.Client, scurl texturl.URL, cfg *relayentry.T) (rs relayStatus, err error) {
	sc := scurl.String()
	if err = cfg.Validate(); err != nil {
		return
	}

	versions := relayentry.Versions{
		Software:      &version.VERSION,
		ClientRelay:   &clientrelay.T.Version,
		RelayRelay:    &relayrelay.T.Version,
		RelayDir:      &relaydir.T.Version,
		RelayContract: &relaycontract.T.Version,
	}

	d := &relayentry.T{
		Addr:     cfg.Addr,
		Role:     cfg.Role,
		Key:      cfg.Key,
		Versions: versions,
		Pubkey:   jsonb.PK(cl.Public()),
	}

	dirinfo := &contractinfo.Directory{}
	dirinfo, err = consume.DirectoryData(cl, &scurl)

	if err != nil {
		err = fmt.Errorf("%w for %s: %s", ErrFetchDirUrl, sc, err)
		return
	}

	dirinfo.Endpoint.Path = path.Join(dirinfo.Endpoint.Path, "/relays")
	dirurl := dirinfo.Endpoint.String()

	if _, err = cl.NewRequest(http.MethodPost, dirurl, d); err != nil {
		err = fmt.Errorf("%w for directory %s: %s", ErrRequest, dirurl, err)
		return
	}

	rs = relayStatus{
		Relay: d,
		rdUrl: dirurl,
		scUrl: sc,
		lock:  &sync.RWMutex{},
	}
	return
}

// Reload RelayStatus Configuration
func (rs *relayStatus) Reload(cfg *relayentry.T) (err error) {
	if err = cfg.Validate(); err != nil {
		return
	} else if rs.Relay.Role != cfg.Role {
		err = fmt.Errorf(errTmpl, ErrReloadCfg, rs.rdUrl)
		return
	}

	rs.Relay.Addr = cfg.Addr
	rs.Relay.Key = cfg.Key
	return
}

func newRSRequest(cl *client.Client, method, url string, d *relayentry.T, init bool) (req *http.Request, err error) {
	if req, err = cl.NewRequest(method, url, d); err != nil {
		return
	}

	// Set Headers
	auth.SetHeader(req.Header, auth.Relay, auth.Version, version.VERSION_STRING)

	if !init {
		// Set Headers
		auth.DelHeader(req.Header, relaydir.T.String(), auth.Version)
	}
	return
}

func (rs *relayStatus) request(cl *client.Client, method string, init bool) (*http.Request, error) {
	return newRSRequest(cl, method, rs.rdUrl, rs.Relay, init)
}

func (rs *relayStatus) enroll(cl *client.Client, init bool, errHandler func(*relayStatus, error) error) (st *status.T, err error) {
	var req *http.Request
	req, err = rs.request(cl, http.MethodPost, init)
	if err != nil {
		return nil, err
	}

	st, err = relaydir.EnrollHandshake(cl, req)

	if err == nil {
		// Update relay status
		rs.status.Enrolled = true
		//rs.status.NetCapReached = false

		if rs.ctx.isNil() {
			// Renew context if not initialised
			ctx, cancel := context.WithCancel(context.Background())
			rs.ctx = ctx_{ctx, cancel}
		}
	} else if errHandler != nil {
		err = errHandler(rs, err)
	}
	return
}

func (rs *relayStatus) disenroll(cl *client.Client, errHandler func(*relayStatus, error) error) (err error) {
	var req *http.Request
	req, err = rs.request(cl, http.MethodDelete, false)
	if err != nil {
		return err
	}

	err = cl.PerformRequestOnce(req, nil)

	if err == nil {
		// Update relay status
		rs.status.Enrolled = false
	} else if errHandler != nil {
		err = errHandler(rs, err)
	}
	return
}

// Enroll relay, returns error
func (rs *relayStatus) Enroll(cl *client.Client) (st *status.T, err error) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.enroll(cl, true, enrollErrHandler)
}

// Enroll relay, returns error
func (rs *relayStatus) Beat(cl *client.Client) (st *status.T, err error) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.enroll(cl, false, beatErrHandler)
}

// Disenroll relay, returns error
func (rs *relayStatus) Disenroll(cl *client.Client) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	return rs.disenroll(cl, disenrollErrHandler)
}

// Disenroll relay, returns ok but forces local status
func (rs *relayStatus) ForceDisenroll(cl *client.Client) bool {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	if rs.disenroll(cl, nil) != nil {
		// Force RS status
		rs.status.Enrolled = false
		return false
	}
	return true
}

// Disable relay close internal connections, returns error
func (rs *relayStatus) Disable() {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	// Update relay status
	//rs.status.NetCapReached = true

	// Cancel context
	if rs.ctx.Context != nil {
		rs.ctx.Cancel()
		rs.ctx.Context = nil
	}
	return
}

func (rs *relayStatus) Status() RelayStatus {

	rs.lock.RLock()
	defer rs.lock.RUnlock()
	return RelayStatus{
		Addr:  rs.Relay.Addr,
		Role:  rs.Relay.Role,
		Flags: rs.status,
	}
}

func (rs *relayStatus) Context() context.Context {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	return rs.ctx.Context
}

func enrollErrHandler(rs *relayStatus, err error) error {
	return fmt.Errorf(
		"could not perform challenge-response proof of work for contract %s: %w",
		rs.scUrl,
		err,
	)
}

func beatErrHandler(rs *relayStatus, err error) error {
	log.Printf("could not send heartbeat to directory %s: %s", rs.rdUrl, err)
	return err
}

func disenrollErrHandler(rs *relayStatus, err error) error {
	log.Printf("error while disenrolling from %s: %s", rs.rdUrl, err)
	return err
}
