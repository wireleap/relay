// Copyright (c) 2021 Wireleap

package relaylib

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/blang/semver"
	"github.com/wireleap/common/api/auth"
	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/consume"
	"github.com/wireleap/common/api/contractinfo"
	"github.com/wireleap/common/api/interfaces/relaydir"
	"github.com/wireleap/common/api/jsonb"
	"github.com/wireleap/common/api/status"
	"github.com/wireleap/common/cli/upgrade"
	"github.com/wireleap/common/wlnet"
	"github.com/wireleap/relay/relaycfg"
	"github.com/wireleap/relay/version"
)

func EnrollRelay(c *relaycfg.C, cl *client.Client, u *upgrade.Config) (final func(), err error) {
	tick := time.NewTicker(5 * time.Minute) // heartbeat interval
	requests := []*http.Request{}

	final = func() {
		// catch potential panic
		var r interface{}

		if r = recover(); r != nil {
			log.Printf("captured panic(\"%v\") to disenroll relay", r)
		}

		// stop sending heartbeat
		tick.Stop()

		// disenroll
		for _, req := range requests {
			req.Method = http.MethodDelete
			err = cl.PerformRequestOnce(req, nil)

			if err != nil {
				log.Printf("error while disenrolling from %s: %s", req.URL, err)
			}
		}

		if r != nil {
			log.Println("back to panicking!")
			panic(r)
		}
	}

	for scurl, cfg := range c.Contracts {
		sc := scurl.String()
		err = cfg.Validate()

		if err != nil {
			return
		}

		if c.Address == nil {
			err = fmt.Errorf("not enrolling as %s in %s: %s", cfg.Role, sc, "wireleap:// listening is not enabled")
			return
		}

		d := *cfg
		d.Version = &wlnet.PROTO_VERSION
		d.Pubkey = jsonb.PK(cl.Public())

		var ddata *contractinfo.Directory
		ddata, err = consume.DirectoryData(cl, &scurl)
		if err != nil {
			err = fmt.Errorf("could not get directory data from %s: %w", sc, err)
			return
		}
		ddata.Endpoint.Path = path.Join(ddata.Endpoint.Path, "/relays")
		dirurl := ddata.Endpoint.String()

		var req *http.Request
		req, err = cl.NewRequest(http.MethodPost, dirurl, d)

		if err != nil {
			err = fmt.Errorf("could not create enrollment request for directory %s: %w", dirurl, err)
			return
		}

		auth.SetHeader(req.Header, auth.API, auth.Version, "")
		auth.SetHeader(req.Header, auth.Relay, auth.Version, version.VERSION_STRING)

		if _, err = relaydir.EnrollHandshake(cl, req); err != nil {
			err = fmt.Errorf(
				"could not perform challenge-response proof of work for contract %s: %w",
				sc,
				err,
			)
			return
		}

		requests = append(requests, req)
		log.Printf("Enrolled successfully as %s relay into %s", d.Role, sc)
	}

	// heartbeat thread
	go func() {
		for _ = range tick.C {
			for _, req := range requests {
				st, err := relaydir.EnrollHandshake(cl, req)
				if err != nil {
					log.Printf("could not send heartbeat to directory %s: %s", req.URL, err)
					continue
				}
				if st.Is(status.ErrUpgrade) {
					v1s := st.Cause.Error() // FIXME this is slightly unobvious
					v1, err := semver.Parse(v1s)
					skip := u.SkippedVersion()
					if err == nil && (skip == nil || skip.NE(v1)) {
						if c.AutoUpgrade && upgrade.Supported {
							// attempt upgrade
							log.Printf("Received new update notification for version %s.", v1s)
							if clog, err := u.GetChangelog(v1); err == nil {
								log.Println("Changelog:")
								fmt.Println(clog)
							} else {
								log.Printf("-- error getting changelog: %s --", err)
							}
							log.Printf("Upgrading to version %s...", v1s)
							// upgrade func will attempt rollback in case of failure so no need to do it here
							if err = u.Upgrade(upgrade.ExecutorSupervised, version.VERSION, v1); err != nil {
								log.Printf(
									"Could not upgrade to new wireleap-relay version %s: %s, skipping update.",
									v1s, err,
								)
								if err = u.SkipVersion(v1); err != nil {
									log.Printf(
										"Could not persist skipped version %s: %s",
										v1s, err,
									)
								}
								continue
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
							if err = u.SkipVersion(v1); err != nil {
								log.Printf(
									"Could not persist skipped version %s: %s",
									v1s, err,
								)
							}
						}
					}
				}
			}
		}
	}()

	return
}
