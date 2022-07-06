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
	"github.com/wireleap/relay/relaycfg"
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

func UnixServer(path string, fm os.FileMode, t *T) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	l, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	defer l.Close()

	if err := os.Chmod(path, fm); err != nil {
		return err
	}

	h := &http.Server{Handler: t.mux}
	return h.Serve(l)
}

func TCPServer(addr string, t *T) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	h := &http.Server{Handler: t.mux}
	return h.Serve(l)
}

func Run(cfg relaycfg.C, path string, t *T) {
	if len(cfg.RestApi.Address) > 0 {
		go func() {
			log.Println("Launching TCP Server")
			if err := TCPServer(cfg.RestApi.Address, t); err != nil {
				log.Print(err)
			}
		}()
	}

	if cfg.RestApi.Socket {
		go func() {
			log.Println("Launching UnixSocket Server")
			if err := UnixServer(path, cfg.RestApi.Umask, t); err != nil {
				log.Print(err)
			}
		}()
	}
}
