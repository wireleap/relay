// Copyright (c) 2021 Wireleap

package relaylib

/**
import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
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
	"github.com/wireleap/contract/contractcfg"
	"github.com/wireleap/contract/handlers/info"
	"github.com/wireleap/dir/dir"
	"github.com/wireleap/dir/dirlib"
	"github.com/wireleap/relay/version"
)

func testHandler(d *dir.D, p_k ed25519.PublicKey, d_k ed25519.PrivateKey, d_i *contractinfo.T) http.Handler {
	mux := provide.NewMux(provide.Routes{
		"/relays": dirlib.NewDirector(d, nil, dirinfo.UpgradeChannels{}, d_k, 30, false),
		"/info":   info.New(d_i, p_k),
	})

	return provide.VersionGate(mux, relaydir.T)
}

func TestRelay(t *testing.T) {
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

	t.Run("TestNormal", func(t *testing.T) {
		// fail on any error
		var rs relayStatus
		var url = texturl.URLMustParse("https://wireleap.com")

		t.Run("setup", func(t *testing.T) {
			rs_tmp, err := NewRelayStatus(cl, *url, &re)

			if err != nil {
				t.Fatal(err)
			}

			rs = rs_tmp
		})

		// enroll
		t.Run("enroll", func(t *testing.T) {
			_, err := rs.Enroll(cl)

			if err != nil {
				t.Fatal(err)
			}

			if !rs.Status().Flags.Enrolled {
				t.Fatal("relay must be enrolled")
			}
		})

		// heartbeat
		t.Run("heartbeat", func(t *testing.T) {
			_, err := rs.Beat(cl)

			if err != nil {
				t.Fatal(err)
			}

			if !rs.Status().Flags.Enrolled {
				t.Fatal("relay must be enrolled")
			}
		})

		// disenroll
		t.Run("disenroll", func(t *testing.T) {
			err := rs.Disenroll(cl)

			if err != nil {
				t.Fatal(err)
			}

			if rs.Status().Flags.Enrolled {
				t.Fatal("relay must be disenrolled")
			}
		})
	})

	t.Run("TestForceDisenroll", func(t *testing.T) {
		// fail on any error
		var rs relayStatus
		var url = texturl.URLMustParse("https://wireleap.com")

		t.Run("setup", func(t *testing.T) {
			rs_tmp, err := NewRelayStatus(cl, *url, &re)

			if err != nil {
				t.Fatal(err)
			}

			rs = rs_tmp
		})

		// enroll
		t.Run("enroll", func(t *testing.T) {
			_, err := rs.Enroll(cl)

			if err != nil {
				t.Fatal(err)
			}

			if !rs.Status().Flags.Enrolled {
				t.Fatal("relay must be enrolled")
			}
		})

		// disenroll
		t.Run("forceDisenroll", func(t *testing.T) {
			ok := rs.ForceDisenroll(cl)

			if !ok {
				t.Fatal("could not disenroll")
			}

			if rs.Status().Flags.Enrolled {
				t.Fatal("relay must be disenrolled")
			}
		})
	})

	t.Run("TestReloadConfig", func(t *testing.T) {
		// fail on any error
		var rs relayStatus
		var url = texturl.URLMustParse("https://wireleap.com")

		t.Run("setup", func(t *testing.T) {
			rs_tmp, err := NewRelayStatus(cl, *url, &re)

			if err != nil {
				t.Fatal(err)
			}

			rs = rs_tmp
		})

		// update ok
		t.Run("reloadOK", func(t *testing.T) {

			rx := relayentry.T{
				Role:     "backing",
				Addr:     addr,
				Key:      "key1",
				Pubkey:   jsonb.PK(pub),
				Versions: versions,
			}

			if err := rs.Reload(&rx); err != nil {
				t.Fatal("could not reload config")
			}

			if rs.Relay.Key != "key1" {
				t.Fatal("loaded config isn't correct")
			}
		})

		// wrong protocol
		t.Run("reloadFailOnValidate", func(t *testing.T) {
			addrx := texturl.URLMustParse("ftp://localhost:1234")

			rx := relayentry.T{
				Role:     "backing",
				Addr:     addrx,
				Key:      "key2",
				Pubkey:   jsonb.PK(pub),
				Versions: versions,
			}

			if err := rs.Reload(&rx); err == nil {
				t.Fatal("reload should have failed")
			}

			if rs.Relay.Key == "key2" {
				t.Fatal("config shouldn't have been loaded")
			}
		})

		// wrong role
		t.Run("reloadFailOnValidate", func(t *testing.T) {
			rx := relayentry.T{
				Role:     "fronting",
				Addr:     addr,
				Key:      "key3",
				Pubkey:   jsonb.PK(pub),
				Versions: versions,
			}

			if err := rs.Reload(&rx); !errors.Is(err, ErrReloadCfg) {
				t.Fatal("reload should have failed")
			}

			if rs.Relay.Key == "key3" {
				t.Fatal("config shouldn't have been loaded")
			}
		})
	})

}

func TestErrHandlers(t *testing.T) {
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

	t.Run("TestErrHandlers", func(t *testing.T) {
		// fail on any error
		var rs relayStatus
		var url = texturl.URLMustParse("https://wireleap.com")

		exErr := errors.New("Example error")

		t.Run("setup", func(t *testing.T) {
			rs_tmp, err := NewRelayStatus(cl, *url, &re)

			if err != nil {
				t.Fatal(err)
			}

			rs = rs_tmp
		})

		// enrollErrHandler
		t.Run("TestEnrollErrHandler", func(t *testing.T) {

			err := enrollErrHandler(&rs, exErr)

			if errors.Unwrap(err) != exErr {
				t.Fatal("Must be the same error")
			}
		})

		// beatErrHandler
		t.Run("TestBeatErrHandler", func(t *testing.T) {

			err := beatErrHandler(&rs, exErr)

			if err != exErr {
				t.Fatal("Must be the same error")
			}
		})

		// disenrollErrHandler
		t.Run("TestDisenrollErrHandler", func(t *testing.T) {

			err := disenrollErrHandler(&rs, exErr)

			if err != exErr {
				t.Fatal("Must be the same error")
			}
		})
	})
}
**/
