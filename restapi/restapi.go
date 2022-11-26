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

	"github.com/prometheus/client_golang/prometheus/promhttp"
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

func (t *T) Run(cfg relaycfg.RestApi) {
	if cfg.Address == nil {
		// Not defined, pass
	} else if cfg.Address.Scheme == "http" {
		log.Printf("Launching HTTP Server: %s\n", cfg.Address.Host)
		if err := t.TCPServer(cfg.Address.Host); err != nil {
			log.Print(err)
		}
	} else if cfg.Address.Scheme == "file" {
		os.RemoveAll(cfg.Address.Path)

		log.Printf("Launching UnixSocket Server: %s\n", cfg.Address.Path)
		if err := t.UnixServer(cfg.Address.Path, os.FileMode(cfg.Umask)); err != nil {
			log.Print(err)
		}
	}
}

func (t *T) UnixServer(path string, fm os.FileMode) error {
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

func (t *T) TCPServer(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	h := &http.Server{Handler: t.mux}
	return h.Serve(l)
}

func New(manager *contractmanager.Manager, telemetry_opt bool) (t *T) {
	t = &T{
		manager: manager,
		l:       log.Default(),
		mux:     http.NewServeMux(),
	}

	t.mux.Handle("/api/status", provide.MethodGate(provide.Routes{http.MethodGet: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := t.manager.Status()
		t.reply(w, o)
	})}))

	if telemetry_opt {
		t.mux.Handle("/metrics", promhttp.Handler())
		t.l.Print("Enabling telemetry endpoint")
	}
	return
}
