// Copyright (c) 2022 Wireleap

package relay

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/wireleap/common/api/sharetoken"
	"github.com/wireleap/common/api/status"
	"github.com/wireleap/common/wlnet"
	"github.com/wireleap/common/wlnet/flushwriter"
	"github.com/wireleap/common/wlnet/h2rwc"
	"github.com/wireleap/common/wlnet/transport"
	"github.com/wireleap/relay/api/labels"
	"github.com/wireleap/relay/api/meteredrwc"
	"github.com/wireleap/relay/contractmanager"
	"github.com/wireleap/relay/telemetry"
)

type T struct {
	*transport.T
	Options
	*contractmanager.Manager
}

type Options struct {
	// BufSize is the size in bytes of the send/receive buffers of a relay.
	BufSize int
	// MaxTime is the maximum time for a single connection.
	MaxTime time.Duration
	// HandleST is a generic function which is called on incoming sharetokens.
	HandleST func(*sharetoken.T) error
	// ErrorOrigin is an optional string to use when signaling the origin of
	// errors downstream.
	ErrorOrigin string
	// AllowLoopback sets whether to allow dialing loopback addresses. While
	// useful for testing, it presents a security risk in production.
	AllowLoopback bool
	// GlobalCap sets the ammount of traffic that can be forwarded during
	// a given period
	GlobalCap uint64
	// ContractCap sets the ammount of traffic that can be forwarded during
	// a given period, by contract
	ContractCap map[string]uint64
}

func New(tt *transport.T, m *contractmanager.Manager, o Options) *T {
	return &T{T: tt, Options: o, Manager: m}
}

// isLoopback determines whether the presented address is a loopback interface
// address.
func isLoopback(addr string) bool {
	if addr == "localhost" {
		return true
	}
	ip := net.ParseIP(addr)
	if ip == nil {
		// probably a fqdn
		return false
	}
	// unspecified ips (0.0.0.0/::) can be used to access loopback too
	return ip.IsLoopback() || ip.IsUnspecified()
}

