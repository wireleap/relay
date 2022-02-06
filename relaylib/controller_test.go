// Copyright (c) 2022 Wireleap

package relaylib

/**
import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/interfaces/clientrelay"
	"github.com/wireleap/common/api/interfaces/relaycontract"
	"github.com/wireleap/common/api/interfaces/relaydir"
	"github.com/wireleap/common/api/interfaces/relayrelay"
	"github.com/wireleap/common/api/jsonb"
	"github.com/wireleap/common/api/relayentry"
	"github.com/wireleap/common/api/signer"
	"github.com/wireleap/common/api/status"
	"github.com/wireleap/common/api/texturl"
	"github.com/wireleap/contract/contractcfg"
	"github.com/wireleap/dir/dir"

	"github.com/wireleap/relay/api/relayentryext"
	"github.com/wireleap/relay/relaycfg"
	"github.com/wireleap/relay/version"
)

var contractId string

func TestController(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)

	if err != nil {
		t.Fatal(err)
	}

	test_cfg := new(contractcfg.C)
	data, err := ioutil.ReadFile("../testdata/contract/config.json")

	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(data, test_cfg)

	if err != nil {
		t.Fatal(err)
	}

	d := dir.New(1 * time.Minute)
	addr := texturl.URLMustParse("wireleap://localhost:1234")
	saddr := addr.String()

	versions := relayentry.Versions{
		Software:      &version.VERSION,
		ClientRelay:   &clientrelay.T.Version,
		RelayRelay:    &relayrelay.T.Version,
		RelayDir:      &relaydir.T.Version,
		RelayContract: &relaycontract.T.Version,
	}

	re := relayentryext.T{
		T: relayentry.T{
			Role:     "backing",
			Addr:     addr,
			Pubkey:   jsonb.PK(pub),
			Versions: versions,
		},
	}

	dh := testHandler(d, pub, priv, &test_cfg.T)
	cl := client.NewMock(signer.New(priv), dh, relaydir.T)

	cfg := relaycfg.C{
		Contracts: map[texturl.URL]*relayentryext.T{
			*test_cfg.T.Endpoint: &re,
		},
		Address: &saddr,
	}

	callback := make(chan *status.T)
	c := NewController(cl, callback)

	t.Run("TestSetup", func(t *testing.T) {
		// Test setup load
		t.Run("load", func(t *testing.T) {
			err := c.Load(&cfg)

			if err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("TestStart", func(t *testing.T) {
		// Test starting controller
		t.Run("startOk", func(t *testing.T) {
			err := c.Start()

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("startStatus", func(t *testing.T) {
			if !c.Started() {
				t.Fatal("Controller should be started")
			}
		})

		t.Run("startFail", func(t *testing.T) {
			err := c.Start()

			if !errors.Is(err, ErrAlreadyStarted) {
				t.Fatal("Controller already started, should return an error")
			}
		})
	})

	t.Run("TestStatus", func(t *testing.T) {
		// Check if relay is online/offline
		t.Run("statusOnline", func(t *testing.T) {
			status := c.Status()

			var s RelayStatus

			if len(status) != 1 {
				t.Fatal("Relay should be listed")
			} else {
				// contractId is set once, just once.
				for contractId, s = range status {
					if !s.Flags.Enrolled {
						t.Fatal("Relay should be enrolled")
					}
				}
			}
		})

		t.Run("acceptConnection", func(t *testing.T) {
			_, err := c.NewConn(contractId)

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("reloadEnrolled", func(t *testing.T) {
			err := c.update(contractId, &re)

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("disenroll", func(t *testing.T) {
			err := c.Disenroll(contractId)

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("reloadDisenrolled", func(t *testing.T) {
			err := c.update(contractId, &re)

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("statusOffline", func(t *testing.T) {
			status := c.Status()

			if s, ok := status[contractId]; !ok {
				t.Fatal("Relay should be listed")
			} else if s.Flags.Enrolled {
				t.Fatal("Relay should be disenrolled")
			}
		})

		t.Run("denyConnection", func(t *testing.T) {
			_, err := c.NewConn(contractId)

			if !errors.Is(err, ErrContractNotAvailable) {
				t.Fatal(err)
			}
		})

		t.Run("disable", func(t *testing.T) {
			c.Disable(contractId)
		})

		t.Run("statusDisabled", func(t *testing.T) {
			status := c.Status()

			if _, ok := status[contractId]; !ok {
				t.Fatal("Relay should be listed")
			} //else if !s.Flags.NetCapReached {
			//t.Fatal("Relay should be disabled")
			//}
		})

		t.Run("enroll", func(t *testing.T) {
			err := c.Enroll(contractId)

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("statusBackOnline", func(t *testing.T) {
			status := c.Status()

			if s, ok := status[contractId]; !ok {
				t.Fatal("Relay should be listed")
			} else if !s.Flags.Enrolled {
				t.Fatal("Relay should be enrolled")
			} //else if s.Flags.NetCapReached {
			//t.Fatal("Relay should be enabled")
			//}
		})
	})

	t.Run("TestReload", func(t *testing.T) {
		// Test setup reload
		t.Run("reload", func(t *testing.T) {
			err := c.Reload(&cfg)

			if err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("TestMissingContract", func(t *testing.T) {
		//Test wrong contractId
		t.Run("disenroll", func(t *testing.T) {
			err := c.Disenroll("unknownContract")

			if !errors.Is(err, ErrContractNotFound) {
				t.Fatal("Contract should be unknown")
			}
		})

		t.Run("enroll", func(t *testing.T) {
			err := c.Enroll("unknownContract")

			if !errors.Is(err, ErrContractNotFound) {
				t.Fatal("Contract should be unknown")
			}
		})

		t.Run("denyConnection", func(t *testing.T) {
			_, err := c.NewConn("unknownContract")

			if !errors.Is(err, ErrContractNotFound) {
				t.Fatal(err)
			}
		})
	})

	t.Run("TestStatusAll", func(t *testing.T) {
		t.Run("statusOnline", func(t *testing.T) {
			status := c.Status()

			var s RelayStatus

			if len(status) != 1 {
				t.Fatal("Relay should be listed")
			} else {
				for _, s = range status {
					if !s.Flags.Enrolled {
						t.Fatal("Relay should be enrolled")
					}
				}
			}
		})

		t.Run("disenroll", func(t *testing.T) {
			err := c.DisenrollAll()

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("statusOffline", func(t *testing.T) {
			status := c.Status()

			if s, ok := status[contractId]; !ok {
				t.Fatal("Relay should be listed")
			} else if s.Flags.Enrolled {
				t.Fatal("Relay should be disenrolled")
			}
		})

		t.Run("enroll", func(t *testing.T) {
			err := c.EnrollAll()

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("statusBackOnline", func(t *testing.T) {
			status := c.Status()

			if s, ok := status[contractId]; !ok {
				t.Fatal("Relay should be listed")
			} else if !s.Flags.Enrolled {
				t.Fatal("Relay should be enrolled")
			}
		})
	})

	t.Run("TestStop", func(t *testing.T) {
		// Test stopping controller
		t.Run("stopOk", func(t *testing.T) {
			err := c.Stop()

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("stopStatus", func(t *testing.T) {
			if c.Started() {
				t.Fatal("Controller should be stopped")
			}
		})

		t.Run("stopFail", func(t *testing.T) {
			err := c.Stop()

			if !errors.Is(err, ErrNotStarted) {
				t.Fatal("Controller already stoppped, should return an error")
			}
		})

		t.Run("denyConnection", func(t *testing.T) {
			_, err := c.NewConn(contractId)

			if !errors.Is(err, ErrNotStarted) {
				t.Fatal(err)
			}
		})
	})

	t.Run("TestFail", func(t *testing.T) {
		// Test controller commands fail when controller isn't started
		t.Run("enroll", func(t *testing.T) {
			err := c.Enroll(contractId)

			if !errors.Is(err, ErrNotStarted) {
				t.Fatal("Controller should be stopped")
			}
		})

		t.Run("enrollAll", func(t *testing.T) {
			err := c.EnrollAll()

			if !errors.Is(err, ErrNotStarted) {
				t.Fatal("Controller should be stopped")
			}
		})

		t.Run("disenroll", func(t *testing.T) {
			err := c.Disenroll(contractId)

			if !errors.Is(err, ErrNotStarted) {
				t.Fatal("Controller should be stopped")
			}
		})

		t.Run("disenrollAll", func(t *testing.T) {
			err := c.DisenrollAll()

			if !errors.Is(err, ErrNotStarted) {
				t.Fatal("Controller should be stopped")
			}
		})
	})

	t.Run("TestRestart", func(t *testing.T) {
		// Test starting controller again
		t.Run("startOk", func(t *testing.T) {
			err := c.Start()

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("startStatus", func(t *testing.T) {
			if !c.Started() {
				t.Fatal("Controller should be started")
			}
		})

		t.Run("startFail", func(t *testing.T) {
			err := c.Start()

			if !errors.Is(err, ErrAlreadyStarted) {
				t.Fatal("Controller already started, should return an error")
			}
		})

		t.Run("allowConnection", func(t *testing.T) {
			_, err := c.NewConn(contractId)

			if err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("TestStatusGetters", func(t *testing.T) {
		// Test controller status getters
		t.Run("status", func(t *testing.T) {
			status := c.Status()

			if len(status) != 1 {
				t.Fatal("Relay should be listed")
			} else if s, ok := status[contractId]; !ok {
				t.Fatal("Relay should be listed")
			} else if !s.Flags.Enrolled {
				t.Fatal("Relay should be enrolled")
			}
		})

		t.Run("contracts", func(t *testing.T) {
			contracts := c.Contracts()

			if len(contracts) != 1 {
				t.Fatal("Relay should be listed")
			} else if contracts[0] != contractId {
				t.Fatal("Relay should be listed")
			}
		})

		t.Run("scs", func(t *testing.T) {
			scs := c.SCS()

			if len(scs) != 1 {
				t.Fatal("Relay should be listed")
			} else if url, ok := scs[contractId]; !ok {
				t.Fatal("Relay should be listed")
			} else if url != test_cfg.Endpoint.String() {
				t.Fatal("Contract url should match")
			}
		})
	})

	t.Run("TestRemove", func(t *testing.T) {
		// Test removing relay
		t.Run("onlineRemove", func(t *testing.T) {
			err := c.remove(contractId)

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("checkRemoved", func(t *testing.T) {
			status := c.Status()

			if _, ok := status[contractId]; ok {
				t.Fatal("Relay should not be listed")
			}
		})

		t.Run("failRemove", func(t *testing.T) {
			err := c.remove(contractId)

			if !errors.Is(err, ErrContractNotFound) {
				t.Fatal("Relay should be already removed")
			}
		})

		t.Run("failReload", func(t *testing.T) {
			err := c.update(contractId, &re)

			if !errors.Is(err, ErrContractNotFound) {
				t.Fatal("Relay should be already removed")
			}
		})
	})

	t.Run("TestStatusGettersZero", func(t *testing.T) {
		// Test controller status getters wtih no relays
		t.Run("status", func(t *testing.T) {
			status := c.Status()

			if len(status) != 0 {
				t.Fatal("Relay should be listed")
			}
		})

		t.Run("contracts", func(t *testing.T) {
			contracts := c.Contracts()

			if len(contracts) != 0 {
				t.Fatal("Relay shouldn't be listed")
			}
		})

		t.Run("scs", func(t *testing.T) {
			scs := c.SCS()

			if len(scs) != 0 {
				t.Fatal("Relay shouldn't be listed")
			}
		})
	})
}
**/
