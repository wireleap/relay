// Copyright (c) 2022 Wireleap

package relay

import (
	"bytes"
	"crypto/ed25519"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wireleap/common/api/interfaces/clientrelay"
	"github.com/wireleap/common/api/servicekey"
	"github.com/wireleap/common/api/sharetoken"
	"github.com/wireleap/common/api/signer"
	"github.com/wireleap/common/api/texturl"
	"github.com/wireleap/common/wlnet"
	"github.com/wireleap/common/wlnet/transport"
	"github.com/wireleap/relay/contractmanager"
)

func TestWLRelay(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Error(err)
	}
	tt := transport.New(transport.Options{
		TLSVerify: false,
		Timeout:   time.Second * 5,
	})
	rl := New(tt,
		contractmanager.NewDummyManager(), // To improve
		Options{
			MaxTime:       time.Second * 0,
			BufSize:       2048,
			AllowLoopback: true,
		})

	s := signer.New(priv)
	sk := servicekey.New(priv)

	var (
		now    = time.Now()
		sopen  = now.Add(1 * time.Minute)
		sclose = sopen.Add(1 * time.Minute)
	)

	sk.Contract.SettlementOpen = sopen.Unix()
	sk.Contract.SettlementClose = sclose.Unix()
	sk.Contract.Sign(s)

	st, err := sharetoken.New(sk, pub)
	if err != nil {
		t.Fatal(err)
	}
	init := &wlnet.Init{
		Command:  "CONNECT",
		Protocol: "tcp",
		Remote:   texturl.URLMustParse("target://localhost:8888"),
		Token:    st,
		Version:  &clientrelay.T.Version,
	}
	p0 := []byte{'h', 'e', 'l', 'l', 'o', '!', '\r', '\n'}
	pr, pw := io.Pipe()

	r := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(p0))
	for k, v := range init.Headers() {
		r.Header.Set(k, v)
	}
	r.Body = io.NopCloser(pr)
	rw := httptest.NewRecorder()

	// emulate target
	l, err := net.Listen("tcp", "localhost:8888")
	if err != nil {
		t.Fatal(err)
	}
	// emulate relay
	go rl.ServeHTTP(rw, r)
	// emulate client
	con, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	n, err := pw.Write(p0)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(p0) {
		t.Fatal("partial write")
	}
	p1 := make([]byte, 32)
	n, err = con.Read(p1)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(p0) {
		t.Fatal("partial read")
	}
	if !bytes.Equal(p0, p1[:n]) {
		t.Fatal("target received corrupted message:", p0, p1[:n])
	}
	_, err = con.Write(p1[:n])
	if err != nil {
		t.Fatal(err)
	}
	con.Close()
	time.Sleep(500 * time.Millisecond)
	p2 := make([]byte, 32)
	n, err = rw.Result().Body.Read(p2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p0, p2[:n]) {
		t.Fatal("wireleap-relay received corrupted message", p0, p2[:n])
	}
}
