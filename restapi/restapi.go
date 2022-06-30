// Copyright (c) 2022 Wireleap

package restapi

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/wireleap/common/api/provide"
	"github.com/wireleap/common/api/status"
	"github.com/wireleap/relay/contractmanager"
)

type T struct {
	manager *contractmanager.Manager
	l       *log.Logger
	mux     *http.ServeMux
}

func (t *T) reply(w http.ResponseWriter, x interface{}) {
	b, err := json.Marshal(x)
	if err != nil {
		t.l.Printf("error %s while serving reply", err)
		status.ErrInternal.WriteTo(w) //err to check
		return
	}
	w.Write(b) //err to check
}

func New(manager *contractmanager.Manager) (t *T) {
	t = &T{
		manager: manager,
		l:       log.Default(),
		mux:     http.NewServeMux(),
	}

	t.mux.Handle("/api/status", provide.MethodGate(provide.Routes{http.MethodGet: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := t.manager.Status()
		t.reply(w, o)
	})}))
	return
}

func UnixServer(p string, fm os.FileMode, t *T) error {
	if err := os.RemoveAll(p); err != nil {
		return err
	}

	l, err := net.Listen("unix", p)
	if err != nil {
		return err
	}
	defer l.Close()

	if err := os.Chmod(p, fm); err != nil {
		return err
	}

	h := &http.Server{Handler: t.mux}
	return h.Serve(l)
}

func TCPServer(port string, t *T) error {
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	h := &http.Server{Handler: t.mux}
	return h.Serve(l)
}
