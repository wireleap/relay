// Copyright (c) 2022 Wireleap

package startcmd

import (
	"bytes"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/interfaces/relaycontract"
	"github.com/wireleap/common/api/interfaces/relaydir"
	"github.com/wireleap/common/api/jsonb"
	"github.com/wireleap/common/api/sharetoken"
	"github.com/wireleap/common/api/signer"
	"github.com/wireleap/common/cli"
	"github.com/wireleap/common/cli/commonsub/startcmd"
	"github.com/wireleap/common/cli/fsdir"
	"github.com/wireleap/common/ststore"
	"github.com/wireleap/common/wlnet/transport"
	"github.com/wireleap/relay/contractmanager"
	"github.com/wireleap/relay/filenames"
	"github.com/wireleap/relay/relaycfg"
	"github.com/wireleap/relay/relaylib"
	"github.com/wireleap/relay/stscheduler"
	"github.com/wireleap/relay/wlnet/relay"
)

func Cmd() *cli.Subcmd { return startcmd.Cmd("wireleap-relay", serverun) }

func serverun(fm fsdir.T) {
	c := relaycfg.Defaults()
	// try versioned config first
	if err := fm.Get(&c, filenames.Config+".next"); err != nil {
		if err = fm.Get(&c, filenames.Config); err != nil {
			log.Fatalf("could not load config file %s: %s", fm.Path(filenames.Config), err)
		}
	}
	if err := c.Validate(); err != nil {
		log.Fatalf("could not validate config file: %s", err)
	}

	privkey, err := cli.LoadKey(fm, filenames.Seed)

	if err != nil {
		log.Fatal(err)
	}

	cl := client.New(signer.New(privkey), relaydir.T, relaycontract.T)

	// load store
	var sts *ststore.T
	sts, err = ststore.New(fm.Path(filenames.Sharetokens), ststore.RelayKeyFunc)

	if err != nil {
		log.Fatal(err)
	}

	// define netstack sharetoken handler
	pk := privkey.Public().(ed25519.PublicKey)

	// initialise the relay manager
	var manager *contractmanager.Manager
	manager, err = contractmanager.NewManager(fm, &c, jsonb.PK(pk).String(), cl)

	if err != nil {
		log.Fatal(err)
	}

	scs := manager.Controller.SCS()

	creds, err := tls.LoadX509KeyPair(
		fm.Path(filenames.TLSCert),
		fm.Path(filenames.TLSKey),
	)

	if err != nil {
		log.Fatal(err)
	}

	var stc *stscheduler.T

	if time.Duration(c.AutoSubmitInterval).Nanoseconds() > 0 {
		// set up archive if required
		var (
			archive   *ststore.T
			archiveST func(*sharetoken.T) error
		)

		if c.ArchiveDir != nil {
			archive, err = ststore.New(fm.Path(*c.ArchiveDir), ststore.RelayKeyFunc)

			if err != nil {
				log.Fatalf("could not initialize sharetoken archive: %s", err)
			}

			archiveST = func(st *sharetoken.T) error { return archive.Add(st) }
		}

		// configure scheduled submission

		stc = stscheduler.New(time.Duration(c.AutoSubmitInterval), func(st *sharetoken.T) error {
			var (
				pk    = st.Contract.PublicKey.String()
				u, ok = scs[pk]
			)
			if !ok {
				return fmt.Errorf("cannot submit sharetoken to unknown SC %s", pk)
			}
			if err := relaylib.SubmitST(cl, u+"/submit", st); err != nil {
				return err
			}
			// succesfully submitted, can be archived
			// those are not submission failures and can be just logged
			if archiveST != nil {
				if err = archiveST(st); err != nil {
					log.Printf(
						"could not archive sharetoken (sig=%s): %s, keeping it in store",
						st.Signature,
						err,
					)
					return nil
				}
			}
			if err = sts.Del(st); err != nil {
				log.Printf(
					"could not clean up submitted sharetoken (sig=%s): %s, keeping it in store",
					st.Signature,
					err,
				)
			}
			return nil
		})

		// schedule tokens in store for submission on startup
		for _, st := range sts.Filter("", "") {
			stc.Schedule(st)
		}
	}

	verifyST := func(st *sharetoken.T) error {
		if !bytes.Equal(st.RelayPubkey, pk) {
			return fmt.Errorf(
				"sharetoken relay public key mismatch: expecting %s, got %s",
				base64.RawURLEncoding.EncodeToString(pk),
				st.RelayPubkey,
			)
		}

		if st.IsExpiredAt(time.Now().Unix()) {
			return fmt.Errorf("sharetoken is expired")
		}

		_, ok := scs[st.Contract.PublicKey.String()]

		if !ok {
			return fmt.Errorf("this sharetoken was not signed by a trusted service contract")
		}

		return st.Verify()
	}

	handleST := func(st *sharetoken.T) (err error) {
		if err = verifyST(st); err != nil {
			return
		}
		if err = sts.Add(st); err != nil {
			return
		}
		if stc != nil {
			stc.Schedule(st)
		}
		return
	}

	n := transport.New(transport.Options{
		TLSVerify: false,
		Certs:     []tls.Certificate{creds},
		Timeout:   time.Duration(c.Timeout),
	})

	r := relay.New(n, manager, relay.Options{
		MaxTime:       time.Duration(c.MaxTime),
		BufSize:       c.BufSize,
		HandleST:      handleST,
		ErrorOrigin:   jsonb.PK(pk).String(),
		AllowLoopback: c.DangerZone.AllowLoopback,
	})

	// wireleap:// HTTP/2 server
	err = r.ListenAndServeHTTP(*c.Address)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listening for H/2 requests on https://%s", *c.Address)

	// finalizer
	if err := r.Manager.Start(); err != nil {
		// finalizer is valid and needs to run even if there was an error
		r.Manager.Stop()
		log.Fatal(err)
	}

	shutdown := func() bool {
		log.Print("gracefully shutting down...")
		r.Manager.Stop()

		fm.Del(filenames.Pid)
		return true
	}

	defer shutdown()

	// check limit on open files (includes tcp connections)
	var rlim syscall.Rlimit
	if err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim); err == nil {
		if rlim.Cur < 65535 {
			log.Printf(
				"%s %d %s %s",
				"NOTE: current `ulimit -n`/RLIMIT_NOFILE value of",
				rlim.Cur,
				"might be too low for production usage. Consider",
				"increasing it to 65535 via /etc/security/limits.conf.",
			)
		}
	}

	cli.SignalLoop(cli.SignalMap{
		syscall.SIGUSR1: func() (_ bool) {
			log.Println("reloading config")

			sts, err = ststore.New(fm.Path(filenames.Sharetokens), ststore.RelayKeyFunc)

			if err != nil {
				log.Printf("could not reload sharetoken store: %s, keeping old store...", err)
				return
			}

			// reload config file
			c := relaycfg.Defaults()
			if err = fm.Get(&c, filenames.Config); err != nil {
				log.Printf("could not load config file %s: %s", fm.Path(filenames.Config), err)
			} else if err = c.Validate(); err != nil {
				log.Printf("could not validate config file: %s", err)
			} else if err = r.Manager.ReloadCfg(&c); err != nil {
				log.Printf("could not reload relay config: %s", err)
			}

			return
		},
		syscall.SIGUSR2: func() (_ bool) {
			log.Println("current status")
			r.Manager.PrintStatus()
			return
		},
		syscall.SIGINT:  shutdown,
		syscall.SIGTERM: shutdown,
		syscall.SIGQUIT: shutdown,
	})
}
