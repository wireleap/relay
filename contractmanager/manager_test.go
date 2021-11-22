// Copyright (c) 2021 Wireleap

package contractmanager

/**
import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/contractinfo"
	"github.com/wireleap/common/api/dirinfo"
	"github.com/wireleap/common/api/interfaces/clientrelay"
	"github.com/wireleap/common/api/interfaces/relaycontract"
	"github.com/wireleap/common/api/interfaces/relaydir"
	"github.com/wireleap/common/api/interfaces/relayrelay"
	"github.com/wireleap/common/api/jsonb"
	"github.com/wireleap/common/api/provide"
	"github.com/wireleap/common/api/relayentry"
	"github.com/wireleap/common/api/signer"
	"github.com/wireleap/common/api/texturl"
	"github.com/wireleap/common/cli/fsdir"
	"github.com/wireleap/contract/contractcfg"
	"github.com/wireleap/contract/handlers/info"
	"github.com/wireleap/dir/dir"
	"github.com/wireleap/dir/dirlib"
	"github.com/wireleap/relay/relaycfg"
	"github.com/wireleap/relay/relaylib"
	"github.com/wireleap/relay/version"
)

var contractId string

func testHandler(d *dir.D, p_k ed25519.PublicKey, d_k ed25519.PrivateKey, d_i *contractinfo.T) http.Handler {
	mux := provide.NewMux(provide.Routes{
		"/relays": dirlib.NewDirector(d, nil, dirinfo.UpgradeChannels{}, d_k, 30, false),
		"/info":   info.New(d_i, p_k),
	})

	return provide.VersionGate(mux, relaydir.T)
}

func tmpFolder(t *testing.T) (string, fsdir.T, error) {
	tmpd, err := ioutil.TempDir("", "wltest.*")

	if err != nil {
		return tmpd, fsdir.T(tmpd), err
	}

	t.Cleanup(func() { os.RemoveAll(tmpd) })

	fm, err := fsdir.New(tmpd)

	return tmpd, fm, err
}

func TestController(t *testing.T) {
	// Disable paralellism
	runtime.GOMAXPROCS(1)

	_, fm, err := tmpFolder(t)

	if err != nil {
		t.Fatal(err)
	}

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

	re := relayentry.T{
		Role:     "backing",
		Addr:     addr,
		Pubkey:   jsonb.PK(pub),
		Versions: versions,
	}

	dh := testHandler(d, pub, priv, &test_cfg.T)
	cl := client.NewMock(signer.New(priv), dh, relaydir.T)

	cfg := relaycfg.C{
		Contracts: map[texturl.URL]*relayentry.T{
			*test_cfg.T.Endpoint: &re,
		},
		Address: &saddr,
	}

	var m *Manager

	t.Run("TestSetup", func(t *testing.T) {
		// Test setup load
		t.Run("load", func(t *testing.T) {
			m, err = NewManager(fm, &cfg, jsonb.PK(pub).String(), cl)

			if err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("TestStart", func(t *testing.T) {
		// Test starting the contract manager
		t.Run("startOk", func(t *testing.T) {
			err := m.Start()

			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("startStatus", func(t *testing.T) {
			if !m.Controller.Started() {
				t.Fatal("Controller should be started")
			}
		})

		t.Run("startFail", func(t *testing.T) {
			err := m.Controller.Start()

			if !errors.Is(err, relaylib.ErrAlreadyStarted) {
				t.Fatal("Controller already started, should return an error")
			}
		})
	})

	t.Run("TestStop", func(t *testing.T) {
		// Test stopping the contract manager
		t.Run("stopOk", func(t *testing.T) {
			m.Stop() // Stop panics on failure
		})
	})

	t.Run("TestStopDup", func(t *testing.T) {
		// Test stopping the contract manager
		t.Run("stopFail", func(t *testing.T) {
			m.Stop() // Stop panics on failure
		})
	})
}
**/
