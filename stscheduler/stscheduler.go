// Copyright (c) 2022 Wireleap

package stscheduler

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/wireleap/common/api/sharetoken"
	"github.com/wireleap/relay/api/labels"
	"github.com/wireleap/relay/telemetry"
)

type T struct {
	scheduled map[int64][]*sharetoken.T
	mu        sync.Mutex
	tt        *time.Ticker
}

func New(dur time.Duration, submit func(*sharetoken.T) error) (t *T) {
	t = &T{
		tt:        time.NewTicker(dur),
		scheduled: map[int64][]*sharetoken.T{},
	}

	// regular submission thread
	go func() {
		for range t.tt.C {
			t.mu.Lock()
			n := 0
			now := time.Now()
			for t0, sts := range t.scheduled {
				if t0 <= now.Unix() {
					for _, st := range sts {
						// Telemetry
						ctlabs := labels.ContractErr{Contract: st.Contract.PublicKey.String()}

						if err := submit(st); err != nil {
							ntime := now.Add(dur)
							blurb := ""

							if st.IsExpiredAt(ntime.Unix()) {
								// next attempt will fail
								blurb = "next submission attempt is past submission window! skipping sharetoken"

								// Telemetry
								telemetry.Metrics.ST.Scheduled(ctlabs.GetContract()).Dec()
								ctlabs = ctlabs.SetError("permanent failure")
							} else {
								// try again later
								t.scheduled[ntime.Unix()] = append(t.scheduled[ntime.Unix()], st)
								blurb = fmt.Sprintf("next submission attempt at %s", ntime)

								// Telemetry
								ctlabs = ctlabs.SetError("temporary failure")
							}

							log.Printf(
								"could not submit sharetoken (sig=%s): %s, %s",
								st.Signature,
								err,
								blurb,
							)

							// Telemetry
							telemetry.Metrics.ST.Submitted(ctlabs).Inc()
						} else {
							n++

							// Telemetry
							telemetry.Metrics.ST.Scheduled(ctlabs.GetContract()).Dec()
							telemetry.Metrics.ST.Submitted(ctlabs).Inc()
						}
					}
					// submission of all sts complete or postponed, clean up
					delete(t.scheduled, t0)
				}
			}
			t.mu.Unlock()
		}
	}()
	log.Printf(
		"sharetoken scheduler started, next tick at %s and every %s.",
		time.Now().Add(dur),
		dur,
	)
	return
}

func (t *T) Schedule(st *sharetoken.T) {
	t.mu.Lock()
	when := st.Contract.SettlementOpen + 1
	t.scheduled[when] = append(t.scheduled[when], st)
	t.mu.Unlock()

	log.Printf(
		"scheduled sharetoken (sig=%s) submission for %s",
		st.Signature,
		time.Unix(when, 0),
	)

	// Telemetry
	ctlabs := labels.Contract{Contract: st.Contract.PublicKey.String()}

	telemetry.Metrics.ST.Scheduled(ctlabs).Inc()
}