// ServeHTTP is the handler function for H2. It being named ServeHTTP allows
// T to expose the http.Handler interface.
// It handles the initial init payload and brokers the subsequent tunnel
// connections or an exit connection if needed.
func (t *T) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		status.ErrMethod.WriteTo(w)
		return
	}

	h := w.Header()
	h.Set("Trailer", "wl-status")

	var c io.ReadWriteCloser = h2rwc.T{
		Writer:     flushwriter.T{Writer: w},
		ReadCloser: r.Body,
	}

	var ctlabs labels.Contract
	//ctlabs := labels.Contract{Contract: "unknown"}
	defer c.Close()

	origin := t.ErrorOrigin
	p, err := wlnet.InitFromHeaders(r.Header)

	if err != nil {
		/**
		telemetry.Metrics.Conn.Total(ctlabs).Inc()
		telemetry.Metrics.Conn.Open(ctlabs).Inc()
		defer telemetry.Metrics.Conn.Open(ctlabs).Dec()

		telemetry.Metrics.Conn.Error(ctlabs.WithErr("bad request")).Inc()
		**/

		(&status.T{
			Code:   http.StatusBadRequest,
			Desc:   err.Error(),
			Origin: origin,
		}).ToHeader(h)
		return
	}

	if p.Command == "PING" {
		// raw, not in wlnet wire format
		ctlabs = ctlabs.SetContract("ping")
		telemetry.Metrics.Conn.Total(ctlabs).Inc()
		//telemetry.Metrics.Conn.Open(ctlabs).Inc() // Muted
		//defer telemetry.Metrics.Conn.Open(ctlabs).Dec() // Muted

		(&status.T{
			Code:   http.StatusOK,
			Desc:   "PONG",
			Origin: origin,
		}).WriteTo(c)
		return
	}

	contractId := p.Token.Contract.PublicKey.String()
	ctlabs = ctlabs.SetContract(contractId)

	// check if contract is accepted by the relay controller
	var ctx context.Context

	if t.Manager.Controller != nil {
		if role, rErr := t.Manager.Controller.Role(contractId); rErr == nil {
			ctlabs = ctlabs.SetRole(role)
		}

		ctx, err = t.Manager.Controller.NewConn(contractId)

		if err != nil {
			errStr := err.Error()

			telemetry.Metrics.Conn.Total(ctlabs).Inc()
			telemetry.Metrics.Conn.Open(ctlabs).Inc()
			defer telemetry.Metrics.Conn.Open(ctlabs).Dec()

			// ToDo: Telemetry to improve
			telemetry.Metrics.Conn.Error(ctlabs.WithErr(errStr)).Inc()

			(&status.T{
				Code:   http.StatusBadRequest,
				Desc:   errStr,
				Origin: origin,
			}).ToHeader(h)
			return
		}

	} else {
		ctx = context.Background()
	}

	//TCPOpenConns++
	//telemetry.Metrics.MitM.TCPRTT(labels).Since(t)
	telemetry.Metrics.Conn.Total(ctlabs).Inc()
	telemetry.Metrics.Conn.Open(ctlabs).Inc()
	defer telemetry.Metrics.Conn.Open(ctlabs).Dec()

	if t.HandleST != nil {
		err = t.HandleST(p.Token)

		if err != nil {
			// ToDo: Telemetry to improve
			telemetry.Metrics.Conn.Error(ctlabs.WithErr("invalid sharetoken")).Inc()

			(&status.T{
				Code:   http.StatusBadRequest,
				Desc:   err.Error(),
				Origin: origin,
			}).ToHeader(h)
			return
		}
	}

	// signal target errors differently
	if p.Remote.Scheme == "target" {
		origin = "target"
	}

	// no dials to localhost (this relay's host)
	if !t.AllowLoopback && isLoopback(p.Remote.Hostname()) {
		// ToDo: Telemetry to improve
		telemetry.Metrics.Conn.Error(ctlabs.WithErr("loopback access denied")).Inc()

		(&status.T{
			Code: http.StatusBadRequest,
			Desc: fmt.Sprintf(
				"loopback address '%s' requested, refusing to dial",
				p.Remote.Hostname(),
			),
			Origin: origin,
		}).ToHeader(h)
		return
	}

	// hide requested target for privacy
	shown := "(target)"

	// ok to show relays though
	if p.Remote.Scheme != "target" {
		shown = p.Remote.String()
	}

	log.Printf("Dialing %s connection to %s", p.Protocol, shown)
	c2, err := t.T.Transport.DialContext(ctx, p.Protocol, p.Remote.Host)

	if err != nil {
		// TODO more granular errors

		if os.IsTimeout(err) {
			telemetry.Metrics.Conn.Error(ctlabs.WithErr("dialing timeout")).Inc()
			(&status.T{
				Code:   http.StatusRequestTimeout,
				Desc:   err.Error(),
				Origin: origin,
			}).ToHeader(h)
		} else {
			telemetry.Metrics.Conn.Error(ctlabs.WithErr("dialing error")).Inc()
			(&status.T{
				Code:   http.StatusBadGateway,
				Desc:   err.Error(),
				Origin: origin,
			}).ToHeader(h)
		}

		return
	}

	err = t.meteredSplice(ctx, c, c2, ctlabs)

	if err != nil {
		// TODO more granular errors

		if os.IsTimeout(err) {
			telemetry.Metrics.Conn.Error(ctlabs.WithErr("circuit timeout")).Inc()
			(&status.T{
				Code:   http.StatusRequestTimeout,
				Desc:   err.Error(),
				Origin: origin,
			}).ToHeader(h)
		} else {
			(&status.T{
				Code:   http.StatusGone,
				Desc:   err.Error(),
				Origin: origin,
			}).ToHeader(h)
		}
	}
}

func (t *T) monitorRWC(cIn, cOut io.ReadWriteCloser, ctlabs labels.Contract) (io.ReadWriteCloser, io.ReadWriteCloser, func() error) {
	// To extend if other metrics need to be recorded
	syncCounter := t.Manager.NetStats.Active.ContractStats.GetOrInit(ctlabs.Contract)
	in, out := syncCounter.Inner() // Get inner counters

	connlabs := ctlabs.GetConnection()
	lIn, lOut := connlabs.SetOrigin("client"), connlabs.SetOrigin("target")
	return meteredrwc.New(cIn, in, lIn), meteredrwc.New(cOut, out, lOut), syncCounter.Close
}

func (t *T) meteredSplice(ctx context.Context, cIn, cOut io.ReadWriteCloser, ctlabs labels.Contract) error {
	var closeFn func() error
	if t.Manager.NetStats.Enabled() {
		cIn, cOut, closeFn = t.monitorRWC(cIn, cOut, ctlabs)
	}

	defer func() {
		if err := closeFn(); err != nil {
			log.Printf("error happened when closing synccounter %s\n", err.Error())
		}
	}()

	return wlnet.Splice(ctx, cIn, cOut, t.MaxTime, t.BufSize)
}

// ListenAndServeHTTP listens on the specified address and passes the
// connections to ServeHTTP.
func (t *T) ListenAndServeHTTP(addr string) error {
	l, err := tls.Listen("tcp", addr, t.Transport.TLSClientConfig)
	if err != nil {
		return err
	}
	s := http.Server{
		Addr:      addr,
		Handler:   t,
		TLSConfig: t.Transport.TLSClientConfig,
	}
	go s.Serve(l)
	return nil
}
