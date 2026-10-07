package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	selhttp "vec-diputacion-granada/internal/modules/seleccion/adapters/http"
	selapp "vec-diputacion-granada/internal/modules/seleccion/application"
	selports "vec-diputacion-granada/internal/modules/seleccion/ports"
)

// The isolated driver accepts only its private Unix socket. The host bridge
// is separately restricted to this fixed loopback origin; no product routing,
// identity headers, account provisioning or permission publication occurs here.
func serve(c configuration) error {
	if len(c.Operations) != 1 || len(c.Actors) != 1 {
		return errors.New("server_fixture_scope")
	}
	o := c.Operations[0]
	a, ok := c.Actors[o.Actor]
	if !ok {
		return errors.New("server_actor_missing")
	}
	crypto := c.Crypto
	key, ok := c.Keys[selapp.AudienciaConsultaConvocatoriaV3]
	if !ok {
		return errors.New("server_key_missing")
	}
	crypto.KeyID, crypto.KeyVersion, crypto.HMAC, crypto.Issuer = key.ID, key.Version, key.HMAC, key.Issuer
	crypto.GovernmentRevision, crypto.GovernmentSHA, crypto.KeyFrom, crypto.KeyUntil = key.GovernmentRevision, key.GovernmentSHA, key.From, key.Until
	const socket = "/socket/s1-http.sock"
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		return errors.New("server_socket_not_absent")
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return errors.New("server_socket_unavailable")
	}
	defer listener.Close()
	if err = os.Chmod(socket, 0600); err != nil {
		return errors.New("server_socket_mode")
	}
	serial := make(chan struct{}, 1)
	server := &http.Server{ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 10 * time.Second, MaxHeaderBytes: 8192,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			if r.URL.Path != selhttp.RutaFichaConvocatoria || r.Host != "127.0.0.1:18571" || r.Header.Get("Origin") != "http://127.0.0.1:18571" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			select {
			case serial <- struct{}{}:
				defer func() { <-serial }()
			default:
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			err := withRuntime(r.Context(), crypto, a, o, func(reader selports.LectorConvocatoriaExacta, s selports.SolicitudConsultaConvocatoria) error {
				h, err := selhttp.NuevaFichaHandler(selhttp.ConfigFicha{Lector: reader, ResolverContexto: func(*http.Request) (selports.SolicitudConsultaConvocatoria, error) { return s, nil }, ValidarFrontera: func(*http.Request) error { return nil }})
				if err != nil {
					return err
				}
				h.ServeHTTP(w, r)
				return nil
			})
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		})}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		if err := server.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "server_close_failed")
		}
	}()
	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
