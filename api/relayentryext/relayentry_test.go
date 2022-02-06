// Copyright (c) 2022 Wireleap

package relayentryext

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/wireleap/common/api/jsonb"
	"github.com/wireleap/common/api/relayentry"
	"github.com/wireleap/common/api/texturl"

	"github.com/blang/semver"
)

func TestValidate(t *testing.T) {
	pk, _, err := ed25519.GenerateKey(rand.Reader)

	if err != nil {
		t.Fatal(err)
	}

	v, err := semver.Make("1.0.0")
	if err != nil {
		t.Fatal(err)
	}

	vs := relayentry.Versions{
		Software:      &v,
		ClientRelay:   &v,
		RelayRelay:    &v,
		RelayDir:      &v,
		RelayContract: &v,
	}

	// Should pass
	r := T{
		T: relayentry.T{
			Role:     "fronting",
			Addr:     texturl.URLMustParse("wireleap://wireleap.com"),
			Pubkey:   jsonb.PK(pk),
			Versions: vs,
		},
		NetUsage: datasize.ByteSize(50),
	}

	if err = r.Validate(); err != nil {
		t.Fatal(err)
	}

	// Should fail with invalid URL scheme
	r = T{
		T: relayentry.T{
			Role:     "fronting",
			Addr:     texturl.URLMustParse("gopher://foo.bar"),
			Pubkey:   jsonb.PK(pk),
			Versions: vs,
		},
		NetUsage: datasize.ByteSize(50),
	}

	if err = r.Validate(); err == nil {
		t.Fatal(err)
	}

	// Should fail with invalid relay role
	r = T{
		T: relayentry.T{
			Role:     "foobar",
			Addr:     texturl.URLMustParse("wireleap://wireleap.com"),
			Pubkey:   jsonb.PK(pk),
			Versions: vs,
		},
		NetUsage: datasize.ByteSize(50),
	}

	if err = r.Validate(); err == nil {
		t.Fatal(err)
	}
}
